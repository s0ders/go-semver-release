// Package parser provides functions to parse a Git repository commit history.
//
// This package is used to compute the semantic version number from a formatted Git repository commit history. To do so,
// it expects the commit history to follow the Conventional Commits specification.
package parser

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/object/commitgraph"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"golang.org/x/sync/errgroup"

	"github.com/s0ders/go-semver-release/v6/internal/appcontext"
	"github.com/s0ders/go-semver-release/v6/internal/branch"
	"github.com/s0ders/go-semver-release/v6/internal/monorepo"
	"github.com/s0ders/go-semver-release/v6/internal/semver"
)

var conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([\w\-.\\\/]+\))?(!)?: ([\w ]+[\s\S]*)`)

type Parser struct {
	ctx *appcontext.AppContext
	mu  sync.Mutex
}

func New(ctx *appcontext.AppContext) *Parser {
	parser := &Parser{ctx: ctx}

	return parser
}

type ComputeNewSemverOutput struct {
	Semver     *semver.Version
	Project    monorepo.Project
	Branch     string
	CommitHash plumbing.Hash
	NewRelease bool
	Error      error
}

type TagInfo struct {
	Semver *semver.Version
	Name   string
	Commit *object.Commit
}

// Run execute a parser on a repository and analyze the given branches and projects contained inside the given
// AppContext.
func (p *Parser) Run(ctx context.Context, repository *git.Repository) ([]ComputeNewSemverOutput, error) {
	var output []ComputeNewSemverOutput

	// This map holds the latest version for each channel.
	// This is to be able to determine lower tier channel versions based of a potential new higher tier version bumped in this run.
	semverMap := make(map[string]*TagInfo)

	index := commitgraph.NewObjectCommitNodeIndex(repository.Storer)

	for _, gitBranch := range p.ctx.Branches {
		branchErr := p.checkoutBranch(repository, gitBranch.Name)

		if len(p.ctx.Projects) == 0 {
			var result ComputeNewSemverOutput
			if branchErr != nil {
				result = createResultWithError(gitBranch.Name, branchErr)
			} else {
				var err error
				result, err = p.ComputeNewSemver(repository, index, monorepo.Project{}, gitBranch, semverMap)
				if err != nil {
					return nil, fmt.Errorf("computing new semver: %w", err)
				}
			}
			output = append(output, result)
		}

		outputBuf := make([]ComputeNewSemverOutput, len(p.ctx.Projects))

		g, _ := errgroup.WithContext(ctx)

		for i, project := range p.ctx.Projects {
			g.Go(func() error {
				var result ComputeNewSemverOutput
				if branchErr != nil {
					result = createResultWithError(gitBranch.Name, branchErr)
				} else {
					var err error
					result, err = p.ComputeNewSemver(repository, index, project, gitBranch, semverMap)
					if err != nil {
						return fmt.Errorf("computing project %q new semver: %w", project.Name, err)
					}
				}
				outputBuf[i] = result
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, fmt.Errorf("parsing monorepository projects: %w", err)
		}

		output = append(output, outputBuf...)
	}

	return output, nil
}

// ComputeNewSemver returns the next, if any, semantic version number from a given Git repository by parsing its commit
// history.
func (p *Parser) ComputeNewSemver(repository *git.Repository, index commitgraph.CommitNodeIndex, project monorepo.Project, branch branch.Branch, semverMap map[string]*TagInfo) (ComputeNewSemverOutput, error) {
	output := ComputeNewSemverOutput{}

	if project.Name != "" {
		output.Project = project
	}

	// fetch latest commit to only check against versions that where release before this commit
	p.mu.Lock()
	head, err := repository.Head()
	if err != nil {
		return output, fmt.Errorf("fetching head: %w", err)
	}

	latestCommit, err := repository.CommitObject(head.Hash())
	if err != nil {
		return output, fmt.Errorf("fetching last commit: %w", err)
	}

	latestNode, err := index.Get(latestCommit.Hash)
	if err != nil {
		return output, fmt.Errorf("fetching commit node: %w", err)
	}
	p.mu.Unlock()

	latestTagInfo, err := p.FetchLatestSemverTag(repository, project, branch, latestCommit)
	if err != nil {
		return output, fmt.Errorf("fetching latest semver tag: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// check latest tag info against higher tier channel tag info, if available
	var latestSemver *semver.Version
	currentTagInfo := semverMap[project.Name]
	if currentTagInfo != nil && currentTagInfo.Commit.Committer.When.Compare(latestCommit.Committer.When) < 1 &&
		(latestTagInfo.Semver == nil || semver.Compare(currentTagInfo.Semver, latestTagInfo.Semver) == 1) {

		latestTagInfo = currentTagInfo
		latestSemver = currentTagInfo.Semver.Clone()
	} else {
		latestSemver = latestTagInfo.Semver
	}

	var firstRelease bool
	if latestSemver == nil {
		p.ctx.Logger.Debug().Msg("no previous tag, creating one")
		if !branch.Prerelease {
			latestSemver = &semver.Version{Major: 0, Minor: 1, Patch: 0}
		} else {
			latestSemver = &semver.Version{Major: 0, Minor: 1, Patch: 0, Prerelease: &semver.Prerelease{Name: branch.Name, Build: 1}}
		}
		firstRelease = true
	} else {
		p.ctx.Logger.Debug().Str("tag", latestTagInfo.Name).Msg("latest semver tag found")
	}

	// Create commit history
	history := []*object.Commit{}

	repositoryLogs := commitgraph.NewCommitNodeIterTopoOrder(latestNode, nil, nil)
	err = repositoryLogs.ForEach(func(cn commitgraph.CommitNode) error {
		c, err := cn.Commit()
		if err != nil {
			return err
		}
		if latestTagInfo.Commit != nil && latestTagInfo.Commit.Hash == c.Hash {
			return storer.ErrStop
		}
		history = append(history, c)
		return nil
	})
	repositoryLogs.Close()

	if err != nil {
		return output, fmt.Errorf("traversing logs: %w", err)
	}

	var newRelease bool
	var releaseType string
	var commitHash plumbing.Hash

	for i := len(history) - 1; i >= 0; i-- {
		commit := history[i]
		commitReleaseFound, commitReleaseType, hash, err := p.ProcessCommit(commit, project)
		if err != nil {
			return output, fmt.Errorf("parsing commit history: %w", err)
		}

		if commitReleaseFound {
			newRelease = true
			commitHash = hash
		}

		// If the commit that has just been parsed brings a new release, check if a release was previously found. If so, only keep the "highest" type of release.
		if commitReleaseType != "" && !firstRelease {
			if (releaseType == "") ||
				(releaseType == "minor" && commitReleaseType == "major") ||
				(releaseType == "patch" && (commitReleaseType == "minor" || commitReleaseType == "major")) {
				releaseType = commitReleaseType
			}
		}
	}

	if firstRelease && commitHash.IsZero() {
		commitHash = latestCommit.Hash
	}

	if releaseType != "" {
		if (branch.Prerelease && latestSemver.Prerelease == nil) ||
			(latestSemver.Prerelease != nil && latestSemver.Prerelease.Name != branch.Name) {
			latestSemver.Prerelease = &semver.Prerelease{Name: branch.Name}
		}
		switch releaseType {
		case "major":
			latestSemver.BumpMajor()
		case "minor":
			latestSemver.BumpMinor()
		case "patch":
			latestSemver.BumpPatch()
		}
	} else if firstRelease && !newRelease {
		latestSemver.Major = 0
		latestSemver.Minor = 0
		latestSemver.Patch = 0
	}

	latestSemver.Metadata = p.ctx.BuildMetadataFlag

	output.Semver = latestSemver
	output.Branch = branch.Name
	output.CommitHash = commitHash
	output.NewRelease = newRelease

	if semverMap != nil && newRelease {
		semverMap[project.Name] = &TagInfo{
			Semver: latestSemver,
			Name:   "(inherited)",
			Commit: latestCommit,
		}
	}

	return output, nil
}

// ProcessCommit parse a commit message and bump the latest semantic version accordingly.
func (p *Parser) ProcessCommit(commit *object.Commit, project monorepo.Project) (bool, string, plumbing.Hash, error) {
	if !conventionalCommitRegex.MatchString(commit.Message) {
		return false, "", plumbing.ZeroHash, nil
	}

	if project.Name != "" {
		containsProjectFiles, err := commitContainsProjectFiles(commit, project.Path)
		if err != nil {
			return false, "", plumbing.ZeroHash, fmt.Errorf("checking if commit contains project files: %w", err)
		}
		if !containsProjectFiles {
			return false, "", plumbing.ZeroHash, nil
		}
	}

	match := conventionalCommitRegex.FindStringSubmatch(commit.Message)
	breakingChange := match[3] == "!" || strings.HasPrefix(commit.Message, "BREAKING CHANGE")
	commitType := match[1]

	if breakingChange {
		return true, "major", commit.Hash, nil
	}

	releaseType, ok := p.ctx.Rules.Map[commitType]
	if !ok {
		return false, "", plumbing.ZeroHash, nil
	}

	if releaseType != "patch" && releaseType != "minor" {
		return false, "", plumbing.ZeroHash, fmt.Errorf("unknown release type %q", releaseType)
	}

	return true, releaseType, commit.Hash, nil
}

// FetchLatestSemverTag parses a Git repository to fetch the tag corresponding to the highest semantic version number
// among all tags.
func (p *Parser) FetchLatestSemverTag(repository *git.Repository, project monorepo.Project, branch branch.Branch, latestCommit *object.Commit) (*TagInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tags, err := repository.TagObjects()
	if err != nil {
		return nil, fmt.Errorf("fetching tag objects: %w", err)
	}

	result := &TagInfo{}

	channel := branch.Name
	if !branch.Prerelease {
		channel = ""
	}

	err = tags.ForEach(func(tag *object.Tag) error {
		tagCommit, err := tag.Commit()
		if err != nil {
			return nil
		}

		if latestCommit != nil && tagCommit.Committer.When.After(latestCommit.Committer.When) {
			return nil
		}

		if !semver.Regex.MatchString(tag.Name) {
			return nil
		}

		if project.Name != "" && !strings.HasPrefix(tag.Name, project.Name+"-") {
			return nil
		}

		currentSemver, err := semver.NewFromString(tag.Name)

		if currentSemver != nil && semver.CompareChannel(currentSemver, channel) == -1 {
			return nil
		}

		if err != nil {
			return fmt.Errorf("converting tag to semver: %w", err)
		}

		if result.Semver == nil || semver.Compare(result.Semver, currentSemver) == -1 {
			result.Semver = currentSemver
			result.Name = tag.Name
			result.Commit = tagCommit
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("looping over tags: %w", err)
	}

	return result, nil
}

// checkoutBranch moves the HEAD pointer of the given repository to the given branch. This function expects the
// repository to be a clone and have a remote to which it will set the branch being checkout to a remote reference to
// the corresponding remote branch.
func (p *Parser) checkoutBranch(repository *git.Repository, branchName string) error {
	remoteBranchRef := plumbing.NewRemoteReferenceName(p.ctx.RemoteNameFlag, branchName)
	_, err := repository.Reference(remoteBranchRef, true)
	if err != nil {
		return fmt.Errorf("remote branch %q not found: %w", remoteBranchRef, err)
	}

	localBranchRef := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewSymbolicReference(localBranchRef, remoteBranchRef)
	err = repository.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("error creating local branch %q: %w", localBranchRef, err)
	}

	// Checkout the new local branch
	w, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("error getting worktree: %w", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: localBranchRef,
		Force:  true,
	})
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("branch %q does not exist: %w", branchName, err)
		}
		return fmt.Errorf("checking out to release branch: %w", err)
	}

	return nil
}

// createResultWithError returns an empty ComputeNewSemverOutput with an error.
func createResultWithError(branch string, err error) ComputeNewSemverOutput {
	return ComputeNewSemverOutput{
		Semver:     &semver.Version{},
		Project:    monorepo.Project{},
		Branch:     branch,
		CommitHash: plumbing.ZeroHash,
		NewRelease: false,
		Error:      err,
	}
}

// commitContainsProjectFiles checks if a given commit changes contain at least one file whose path belongs to the
// given project's path.
func commitContainsProjectFiles(commit *object.Commit, projectPath string) (bool, error) {
	commitTree, err := commit.Tree()
	if err != nil {
		return false, fmt.Errorf("getting commit tree: %w", err)
	}

	var parentTree *object.Tree
	if parent, err := commit.Parent(0); err == nil {
		parentTree, err = parent.Tree()
		if err != nil {
			return false, fmt.Errorf("getting parent tree: %w", err)
		}
	}

	changes, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return false, fmt.Errorf("getting diff tree: %w", err)
	}

	for _, change := range changes {
		dir := filepath.Dir(change.To.Name)
		if strings.HasPrefix(dir, projectPath) {
			return true, nil
		}
	}

	return false, nil
}

func shortenMessage(message string) string {
	if len(message) > 50 {
		return fmt.Sprintf("%s...", message[0:47])
	}

	return message
}

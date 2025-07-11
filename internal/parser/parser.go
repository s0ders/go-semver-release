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
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	Semver         *semver.Version
	Project        monorepo.Project
	Branch         string
	CommitHash     plumbing.Hash
	NewRelease     bool
	LastReleaseTag *object.Tag
	ForceRelease   bool
	NeverReleased  bool
}

// Run execute a parser on a repository and analyze the given branches and projects contained inside the given
// AppContext.
func (p *Parser) Run(ctx context.Context, repository *git.Repository) ([]ComputeNewSemverOutput, error) {
	var output []ComputeNewSemverOutput

	for _, gitBranch := range p.ctx.Branches {
		err := p.checkoutBranch(repository, gitBranch.Name)
		if err != nil {
			return output, fmt.Errorf("checking out to gitBranch %q: %w", gitBranch.Name, err)
		}

		if len(p.ctx.Projects) == 0 {
			computerNewSemverOutput, err := p.ComputeNewSemver(repository, monorepo.Project{}, gitBranch)
			if err != nil {
				return nil, fmt.Errorf("computing new semver: %w", err)
			}
			output = append(output, computerNewSemverOutput)
			continue
		}

		outputBuf := make([]ComputeNewSemverOutput, len(p.ctx.Projects))

		g, _ := errgroup.WithContext(ctx)

		// First pass: Compute a new semantic version for projects which are actually released
		for i, project := range p.ctx.Projects {
			g.Go(func() error {
				if !project.Release {
					outputBuf[i] = ComputeNewSemverOutput{
						NeverReleased: true,
						NewRelease:    false,
						Project:       project,
						Branch:        gitBranch.Name,
					}
					return nil
				}
				result, err := p.ComputeNewSemver(repository, project, gitBranch)
				if err != nil {
					return fmt.Errorf("computing project %q new semver: %w", project.Name, err)
				}
				outputBuf[i] = result
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, fmt.Errorf("parsing monorepository projects (1st pass): %w", err)
		}

		// Second pass: For each not-released project, first get a list of its dependencies, and then check if there
		// are commits on dependent projects since last release. Skip if there was no release at all yet or if the
		// project is not released by configuration override. Run second pass only if there is at least one project with
		// a dependency.
		if slices.ContainsFunc(p.ctx.Projects, func(project monorepo.Project) bool { return len(project.DependsOn) > 0 }) {
			g, _ = errgroup.WithContext(ctx)
			for i, project := range p.ctx.Projects {
				g.Go(func() error {
					if !project.Release || outputBuf[i].NewRelease || outputBuf[i].LastReleaseTag == nil || len(project.DependsOn) == 0 {
						return nil
					}
					var dependencies []monorepo.Project
					p.collectDependencies(project, &dependencies)
					release, err := p.HasDependenciesChanged(repository, *outputBuf[i].LastReleaseTag, dependencies, outputBuf)
					if err != nil {
						return err
					}
					if release {
						outputBuf[i].Semver.BumpMinor()
						outputBuf[i].NewRelease = true
						outputBuf[i].ForceRelease = true
					}
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return nil, fmt.Errorf("parsing monorepository projects (2nd pass): %w", err)
			}
		}

		output = append(output, outputBuf...)
	}

	return output, nil
}

func (p *Parser) HasDependenciesChanged(repository *git.Repository, latestSemverTag object.Tag, dependencies []monorepo.Project, computeNewSemVerOutput []ComputeNewSemverOutput) (bool, error) {

	var (
		history    []*object.Commit
		logOptions git.LogOptions
	)

	p.mu.Lock()
	defer p.mu.Unlock()

	latestSemverTagCommit, err := latestSemverTag.Commit()
	if err != nil {
		return false, fmt.Errorf("fetching latest semver tag commit: %w", err)
	}

	since := latestSemverTagCommit.Committer.When
	logOptions = git.LogOptions{
		Since: &since,
	}

	repositoryLogs, err := repository.Log(&logOptions)
	if err != nil {
		return false, fmt.Errorf("fetching commit history: %w", err)
	}

	// Create commit history
	_ = repositoryLogs.ForEach(func(c *object.Commit) error {
		history = append(history, c)
		return nil
	})

	for _, dependency := range dependencies {
		// First, check if the dependency is regularly released, in this case three is no need to loop through commit history.
		for i := range computeNewSemVerOutput {
			if computeNewSemVerOutput[i].Project.Name == dependency.Name {
				if computeNewSemVerOutput[i].NewRelease {
					return true, nil
				}
			}
		}
		if dependency.Release {
			continue
		}
		// Looping through commit history and check if there are commits for this dependency.
		for _, commit := range history {
			containsProjectFiles, err := commitContainsProjectFiles(commit, dependency.Path)
			if err != nil {
				return false, fmt.Errorf("checking if commit contains project files: %w", err)
			}
			if containsProjectFiles {
				match := conventionalCommitRegex.FindStringSubmatch(commit.Message)
				breakingChange := match[3] == "!" || strings.HasPrefix(commit.Message, "BREAKING CHANGE")
				commitType := match[1]
				_, doAction := p.ctx.Rules.Map[commitType]
				if doAction || breakingChange {
					return true, nil
				}
			}
		}
	}

	// If this is reached, we've checked all dependencies and there is no need to release the project.
	return false, nil
}

// ComputeNewSemver returns the next, if any, semantic version number from a given Git repository by parsing its commit
// history.
func (p *Parser) ComputeNewSemver(repository *git.Repository, project monorepo.Project, branch branch.Branch) (ComputeNewSemverOutput, error) {
	output := ComputeNewSemverOutput{}

	if project.Name != "" {
		output.Project = project
	}

	latestSemverTag, err := p.FetchLatestSemverTag(repository, project)
	if err != nil {
		return output, fmt.Errorf("fetching latest semver tag: %w", err)
	}

	var (
		latestSemver *semver.Version
		history      []*object.Commit
		logOptions   git.LogOptions
	)

	if latestSemverTag == nil {
		p.ctx.Logger.Debug().Msg("no previous tag, creating one")

		latestSemver = &semver.Version{Major: 0, Minor: 0, Patch: 0}
	} else {
		p.ctx.Logger.Debug().Str("tag", latestSemverTag.Name).Msg("latest semver tag found")

		latestSemver, err = semver.NewFromString(latestSemverTag.Name)
		if err != nil {
			return output, fmt.Errorf("building semver from git tag: %w", err)
		}

		p.mu.Lock()
		latestSemverTagCommit, err := latestSemverTag.Commit()
		if err != nil {
			return output, fmt.Errorf("fetching latest semver tag commit: %w", err)
		}
		p.mu.Unlock()

		// Show all commits that are at least one second older than the latest one pointed by SemVer tag
		since := latestSemverTagCommit.Committer.When.Add(time.Second)
		logOptions.Since = &since
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	repositoryLogs, err := repository.Log(&logOptions)
	if err != nil {
		return output, fmt.Errorf("fetching commit history: %w", err)
	}

	// Create commit history
	_ = repositoryLogs.ForEach(func(c *object.Commit) error {
		history = append(history, c)
		return nil
	})

	// Sort commit history from oldest to most recent
	sort.Slice(history, func(i, j int) bool {
		return history[i].Committer.When.Before(history[j].Committer.When)
	})

	var newRelease bool
	var commitHash plumbing.Hash

	for _, commit := range history {
		newReleaseFound, hash, err := p.ProcessCommit(commit, latestSemver, project)
		if err != nil {
			return output, fmt.Errorf("parsing commit history: %w", err)
		}

		if newReleaseFound {
			newRelease = true
			commitHash = hash
		}
	}

	if branch.Prerelease {
		latestSemver.Prerelease = branch.Name
	}

	latestSemver.Metadata = p.ctx.BuildMetadataFlag

	output.Semver = latestSemver
	output.Branch = branch.Name
	output.CommitHash = commitHash
	output.NewRelease = newRelease
	output.NeverReleased = !project.Release
	if latestSemverTag != nil {
		output.LastReleaseTag = latestSemverTag
	}

	return output, nil
}

// ProcessCommit parse a commit message and bump the latest semantic version accordingly.
func (p *Parser) ProcessCommit(commit *object.Commit, latestSemver *semver.Version, project monorepo.Project) (bool, plumbing.Hash, error) {
	if !conventionalCommitRegex.MatchString(commit.Message) {
		return false, plumbing.ZeroHash, nil
	}

	if project.Name != "" {
		containsProjectFiles, err := commitContainsProjectFiles(commit, project.Path)
		if err != nil {
			return false, plumbing.ZeroHash, fmt.Errorf("checking if commit contains project files: %w", err)
		}
		if !containsProjectFiles {
			return false, plumbing.ZeroHash, nil
		}
	}

	match := conventionalCommitRegex.FindStringSubmatch(commit.Message)
	breakingChange := match[3] == "!" || strings.HasPrefix(commit.Message, "BREAKING CHANGE")
	commitType := match[1]

	if breakingChange {
		latestSemver.BumpMajor()
		return true, commit.Hash, nil
	}

	releaseType, ok := p.ctx.Rules.Map[commitType]
	if !ok {
		return false, plumbing.ZeroHash, nil
	}

	switch releaseType {
	case "patch":
		latestSemver.BumpPatch()
	case "minor":
		latestSemver.BumpMinor()
	default:
		return false, plumbing.ZeroHash, fmt.Errorf("unknown release type %q", releaseType)
	}

	return true, commit.Hash, nil
}

// FetchLatestSemverTag parses a Git repository to fetch the tag corresponding to the highest semantic version number
// among all tags.
func (p *Parser) FetchLatestSemverTag(repository *git.Repository, project monorepo.Project) (*object.Tag, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tags, err := repository.TagObjects()
	if err != nil {
		return nil, fmt.Errorf("fetching tag objects: %w", err)
	}

	var (
		latestSemver *semver.Version
		latestTag    *object.Tag
	)

	err = tags.ForEach(func(tag *object.Tag) error {
		if !semver.Regex.MatchString(tag.Name) {
			return nil
		}

		if project.Name != "" && !strings.HasPrefix(tag.Name, project.Name+"-") {
			return nil
		}

		currentSemver, err := semver.NewFromString(tag.Name)
		if err != nil {
			return fmt.Errorf("converting tag to semver: %w", err)
		}

		if latestSemver == nil || semver.Compare(latestSemver, currentSemver) == -1 {
			latestSemver = currentSemver
			latestTag = tag
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("looping over tags: %w", err)
	}

	return latestTag, nil
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

func (p *Parser) collectDependencies(project monorepo.Project, dependencies *[]monorepo.Project) {
	for _, dep := range project.DependsOn {
		for _, p := range p.ctx.Projects {
			if p.Name == dep {
				if !slices.ContainsFunc(*dependencies, func(item monorepo.Project) bool { return item.Name == dep }) {
					*dependencies = append(*dependencies, p)
				}
			}
		}
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

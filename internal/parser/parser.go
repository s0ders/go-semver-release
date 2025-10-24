// Package parser provides functions to parse a Git repository commit history.
//
// This package is used to compute the semantic version number from a formatted Git repository commit history. To do so,
// it expects the commit history to follow the Conventional Commits specification.
package parser

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/s0ders/go-semver-release/v6/internal/appcontext"
	"github.com/s0ders/go-semver-release/v6/internal/branch"
	"github.com/s0ders/go-semver-release/v6/internal/monorepo"
	"github.com/s0ders/go-semver-release/v6/internal/semver"
)

type BumpType int

const (
	BumpNone BumpType = iota
	BumpPrerelease
	BumpPatch
	BumpMinor
	BumpMajor
)

var conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([\w\-.\\\/]+\))?(!)?: ([\w ]+[\s\S]*)`)

type Parser struct {
	ctx *appcontext.AppContext
}

func New(ctx *appcontext.AppContext) *Parser {
	parser := &Parser{ctx: ctx}

	return parser
}

type ComputeNewSemverOutput struct {
	Semver     *semver.Version
	Project    monorepo.Item
	Branch     string
	CommitHash plumbing.Hash
	NewRelease bool
}

// Run executes a parser on a repository and analyzes the given branches and projects contained inside the given
// AppContext.
func (p *Parser) Run(repository *git.Repository) ([]ComputeNewSemverOutput, error) {
	var output []ComputeNewSemverOutput

	for _, gitBranch := range p.ctx.BranchesCfg {
		if len(p.ctx.MonorepositoryCfg) == 0 {
			result, err := p.ComputeNewSemver(repository, monorepo.Item{}, gitBranch)
			if err != nil {
				return nil, fmt.Errorf("computing new semver: %w", err)
			}
			output = append(output, result)
			continue
		}

		for _, project := range p.ctx.MonorepositoryCfg {
			result, err := p.ComputeNewSemver(repository, project, gitBranch)
			if err != nil {
				return nil, fmt.Errorf("computing project %q new semver: %w", project.Name, err)
			}
			output = append(output, result)
		}
	}

	return output, nil
}

// ComputeNewSemver returns the next, if any, semantic version number from a given Git repository by parsing its commit
// history.
func (p *Parser) ComputeNewSemver(repository *git.Repository, project monorepo.Item, gitBranch branch.Item) (ComputeNewSemverOutput, error) {
	output := ComputeNewSemverOutput{}

	// Get branch reference without checking out
	ref, err := p.getBranchReference(repository, gitBranch.Name)
	if err != nil {
		return output, fmt.Errorf("getting reference for branch %q: %w", gitBranch.Name, err)
	}

	if project.Name != "" {
		output.Project = project
	}

	// Build reachable commits once - used for both tag lookup and history
	reachable, err := BuildReachableCommits(repository, ref)
	if err != nil {
		return output, fmt.Errorf("building reachable commits: %w", err)
	}

	latestSemverTag, err := p.FetchLatestSemverTag(repository, project, reachable.HashSet)
	if err != nil {
		return output, fmt.Errorf("fetching latest semver tag: %w", err)
	}

	var (
		latestSemver *semver.Version
		history      []*object.Commit
	)

	if latestSemverTag == nil {
		p.ctx.Logger.Debug().Msg("no previous tag, creating one")

		latestSemver = &semver.Version{Major: 0, Minor: 0, Patch: 0}
		// No tag means all commits are part of history
		history = reachable.Commits
	} else {
		p.ctx.Logger.Debug().Str("tag", latestSemverTag.Name).Msg("latest semver tag found")

		latestSemver, err = semver.NewFromString(latestSemverTag.Name)
		if err != nil {
			return output, fmt.Errorf("building semver from git tag: %w", err)
		}

		latestSemverTagCommit, err := latestSemverTag.Commit()
		if err != nil {
			return output, fmt.Errorf("fetching latest semver tag commit: %w", err)
		}

		// Filter commits that are newer than the tagged commit
		sinceTime := latestSemverTagCommit.Committer.When.Add(time.Second)
		for _, c := range reachable.Commits {
			if c.Committer.When.After(sinceTime) || c.Committer.When.Equal(sinceTime) {
				history = append(history, c)
			}
		}
	}

	// Sort commit history from oldest to most recent
	sort.Slice(history, func(i, j int) bool {
		return history[i].Committer.When.Before(history[j].Committer.When)
	})

	var newRelease bool
	var commitHash plumbing.Hash

	var maxBumpType BumpType

	for _, commit := range history {
		processCommitOutput, err := p.ProcessCommit(commit, project)
		if err != nil {
			return output, fmt.Errorf("parsing commit history: %w", err)
		}

		if processCommitOutput.NewRelease {
			newRelease = true
			commitHash = processCommitOutput.CommitHash
			maxBumpType = max(maxBumpType, processCommitOutput.BumpType)
		}
	}

	// Single bump per release
	// TODO: handle prerelease bump
	switch maxBumpType {
	case BumpMajor:
		latestSemver.BumpMajor()
	case BumpMinor:
		latestSemver.BumpMinor()
	case BumpPatch:
		latestSemver.BumpPatch()
	}

	if gitBranch.Prerelease {
		latestSemver.Prerelease = gitBranch.Name
	}

	latestSemver.Metadata = p.ctx.BuildMetadata

	output.Semver = latestSemver
	output.Branch = gitBranch.Name
	output.CommitHash = commitHash
	output.NewRelease = newRelease

	return output, nil
}

type ProcessCommitOutput struct {
	NewRelease bool
	CommitHash plumbing.Hash
	BumpType   BumpType
}

// ProcessCommit parse a commit message and bump the latest semantic version accordingly.
func (p *Parser) ProcessCommit(commit *object.Commit, project monorepo.Item) (ProcessCommitOutput, error) {
	output := ProcessCommitOutput{
		NewRelease: false,
		CommitHash: plumbing.ZeroHash,
		BumpType:   BumpNone,
	}

	match := conventionalCommitRegex.FindStringSubmatch(commit.Message)
	if match == nil {
		return output, nil
	}

	if project.Name != "" {
		containsProjectFiles, err := commitContainsProjectFiles(commit, project)
		if err != nil {
			return output, fmt.Errorf("checking if commit contains project files: %w", err)
		}
		if !containsProjectFiles {
			return output, nil
		}
	}

	breakingChange := match[3] == "!" || strings.HasPrefix(commit.Message, "BREAKING CHANGE")
	commitType := match[1]

	if breakingChange {
		output.BumpType = BumpMajor
		output.NewRelease = true
		output.CommitHash = commit.Hash

		return output, nil
	}

	releaseType, ok := p.ctx.RulesCfg[commitType]
	if !ok {
		return output, nil
	}

	output.NewRelease = true
	output.CommitHash = commit.Hash

	switch releaseType {
	case "patch":
		output.BumpType = BumpPatch
	case "minor":
		output.BumpType = BumpMinor
	default:
		return output, fmt.Errorf("unknown release type %q", releaseType)
	}

	return output, nil
}

// FetchLatestSemverTag parses a Git repository to fetch the tag corresponding to the highest semantic version number
// among all tags reachable from the given branch. The reachableHashSet should contain all commit hashes reachable
// from the branch reference.
func (p *Parser) FetchLatestSemverTag(repository *git.Repository, project monorepo.Item, reachableHashSet map[plumbing.Hash]struct{}) (*object.Tag, error) {
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

		// Get the commit this tag points to
		tagCommit, err := tag.Commit()
		if err != nil {
			// Tag might point to a tree or blob; skip it
			return nil
		}

		// Check if tag's commit is reachable from the branch
		if _, isReachable := reachableHashSet[tagCommit.Hash]; !isReachable {
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

// ReachableCommits holds the results of walking reachable commits from a branch reference.
type ReachableCommits struct {
	HashSet map[plumbing.Hash]struct{}
	Commits []*object.Commit
}

// BuildReachableCommits walks all commits reachable from the given reference and returns
// both a hash set (for tag reachability checks) and the commit objects (for history processing).
func BuildReachableCommits(repository *git.Repository, fromRef *plumbing.Reference) (*ReachableCommits, error) {
	result := &ReachableCommits{
		HashSet: make(map[plumbing.Hash]struct{}),
	}

	logIter, err := repository.Log(&git.LogOptions{From: fromRef.Hash()})
	if err != nil {
		return nil, fmt.Errorf("creating log iterator: %w", err)
	}

	err = logIter.ForEach(func(c *object.Commit) error {
		result.HashSet[c.Hash] = struct{}{}
		result.Commits = append(result.Commits, c)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}

	return result, nil
}

// getBranchReference attempts to get a reference to a branch, trying remote-tracking branches first,
// then falling back to local branches if the remote reference is not found.
func (p *Parser) getBranchReference(repository *git.Repository, branchName string) (*plumbing.Reference, error) {
	// Try remote-tracking branch first (refs/remotes/origin/branchName)
	remoteBranchRef := plumbing.NewRemoteReferenceName(p.ctx.RemoteName, branchName)
	ref, err := repository.Reference(remoteBranchRef, true)
	if err == nil {
		p.ctx.Logger.Debug().
			Str("branch", branchName).
			Str("ref", remoteBranchRef.String()).
			Msg("using remote-tracking branch reference")
		return ref, nil
	}

	// If remote reference not found, try local branch (refs/heads/branchName)
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		localBranchRef := plumbing.NewBranchReferenceName(branchName)
		ref, err = repository.Reference(localBranchRef, true)
		if err == nil {
			p.ctx.Logger.Debug().
				Str("branch", branchName).
				Str("ref", localBranchRef.String()).
				Msg("using local branch reference")
			return ref, nil
		}

		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, fmt.Errorf("branch %q not found in remote or local references: %w", branchName, err)
		}
	}

	return nil, fmt.Errorf("getting reference for branch %q: %w", branchName, err)
}

// commitContainsProjectFiles checks if a given commit change contains at least one file whose path belongs to the
// given project's path.
func commitContainsProjectFiles(commit *object.Commit, project monorepo.Item) (bool, error) {
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
		// Check both To.Name (added/modified) and From.Name (deleted/renamed)
		if fileMatchesProject(change.To.Name, project) || fileMatchesProject(change.From.Name, project) {
			return true, nil
		}
	}

	return false, nil
}

// fileMatchesProject checks if a file path belongs to any of the project's configured paths.
// Uses path separator-aware matching to avoid false positives (e.g., "api" matching "api-v2").
func fileMatchesProject(filePath string, project monorepo.Item) bool {
	if filePath == "" {
		return false
	}

	// Git always uses forward slashes, regardless of OS
	if project.Path != "" && pathBelongsTo(filePath, project.Path) {
		return true
	}

	for _, projectPath := range project.Paths {
		if pathBelongsTo(filePath, projectPath) {
			return true
		}
	}

	return false
}

// pathBelongsTo checks if filePath is under the given directory path.
// Handles exact matches and ensures proper path boundary matching.
func pathBelongsTo(filePath, dirPath string) bool {
	if !strings.HasPrefix(filePath, dirPath) {
		return false
	}
	// Ensure we match at a path boundary: either exact match or followed by "/"
	// This prevents "api" from matching "api-v2/file.txt"
	return len(filePath) == len(dirPath) || filePath[len(dirPath)] == '/'
}

func shortenMessage(message string) string {
	if len(message) > 50 {
		return message[0:47] + "..."
	}

	return message
}

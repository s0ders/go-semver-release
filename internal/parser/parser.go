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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/s0ders/go-semver-release/v5/internal/monorepo"
	"github.com/s0ders/go-semver-release/v5/internal/rule"
	"github.com/s0ders/go-semver-release/v5/internal/semver"
	"github.com/s0ders/go-semver-release/v5/internal/tag"
)

var conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([\w\-.\\\/]+\))?(!)?: ([\w ]+[\s\S]*)`)

type Parser struct {
	rules                rule.Rules
	tagger               *tag.Tagger
	logger               zerolog.Logger
	releaseBranch        string
	buildMetadata        string
	prereleaseIdentifier string
	prereleaseMode       bool
	projects             []monorepo.Project
	mu                   sync.Mutex
}

type OptionFunc func(*Parser)

func WithProjects(projects []monorepo.Project) OptionFunc {
	return func(p *Parser) {
		p.projects = projects
	}
}

func WithBuildMetadata(metadata string) OptionFunc {
	return func(p *Parser) {
		p.buildMetadata = metadata
	}
}

func New(logger zerolog.Logger, tagger *tag.Tagger, rules rule.Rules, options ...OptionFunc) *Parser {
	parser := &Parser{
		logger: logger,
		tagger: tagger,
		rules:  rules,
	}

	for _, option := range options {
		option(parser)
	}

	return parser
}

type ComputeNewSemverOutput struct {
	Project    monorepo.Project
	Semver     *semver.Version
	CommitHash plumbing.Hash
	NewRelease bool
}

func (p *Parser) SetBranch(branch string) {
	p.releaseBranch = branch
}

func (p *Parser) SetPrerelease(b bool) {
	p.prereleaseMode = b
}

func (p *Parser) SetPrereleaseIdentifier(prereleaseID string) {
	p.prereleaseIdentifier = prereleaseID
}

func (p *Parser) Run(ctx context.Context, repository *git.Repository) ([]ComputeNewSemverOutput, error) {
	output := make([]ComputeNewSemverOutput, len(p.projects))

	err := p.checkoutBranch(repository)
	if err != nil {
		return output, err
	}

	if len(p.projects) == 0 {
		computerNewSemverOutput, err := p.ComputeNewSemver(repository, monorepo.Project{})
		if err != nil {
			return nil, err
		}

		return []ComputeNewSemverOutput{computerNewSemverOutput}, nil
	}

	g, _ := errgroup.WithContext(ctx)

	for i, project := range p.projects {
		g.Go(func() error {
			result, err := p.ComputeNewSemver(repository, project)
			if err != nil {
				return err
			}

			output[i] = result
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return output, nil
}

// ComputeNewSemver returns the next, if any, semantic version number from a given Git repository by parsing its commit
// history.
func (p *Parser) ComputeNewSemver(repository *git.Repository, project monorepo.Project) (ComputeNewSemverOutput, error) {
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
		p.logger.Debug().Msg("no previous tag, creating one")

		latestSemver = &semver.Version{Major: 0, Minor: 0, Patch: 0}
	} else {
		p.logger.Debug().Str("tag", latestSemverTag.Name).Msg("latest semver tag found")

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

		// Show all commit that are at least one second older than the latest one pointed by SemVer tag
		since := latestSemverTagCommit.Committer.When.Add(time.Second)
		logOptions.Since = &since
	}

	p.mu.Lock()
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
	p.mu.Unlock()

	newRelease, commitHash, err := p.ParseHistory(history, latestSemver, project)
	if err != nil {
		return output, fmt.Errorf("parsing commit history: %w", err)
	}

	if p.prereleaseMode {
		latestSemver.Prerelease = p.prereleaseIdentifier
	}

	latestSemver.Metadata = p.buildMetadata

	output.Semver = latestSemver
	output.CommitHash = commitHash
	output.NewRelease = newRelease

	return output, nil
}

// ParseHistory parses a slice of commits and modifies the given semantic version number according to the release rule
// provided.
func (p *Parser) ParseHistory(commits []*object.Commit, latestSemver *semver.Version, project monorepo.Project) (bool, plumbing.Hash, error) {
	newRelease := false
	latestReleaseCommitHash := plumbing.Hash{}
	rulesMap := p.rules.Map

	for _, commit := range commits {
		if !conventionalCommitRegex.MatchString(commit.Message) {
			continue
		}

		if project.Name != "" {
			p.mu.Lock()
			containsProjectFiles, err := commitContainsProjectFiles(commit, project.Path)
			if err != nil {
				return false, latestReleaseCommitHash, fmt.Errorf("checking if commit contains project files: %w", err)
			}
			p.mu.Unlock()

			if !containsProjectFiles {
				continue
			}
		}

		match := conventionalCommitRegex.FindStringSubmatch(commit.Message)
		breakingChange := match[3] == "!" || strings.Contains(match[0], "BREAKING CHANGE")
		commitType := match[1]
		shortHash := commit.Hash.String()[0:7]
		shortMessage := shortenMessage(commit.Message)

		if breakingChange {
			p.logger.Debug().Str("commit-hash", shortHash).Str("commit-message", shortMessage).Msg("breaking change found")
			latestSemver.BumpMajor()
			latestReleaseCommitHash = commit.Hash
			newRelease = true
			continue
		}

		releaseType, ok := rulesMap[commitType]

		if !ok {
			continue
		}

		switch releaseType {
		case "patch":
			latestSemver.BumpPatch()
		case "minor":
			latestSemver.BumpMinor()
		default:
			return false, latestReleaseCommitHash, fmt.Errorf("unknown release type %q", releaseType)
		}

		latestReleaseCommitHash = commit.Hash
		newRelease = true

		p.logger.Debug().Str("commit-hash", shortHash).Str("commit-message", shortMessage).Str("release-type", releaseType).Msg("new release found")
	}

	return newRelease, latestReleaseCommitHash, nil
}

// FetchLatestSemverTag parses a Git repository to fetch the tag corresponding to the highest semantic version number
// among all tags.
func (p *Parser) FetchLatestSemverTag(repository *git.Repository, project monorepo.Project) (*object.Tag, error) {
	p.mu.Lock()

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

		if project.Name != "" {
			matchProjectTagFormat, err := regexp.MatchString(fmt.Sprintf(`^%s\-.*`, project.Name), tag.Name)
			if err != nil {
				return err
			}

			if !matchProjectTagFormat {
				return nil
			}
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

	p.mu.Unlock()
	return latestTag, nil
}

func (p *Parser) checkoutBranch(repository *git.Repository) error {
	worktree, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("fetching worktree: %w", err)
	}

	if worktree == nil {
		return fmt.Errorf("no worktree, check that repository is initialized")
	}

	// Checkout to release branch
	releaseBranchRef := plumbing.NewBranchReferenceName(p.releaseBranch)
	branchCheckOutOpts := git.CheckoutOptions{
		Branch: releaseBranchRef,
		Force:  true,
	}

	err = worktree.Checkout(&branchCheckOutOpts)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("branch %q does not exist: %w", p.releaseBranch, err)
		}
		return fmt.Errorf("checking out to release branch: %w", err)
	}

	return nil
}

// commitContainsProjectFiles checks if a given commit changes contain at least one file whose path belongs to the
// given project's path.
func commitContainsProjectFiles(commit *object.Commit, projectPath string) (bool, error) {
	regex, err := regexp.Compile(fmt.Sprintf("^%s", projectPath))
	if err != nil {
		return false, fmt.Errorf("compiling project's path regexp: %w", err)
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return false, fmt.Errorf("getting commit tree: %w", err)
	}

	parentCommit := commit.Parents()
	parentTree := &object.Tree{}
	if parentCommit != nil {
		parent, err := parentCommit.Next()
		if err == nil {
			parentTree, err = parent.Tree()
			if err != nil {
				return false, fmt.Errorf("getting parent tree: %w", err)
			}
		}
	}

	changes, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return false, fmt.Errorf("getting diff tree: %w", err)
	}

	for _, change := range changes {
		if regex.MatchString(filepath.Dir(change.To.Name)) {
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

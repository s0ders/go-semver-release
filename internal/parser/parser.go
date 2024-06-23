// Package parser provides functions to parse a Git repository commit history.
//
// This package is used to compute the semantic version number from a formatted Git repository commit history. To do so,
// it expects the commit history to follow the Conventional Commits specification.
package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"

	"github.com/s0ders/go-semver-release/v3/internal/rule"
	"github.com/s0ders/go-semver-release/v3/internal/semver"
	"github.com/s0ders/go-semver-release/v3/internal/tag"
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
}

type OptionFunc func(*Parser)

func WithReleaseBranch(branch string) OptionFunc {
	return func(p *Parser) {
		p.releaseBranch = branch
	}
}

func WithBuildMetadata(metadata string) OptionFunc {
	return func(p *Parser) {
		p.buildMetadata = metadata
	}
}

func WithPrereleaseMode(b bool) OptionFunc {
	return func(p *Parser) {
		p.prereleaseMode = b
	}
}

func WithPrereleaseIdentifier(s string) OptionFunc {
	return func(p *Parser) {
		p.prereleaseIdentifier = s
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
	Semver     *semver.Semver
	CommitHash plumbing.Hash
	NewRelease bool
}

// ComputeNewSemver returns the next, if any, semantic version number from a given Git repository by parsing its commit
// history.
func (p *Parser) ComputeNewSemver(repository *git.Repository) (ComputeNewSemverOutput, error) {
	output := ComputeNewSemverOutput{}

	latestSemverTag, err := FetchLatestSemverTag(repository)
	if err != nil {
		return output, fmt.Errorf("fetching latest semver tag: %w", err)
	}

	var (
		latestSemver *semver.Semver
		history      []*object.Commit
		logOptions   git.LogOptions
	)

	if latestSemverTag == nil {
		p.logger.Debug().Msg("no previous tag, creating one")

		latestSemver = &semver.Semver{Major: 0, Minor: 0, Patch: 0}
	} else {
		p.logger.Debug().Str("tag", latestSemverTag.Name).Msg("latest semver tag found")

		latestSemver, err = semver.FromGitTag(latestSemverTag)
		if err != nil {
			return output, fmt.Errorf("building semver from git tag: %w", err)
		}

		latestSemverTagCommit, err := latestSemverTag.Commit()
		if err != nil {
			return output, fmt.Errorf("fetching latest semver tag commit: %w", err)
		}

		// Show all commit that are at least one second older than the latest one pointed by SemVer tag
		since := latestSemverTagCommit.Committer.When.Add(time.Second)
		logOptions.Since = &since
	}

	worktree, err := repository.Worktree()
	if err != nil {
		return output, fmt.Errorf("fetching worktree: %w", err)
	}

	if worktree == nil {
		return output, fmt.Errorf("no worktree, check that repository is initialized")
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
			return output, fmt.Errorf("branch %q does not exist: %w", p.releaseBranch, err)
		}
		return output, fmt.Errorf("checking out to release branch: %w", err)
	}

	repositoryLogs, err := repository.Log(&logOptions)
	if err != nil {
		return output, fmt.Errorf("fetching commit history: %w", err)
	}

	// Create commit history
	_ = repositoryLogs.ForEach(func(c *object.Commit) error {
		history = append(history, c)
		return nil
	})

	// Reverse commit history to go from oldest to newest
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	newRelease, commitHash, err := p.ParseHistory(history, latestSemver)
	if err != nil {
		return output, fmt.Errorf("parsing commit history: %w", err)
	}

	latestSemver.BuildMetadata = p.buildMetadata

	output.Semver = latestSemver
	output.NewRelease = newRelease
	output.CommitHash = commitHash

	return output, nil
}

// ParseHistory parses a slice of commits and modifies the given semantic version number according to the release rule
// provided.
func (p *Parser) ParseHistory(commits []*object.Commit, latestSemver *semver.Semver) (bool, plumbing.Hash, error) {
	newRelease := false
	latestReleaseCommitHash := plumbing.Hash{}
	rulesMap := p.rules.Map

	if p.prereleaseMode {
		latestSemver.Prerelease = p.prereleaseIdentifier
	}

	for _, commit := range commits {
		if !conventionalCommitRegex.MatchString(commit.Message) {
			continue
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
func FetchLatestSemverTag(repository *git.Repository) (*object.Tag, error) {
	tags, err := repository.TagObjects()
	if err != nil {
		return nil, fmt.Errorf("fetching tag objects: %w", err)
	}

	var (
		latestSemver *semver.Semver
		latestTag    *object.Tag
	)

	err = tags.ForEach(func(tag *object.Tag) error {
		if !semver.Regex.MatchString(tag.Name) {
			return nil
		}

		currentSemver, err := semver.FromGitTag(tag)
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

func shortenMessage(message string) string {
	if len(message) > 50 {
		return fmt.Sprintf("%s...", message[0:47])
	}

	return message
}

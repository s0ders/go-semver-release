// Package parser provides functions to parse a Git repository commit history.
//
// This package is used to compute the semantic version number from a formatted Git repository commit history. To do so,
// it expects the commit history to follow the Conventional Commits specification.
package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"

	"github.com/s0ders/go-semver-release/v2/internal/rules"
	"github.com/s0ders/go-semver-release/v2/internal/semver"
	"github.com/s0ders/go-semver-release/v2/internal/tag"
)

var conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([\w\-.\\\/]+\))?(!)?: ([\w ]+[\s\S]*)`)

type Parser struct {
	releaseRules *rules.ReleaseRules
	logger       zerolog.Logger
}

func New(logger zerolog.Logger, releaseRules *rules.ReleaseRules) *Parser {
	return &Parser{
		releaseRules: releaseRules,
		logger:       logger,
	}
}

// ComputeNewSemver returns the next, if any, semantic version number from a given Git repository by parsing its commit
// history.
func (p *Parser) ComputeNewSemver(r *git.Repository) (*semver.Semver, bool, error) {
	latestSemverTag, err := p.fetchLatestSemverTag(r)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch latest semver: %w", err)
	}

	semverFromTag, err := semver.FromGitTag(latestSemverTag)
	if err != nil {
		return nil, false, fmt.Errorf("failed to build semver from git tag: %w", err)
	}

	logOptions := &git.LogOptions{}

	if !semverFromTag.IsZero() {
		logOptions.Since = &latestSemverTag.Tagger.When
	}

	commitHistory, err := r.Log(logOptions)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch commit history: %w", err)
	}

	var history []*object.Commit

	err = commitHistory.ForEach(func(c *object.Commit) error {
		history = append(history, c)
		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to loop over commit history: %w", err)
	}

	// Reverse commit history to go from oldest to most recent
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	newRelease, err := p.ParseHistory(history, semverFromTag)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse commit history: %w", err)
	}

	return semverFromTag, newRelease, nil
}

// ParseHistory parses a slice of commits and modifies the given semantic version number according to the release rules
// provided.
func (p *Parser) ParseHistory(commits []*object.Commit, latestSemver *semver.Semver) (bool, error) {
	newRelease := false
	rulesMap := p.releaseRules.Map()

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
			newRelease = true
			continue
		}

		releaseType, ruleMatch := rulesMap[commitType]

		if !ruleMatch {
			continue
		}

		switch releaseType {
		case "patch":
			latestSemver.BumpPatch()
		case "minor":
			latestSemver.BumpMinor()
		default:
			return false, fmt.Errorf("unknown release type %s", releaseType)
		}

		newRelease = true
		p.logger.Debug().Str("commit-hash", shortHash).Str("commit-message", shortMessage).Str("release-type", releaseType).Msg("new release found")
	}

	return newRelease, nil
}

// fetchLatestSemverTag parses a Git repository to fetch the tag corresponding to the highest semantic version number
// among all tags.
func (p *Parser) fetchLatestSemverTag(repository *git.Repository) (*object.Tag, error) {
	semverRegex := regexp.MustCompile(semver.Regex)

	tags, err := repository.TagObjects()
	if err != nil {
		return nil, err
	}

	var latestSemver *semver.Semver
	var latestTag *object.Tag

	err = tags.ForEach(func(tag *object.Tag) error {
		if !semverRegex.MatchString(tag.Name) {
			return nil
		}

		currentSemver, err := semver.FromGitTag(tag)
		if err != nil {
			return err
		}

		if latestSemver == nil || latestSemver.Precedence(currentSemver) != 1 {
			latestSemver = currentSemver
			latestTag = tag
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to loop over tags: %w", err)
	}

	if latestSemver == nil {
		p.logger.Debug().Msg("no previous tag, creating one")

		head, err := repository.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch head: %w", err)
		}

		return tag.NewFromSemver(semver.Semver{}, head.Hash()), nil
	}

	p.logger.Debug().Str("tag", latestTag.Name).Msg("latest semver tag found")

	return latestTag, nil
}

func shortenMessage(message string) string {
	if len(message) > 50 {
		return fmt.Sprintf("%s...", message[0:47])
	}

	return message
}

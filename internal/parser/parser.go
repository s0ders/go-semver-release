package parser

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/s0ders/go-semver-release/v2/internal/rules"
	"github.com/s0ders/go-semver-release/v2/internal/semver"
	"github.com/s0ders/go-semver-release/v2/internal/tagger"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([\w\-\.\\\/]+\))?(!)?: ([\w ])+([\s\S]*)`)

type Parser struct {
	logger       *slog.Logger
	releaseRules *rules.ReleaseRules
	verbose      bool
}

func New(logger *slog.Logger, releaseRules *rules.ReleaseRules, verbose bool) *Parser {
	return &Parser{
		logger:       logger,
		releaseRules: releaseRules,
		verbose:      verbose,
	}
}

func (p *Parser) fetchLatestSemverTag(repository *git.Repository) (*object.Tag, error) {
	semverRegex := regexp.MustCompile(semver.Regex)

	// TODO: use .Tags() to fetch all kind of tags and not just annotated ones
	tags, err := repository.TagObjects()
	if err != nil {
		return nil, err
	}

	var latestSemver *semver.Semver
	var latestTag *object.Tag

	err = tags.ForEach(func(tag *object.Tag) error {
		if semverRegex.MatchString(tag.Name) {
			currentSemver, err := semver.FromGitTag(tag)
			if err != nil {
				return err
			}

			if latestSemver == nil || latestSemver.Precedence(currentSemver) != 1 {
				latestSemver = currentSemver
				latestTag = tag
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to loop over tags: %w", err)
	}

	if latestSemver == nil {
		if p.verbose {
			p.logger.Info("no previous tag, creating one")
		}

		head, err := repository.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch head: %w", err)
		}

		version, err := semver.New(0, 0, 0, "")
		if err != nil {
			return nil, fmt.Errorf("failed to build new semver: %w", err)
		}

		return tagger.NewTagFromSemver(*version, head.Hash()), nil
	}

	if p.verbose {
		p.logger.Info("found latest semver tag", "tag", latestTag.Name)
	}

	return latestTag, nil
}

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

func (p *Parser) ParseHistory(commits []*object.Commit, latestSemver *semver.Semver) (bool, error) {
	newRelease := false
	newReleaseType := ""
	rulesMap := p.releaseRules.Map()

	for _, commit := range commits {

		if !conventionalCommitRegex.MatchString(commit.Message) {
			continue
		}

		submatch := conventionalCommitRegex.FindStringSubmatch(commit.Message)
		breakingChange := strings.Contains(submatch[3], "!") || strings.Contains(submatch[0], "BREAKING CHANGE")
		commitType := submatch[1]
		shortHash := commit.Hash.String()[0:7]
		shortMessage := p.shortMessage(commit.Message)

		if breakingChange {
			if p.verbose {
				p.logger.Info("found breaking change", "commit-hash", shortHash, "commit-message", shortMessage)
			}
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
			newRelease = true
			newReleaseType = "patch"
		case "minor":
			latestSemver.BumpMinor()
			newRelease = true
			newReleaseType = "minor"
		case "major":
			latestSemver.BumpMajor()
			newRelease = true
			newReleaseType = "major"
		default:
			return false, fmt.Errorf("unknown release type %s", releaseType)
		}

		if p.verbose {
			p.logger.Info("new release found", "commit-hash", shortHash, "commit-message", shortMessage, "release-type", newReleaseType)
		}

	}

	return newRelease, nil
}

func (p *Parser) shortMessage(message string) string {
	if len(message) > 50 {
		return fmt.Sprintf("%s...", message[0:47])
	}

	return message
}

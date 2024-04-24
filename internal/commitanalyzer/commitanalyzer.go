package commitanalyzer

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/s0ders/go-semver-release/internal/releaserules"
	"github.com/s0ders/go-semver-release/internal/semver"
	"github.com/s0ders/go-semver-release/internal/tagger"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test){1}(\([\w\-\.\\\/]+\))?(!)?: ([\w ])+([\s\S]*)`)

type CommitAnalyzer struct {
	logger       *slog.Logger
	releaseRules *releaserules.ReleaseRules
	verbose      bool
}

func New(logger *slog.Logger, releaseRules *releaserules.ReleaseRules, verbose bool) *CommitAnalyzer {
	return &CommitAnalyzer{
		logger:       logger,
		releaseRules: releaseRules,
		verbose:      verbose,
	}
}

func (c *CommitAnalyzer) fetchLatestSemverTag(r *git.Repository) (*object.Tag, error) {
	semverRegex := regexp.MustCompile(semver.Regex)

	tags, err := r.TagObjects()
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
		c.logger.Info("no previous tag, creating one")
		head, err := r.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch head: %w", err)
		}

		version, err := semver.New(0, 0, 0, "")
		if err != nil {
			return nil, fmt.Errorf("failed to build new semver: %w", err)
		}

		return tagger.NewTagFromSemver(*version, head.Hash()), nil
	}

	c.logger.Info("found latest semver tag", "tag", latestTag.Name)

	return latestTag, nil
}

func (c *CommitAnalyzer) ComputeNewSemver(r *git.Repository) (*semver.Semver, bool, error) {
	latestSemverTag, err := c.fetchLatestSemverTag(r)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch latest semver: %w", err)
	}

	newRelease := false
	newReleaseType := ""
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
		return nil, false, err
	}

	var history []*object.Commit

	err = commitHistory.ForEach(func(c *object.Commit) error {
		history = append(history, c)
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	// Reverse commit history to go from oldest to most recent
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	rulesMap := c.releaseRules.Map()

	for _, commit := range history {

		if !conventionalCommitRegex.MatchString(commit.Message) {
			continue
		}

		submatch := conventionalCommitRegex.FindStringSubmatch(commit.Message)
		breakingChange := strings.Contains(submatch[3], "!") || strings.Contains(submatch[0], "BREAKING CHANGE")
		commitType := submatch[1]
		shortHash := commit.Hash.String()[0:7]
		shortMessage := c.shortMessage(commit.Message)

		if breakingChange {
			c.logger.Info("found breaking change", "commit", shortHash)
			semverFromTag.BumpMajor()
			newRelease = true
			continue
		}

		releaseType, commitMatchesARule := rulesMap[commitType]

		if !commitMatchesARule {
			continue
		}

		switch releaseType {
		case "patch":
			semverFromTag.BumpPatch()
			newRelease = true
			newReleaseType = "patch"
		case "minor":
			semverFromTag.BumpMinor()
			newRelease = true
			newReleaseType = "minor"
		case "major":
			semverFromTag.BumpMajor()
			newRelease = true
			newReleaseType = "major"
		default:
			return nil, false, fmt.Errorf("unknown release type %s", releaseType)
		}

		if c.verbose {
			c.logger.Info("new release found", "commit-hash", shortHash, "commit-message", shortMessage, "release-type", newReleaseType)
		}
	}

	return semverFromTag, newRelease, nil
}

func (c *CommitAnalyzer) shortMessage(message string) string {
	if len(message) > 50 {
		return fmt.Sprintf("%s...", message[0:47])
	}

	return message
}

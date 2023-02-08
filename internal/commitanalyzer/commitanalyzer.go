package commitanalyzer

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/s0ders/go-semver-release/internal/releaserules"
	"github.com/s0ders/go-semver-release/internal/semver"
	"github.com/s0ders/go-semver-release/internal/tagger"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test){1}(\([\w\-\.\\\/]+\))?(!)?: ([\w ])+([\s\S]*)`)
)

type CommitAnalyzer struct {
	logger       *log.Logger
	releaseRules *releaserules.ReleaseRules
}

func NewCommitAnalyzer(releaseRules *releaserules.ReleaseRules) *CommitAnalyzer {
	logger := log.New(os.Stdout, fmt.Sprintf("%-20s ", "[commit-analyzer]"), log.Default().Flags())

	return &CommitAnalyzer{
		logger:       logger,
		releaseRules: releaseRules,
	}
}

func (c *CommitAnalyzer) fetchLatestSemverTag(r *git.Repository) (*object.Tag, error) {

	semverRegex := regexp.MustCompile(semver.SemverRegex)

	tags, err := r.TagObjects()
	if err != nil {
		return nil, err
	}

	var semverTags []*object.Tag

	tags.ForEach(func(tag *object.Tag) error {
		if semverRegex.MatchString(tag.Name) {
			semverTags = append(semverTags, tag)
		}
		return nil
	})

	if len(semverTags) == 0 {
		c.logger.Println("no previous tag, creating one")
		head, err := r.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch head: %w", err)
		}
		version, err := semver.NewSemver(0, 0, 0, "")
		if err != nil {
			return nil, fmt.Errorf("failed to build new semver: %w", err)
		}
		return tagger.NewTag(*version, head.Hash()), nil
	}

	if len(semverTags) == 1 {
		return semverTags[0], nil
	}

	var latestSemverTag *object.Tag

	for i, tag := range semverTags {
		current, err := semver.NewSemverFromGitTag(semverTags[i])
		if err != nil {
			return nil, fmt.Errorf("failed to build semver from tag: %w", err)
		}

		if i == 0 {
			latestSemverTag = tag
			continue
		}

		old, err := semver.NewSemverFromGitTag(latestSemverTag)
		if err != nil {
			return nil, fmt.Errorf("failed to build semver from tag: %w", err)
		}

		if current.Precedence(*old) == 1 {
			latestSemverTag = tag
		}
	}

	c.logger.Printf("latest semver tag: %s\n", latestSemverTag.Name)

	return latestSemverTag, nil
}

func (c *CommitAnalyzer) ComputeNewSemver(r *git.Repository) (*semver.Semver, bool, error) {

	latestSemverTag, err := c.fetchLatestSemverTag(r)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch latest semver: %w", err)
	}

	newRelease := false
	semver, err := semver.NewSemverFromGitTag(latestSemverTag)
	if err != nil {
		return nil, false, fmt.Errorf("failed to build semver from git tag: %w", err)
	}

	logOptions := &git.LogOptions{}

	if !semver.IsZero() {
		logOptions.Since = &latestSemverTag.Tagger.When
	}

	commitHistory, err := r.Log(logOptions)
	if err != nil {
		c.logger.Fatalf("failed to fetch commit history: %s", err)
	}

	var history []*object.Commit

	commitHistory.ForEach(func(c *object.Commit) error {
		history = append(history, c)
		return nil
	})

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
			c.logger.Printf("(%s) breaking change", shortHash)
			semver.BumpMajor()
			newRelease = true
			continue
		}

		releaseType, commitMatchesARule := rulesMap[commitType]

		if !commitMatchesARule {
			continue
		}

		switch releaseType {
		case "patch":
			c.logger.Printf("(%s) patch: \"%s\"", shortHash, shortMessage)
			semver.BumpPatch()
			newRelease = true
		case "minor":
			c.logger.Printf("(%s) minor: \"%s\"", shortHash, shortMessage)
			semver.BumpMinor()
			newRelease = true
		case "major":
			c.logger.Printf("(%s) major: \"%s\"", shortHash, shortMessage)
			semver.BumpMajor()
			newRelease = true
		default:
			c.logger.Fatalf("found a rule but no associated release type")
		}
		c.logger.Printf("version is now %s", semver)
	}

	if err != nil {
		return nil, false, fmt.Errorf("failed to parse commit history: %w", err)
	}

	return semver, newRelease, nil
}

func (c *CommitAnalyzer) shortMessage(message string) string {
	if len(message) > 50 {
		return fmt.Sprintf("%s...", message[0:47])
	}

	return message[0 : len(message)-1]
}

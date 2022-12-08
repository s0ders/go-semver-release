package commitanalyzer

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/s0ders/go-semver-release/semver"
	"github.com/s0ders/go-semver-release/tagger"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/go-playground/validator/v10"
)

var (
	conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test){1}(\([\w\-\.\\\/]+\))?(!)?: ([\w ])+([\s\S]*)`)
)

type ReleaseRule struct {
	CommitType  string `json:"type" validate:"required,oneof=build chore ci docs feat fix perf refactor revert style test"`
	ReleaseType string `json:"release" validate:"required,oneof=major minor patch"`
}

type ReleaseRules struct {
	Rules []ReleaseRule `json:"releaseRules" validate:"required"`
}

type CommitAnalyzer struct {
	logger       *log.Logger
	releaseRules *ReleaseRules
}

func NewCommitAnalyzer(l *log.Logger, releaseRulesReader io.Reader) (*CommitAnalyzer, error) {

	releaseRules, err := ParseReleaseRules(releaseRulesReader)
	if err != nil {
		return nil, fmt.Errorf("NewCommitAnalyzer: failed parsing release rules: %w", err)
	}

	return &CommitAnalyzer{l, releaseRules}, nil
}

func ParseReleaseRules(releaseRulesReader io.Reader) (*ReleaseRules, error) {
	var releaseRules *ReleaseRules

	decoder := json.NewDecoder(releaseRulesReader)
	decoder.Decode(&releaseRules)

	validate := validator.New()
	if err := validate.Struct(releaseRules); err != nil {
		return nil, fmt.Errorf("ParseReleaseRules: failed to validate release rules: %w", err)
	}

	for _, rule := range releaseRules.Rules {
		err := validate.Struct(rule)
		if err = validate.Struct(releaseRules); err != nil {
			return nil, fmt.Errorf("ParseReleaseRules: failed to validate release rules: %w", err)
		}
	}

	return releaseRules, nil
}

// FetchLatestSemverTag fetches all tags from a given Git repository
// and match them all against a regex describing a valid semver number.
// The valid semver tag are then sorted and the one with the highest
// precedence (i.e. latest tag) is returned.
func (c *CommitAnalyzer) FetchLatestSemverTag(r *git.Repository) (*object.Tag, error) {

	semverRegex := regexp.MustCompile(semver.SemverRegex)

	tags, err := r.TagObjects()
	if err != nil {
		return nil, err
	}

	semverTags := make([]*object.Tag, 0)

	
	// Filter tags who match a semver number (no matter the prefix)
	tags.ForEach(func(tag *object.Tag) error {
		if semverRegex.MatchString(tag.Name) {
			semverTags = append(semverTags, tag)
		}
		return nil
	})
	
	// If there are no existing semver tag, create one
	if len(semverTags) == 0 {
		c.logger.Println("no previous tag, creating one")
		head, err := r.Head()
		if err != nil {
			return nil, fmt.Errorf("FetchLatestSemverTag: failed to fetch head: %w", err)
		}
		version, err := semver.NewSemver(0, 0, 0, "")
		if err != nil {
			return nil, fmt.Errorf("FetchLatestSemverTag: failed to build new semver: %w", err)
		}
		return tagger.NewTag(*version, head.Hash()), nil
		
	}
	
	// If there is only one semver tags
	if len(semverTags) == 1 {
		return semverTags[0], nil
	}
	
	
	// If there are multiple semver tags, they are sorted to find the semver tags who has the precedence
	var latestSemverTag *object.Tag

	for i, tag := range semverTags {
		current, err := semver.NewSemverFromGitTag(semverTags[i])
		if err != nil {
			return nil, fmt.Errorf("FetchLatestSemverTag: failed to build semver from tag: %w", err)
		}

		if i == 0 {
			latestSemverTag = tag
			continue
		}

		old, err := semver.NewSemverFromGitTag(latestSemverTag)
		if err != nil {
			return nil, fmt.Errorf("FetchLatestSemverTag: failed to build semver from tag: %w", err)
		}

		if current.Precedence(*old) == 1 {
			latestSemverTag = tag
		}
	}

	c.logger.Printf("latest semver tag: %s\n", latestSemverTag.Name)

	return latestSemverTag, nil
}

// ComputeNewSemverNumber takes a chronologically ordered (starting from oldest)
// slice of commit history and the latest valid semver from the repository and 
// returns the updated semver number using the defined release rules.
func (c *CommitAnalyzer) ComputeNewSemverNumber(r *git.Repository, latestSemverTag *object.Tag, releaseBranch string) (*semver.Semver, bool, error) {

	newRelease := false
	semver, err := semver.NewSemverFromGitTag(latestSemverTag)
	if err != nil {
		return nil, false, fmt.Errorf("ComputeNewSemverNumber: failed to build SemVer from Git tag: %w", err)
	}

	logOptions := &git.LogOptions{}

	if semver.IsZero() {
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

	for _, commit := range history {

		if !conventionalCommitRegex.MatchString(commit.Message) {
			continue
		}

		submatch := conventionalCommitRegex.FindStringSubmatch(commit.Message)
		commitType := submatch[1]
		breakingChange := strings.Contains(submatch[3], "!") || strings.Contains(submatch[0], "BREAKING CHANGE")
		shortHash := commit.Hash.String()[0:7]
		var shortMessage string

		if len(commit.Message) > 60 {
			shortMessage = fmt.Sprintf("%s...", commit.Message[0:57])
		} else {
			shortMessage = commit.Message[0 : len(commit.Message)-1]
		}

		if breakingChange {
			c.logger.Printf("(%s) Breaking change", shortHash)
			semver.BumpMajor()
			newRelease = true
		}

		for _, rule := range c.releaseRules.Rules {
			if commitType != rule.CommitType {
				continue
			}

			switch rule.ReleaseType {
			case "major":
				c.logger.Printf("(%s) major: \"%s\"", shortHash, shortMessage)
				semver.BumpMajor()
				newRelease = true
			case "minor":
				c.logger.Printf("(%s) minor: \"%s\"", shortHash, shortMessage)
				semver.BumpMinor()
				newRelease = true
			case "patch":
				c.logger.Printf("(%s) patch: \"%s\"", shortHash, shortMessage)
				semver.BumpPatch()
				newRelease = true
			default:
				c.logger.Printf("no release to apply")
			}
			c.logger.Printf("version is now %s", semver)
		}

	}

	if err != nil {
		return nil, false, fmt.Errorf("ComputeNewSemverNumber: failed to parse commit history: %w", err)
	}

	return semver, newRelease, nil
}

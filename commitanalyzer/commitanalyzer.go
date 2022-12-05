package commitanalyzer

import (
	"log"
	"regexp"

	"github.com/s0ders/go-semver-release/semver"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// Regular expression to match valid semantic version number
var semverRegex = regexp.MustCompile("v[0-9]+.[0-9]+.[0-9]+")

// Regular expression to match valid conventional commit
var conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test){1}(\([\w\-\.]+\))?(!)?: ([\w ])+([\s\S]*)`)
var commitTypeRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test){1}`)

var breakingChangeRegex = regexp.MustCompile("BREAKING CHANGE")
var breakingChangeScopeRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test){1}(\([\w\-\.]+\))?!:`)

type CommitAnalyzer struct {
	logger *log.Logger
}

func NewCommitAnalyzer(l *log.Logger) CommitAnalyzer {
	return CommitAnalyzer{l}
}


func (c CommitAnalyzer) FetchLatestSemverTag(tags *object.TagIter) *object.Tag {
	// Stores all tags matching a semver
	semverTags := make([]*object.Tag, 0)

	var latestSemverTag *object.Tag

	// Fetch all tags matching a semver
	tags.ForEach(func(tag *object.Tag) error {
		if semverRegex.MatchString(string(tag.Name)) {
			semverTags = append(semverTags, tag)
		}
		return nil
	})

	if len(semverTags) < 1 {
		// TODO: create and push annotated tag v0.0.0
		c.logger.Fatalf("No tags on repository")
	} else if len(semverTags) < 2 {
		latestSemverTag = semverTags[0]
	} else {
		for i := 0; i < len(semverTags)-1; i++ {
			v1, err := semver.NewSemver(semverTags[i])
			failOnError(err)
			v2, err := semver.NewSemver(semverTags[i+1])
			failOnError(err)

			comparison := semver.CompareSemver(*v1, *v2)

			switch comparison {
			case 1:
				latestSemverTag = semverTags[i]
			case -1:
				latestSemverTag = semverTags[i+1]
			default:
				latestSemverTag = semverTags[i]
			}
		}
	}

	c.logger.Printf("Latest semver tag: %s\n", latestSemverTag.Name)

	return latestSemverTag
}

func (c CommitAnalyzer) ComputeNewSemverNumber(history object.CommitIter, latestSemverTag *object.Tag, releaseRules *string) *semver.Semver {

	semver, err := semver.NewSemver(latestSemverTag)
	failOnError(err)

	err = history.ForEach(func(commit *object.Commit) error {

		c.logger.Printf("New commit since last tag: %s\n", commit.Message)
		
		if !conventionalCommitRegex.MatchString(commit.Message) {
			c.logger.Printf("Commit did not match CC spec: %s\n", commit.Message)
			return nil
		}

		breakingChange := breakingChangeRegex.MatchString(commit.Message) || breakingChangeScopeRegex.MatchString(commit.Message)

		if breakingChange {
			c.logger.Printf("Detected breaking change")
			semver.IncrMajor()
			return nil
		}

		commitType := commitTypeRegex.FindString(commit.Message)

		c.logger.Printf("Commit type: %s\n", commitType)

		switch commitType {
		case "feat":
			c.logger.Printf("Detected minor change")
			semver.IncrMinor()
		case "fix":
			c.logger.Printf("Detected patch change")
			semver.IncrPatch()
		}

		return nil
	})
	failOnError(err) 
	
	return semver
}



func failOnError(e error) {
	if e != nil {
		log.Fatalf("Error: %s\n", e)
	}
}
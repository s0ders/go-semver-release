package commitanalyzer

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/s0ders/go-semver-release/semver"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/go-playground/validator/v10"
)

var ( 
	semverRegex = regexp.MustCompile("^v[0-9]+.[0-9]+.[0-9]+$")
 	conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test){1}(\([\w\-\.\\\/]+\))?(!)?: ([\w ])+([\s\S]*)`)
	defaultReleaseRules = ReleaseRules{Rules: []ReleaseRule{{"feat", "minor"},	{"fix", "patch"}}}
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
	releaseRules ReleaseRules
}

func NewCommitAnalyzer(l *log.Logger, releaseRulesPath *string) CommitAnalyzer {

	if *releaseRulesPath != "" {
		log.Printf("Using custom release rules")
		releaseRules := ParseReleaseRules(releaseRulesPath)
		log.Printf("%+v", releaseRules)
		return CommitAnalyzer{l, releaseRules}
	}

	log.Printf("Using default release rules")

	return CommitAnalyzer{l, defaultReleaseRules}
}

func ParseReleaseRules(path *string) ReleaseRules {
	jsonFile, err := os.Open(*path)
	failOnError(err)

	var releaseRules ReleaseRules

	decoder := json.NewDecoder(jsonFile)

	decoder.Decode(&releaseRules)

	validate := validator.New()

	err = validate.Struct(releaseRules)
	failOnError(err)

	for _, rule := range releaseRules.Rules {
		err := validate.Struct(rule)
		failOnError(err)
	}

	return releaseRules
}

func (c CommitAnalyzer) FetchLatestSemverTag(tags *object.TagIter) *object.Tag {
	semverTags := make([]*object.Tag, 0)

	var latestSemverTag *object.Tag

	tags.ForEach(func(tag *object.Tag) error {
		if semverRegex.MatchString(string(tag.Name)) {
			semverTags = append(semverTags, tag)
		}
		return nil
	})

	if len(semverTags) < 1 {
		// TODO: Handle case where the repo has not semver tag (craft one ?)
		panic("No tags yet!")
	} else if len(semverTags) < 2 {
		return semverTags[0]
	}
	
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

	c.logger.Printf("Latest semver tag: %s\n", latestSemverTag.Name)

	return latestSemverTag
}

func (c CommitAnalyzer) ComputeNewSemverNumber(history object.CommitIter, latestSemverTag *object.Tag) *semver.Semver {

	semver, err := semver.NewSemver(latestSemverTag)
	failOnError(err)

	err = history.ForEach(func(commit *object.Commit) error {

		c.logger.Printf("New commit since last tag: %s\n", commit.Message)

		if !conventionalCommitRegex.MatchString(commit.Message) {
			c.logger.Printf("Commit did not match CC spec: %s\n", commit.Message)
			return nil
		}

		submatch := conventionalCommitRegex.FindStringSubmatch(commit.Message)
		commitType := submatch[1]		
		breakingChange := strings.Contains(submatch[3], "!") || strings.Contains(submatch[0], "BREAKING CHANGE")

		if breakingChange {
			c.logger.Printf("Detected breaking change")
			semver.IncrMajor()
			return nil
		}

		c.logger.Printf("Commit type: %s\n", commitType)

		for _, rule := range c.releaseRules.Rules {
			if commitType != rule.CommitType {
				break
			}

			switch rule.ReleaseType {
			case "major":
				c.logger.Printf("Applying major release rule")
				semver.IncrMajor()
			case "minor":
				c.logger.Printf("Applying minor release rule")
				semver.IncrMinor()
			case "patch":
				c.logger.Printf("Applying patch release rule")
				semver.IncrPatch()
			default:
				c.logger.Printf("No release rule to apply")
			}
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

// Package rule provides functions to handle release rule configuration.
package rule

import (
	"errors"
)

type Rules struct {
	Map map[string]string
}

var Default = Rules{
	Map: map[string]string{
		"feat":   "minor",
		"fix":    "patch",
		"perf":   "patch",
		"revert": "patch",
	},
}

var (
	ErrInvalidCommitType    = errors.New("invalid commit type")
	ErrInvalidReleaseType   = errors.New("invalid release type")
	ErrDuplicateReleaseRule = errors.New("duplicate release rule for the same commit type")
	ErrNoRules              = errors.New("no rule found")
)

var validCommitTypes = map[string]struct{}{
	"build":    {},
	"chore":    {},
	"ci":       {},
	"docs":     {},
	"feat":     {},
	"fix":      {},
	"perf":     {},
	"refactor": {},
	"revert":   {},
	"style":    {},
	"test":     {},
}

var validReleaseTypes = map[string]struct{}{
	"minor": {},
	"patch": {},
}

// Unmarshall takes a raw Viper configuration and returns a Rules struct representing release rules configuration.
func Unmarshall(input map[string][]string) (Rules, error) {
	var rules Rules
	rules.Map = make(map[string]string)

	if len(input) == 0 {
		return rules, ErrNoRules
	}

	for releaseType, commitTypes := range input {
		if _, ok := validReleaseTypes[releaseType]; !ok {
			return rules, ErrInvalidReleaseType
		}

		for _, commitType := range commitTypes {
			if _, ok := validCommitTypes[commitType]; !ok {
				return rules, ErrInvalidCommitType
			}

			if _, ok := rules.Map[commitType]; ok {
				return rules, ErrDuplicateReleaseRule
			}

			rules.Map[commitType] = releaseType
		}
	}

	return rules, nil
}

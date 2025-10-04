// Package rule provides functions to handle release rule configuration.
package rule

import (
	"errors"
)

var (
	ErrInvalidCommitType    = errors.New("invalid commit type")
	ErrInvalidReleaseType   = errors.New("invalid release type")
	ErrDuplicateReleaseRule = errors.New("duplicate release rule for the same commit type")
)

var Default = map[string][]string{
	"minor": {"feat"},
	"patch": {"fix", "perf", "revert"},
}

var ValidCommitTypes = map[string]struct{}{
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

var ValidReleaseTypes = map[string]struct{}{
	"minor": {},
	"patch": {},
}

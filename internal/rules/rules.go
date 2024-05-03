// Package rules provides functions to deal with release rules.
package rules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const Default = `{
	"rules": [
		{"type": "feat", "release": "minor"},
		{"type": "fix", "release": "patch"},
		{"type": "perf", "release": "patch"},
		{"type": "revert", "release": "patch"}
	]
}`

var (
	ErrInvalidCommitType    = errors.New("invalid commit type")
	ErrInvalidReleaseType   = errors.New("invalid release type")
	ErrDuplicateReleaseRule = errors.New("duplicate release rule for the same commit type")
	ErrNoRules              = errors.New("no rules found")
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

type ReleaseRule struct {
	CommitType  string `json:"type"`
	ReleaseType string `json:"release"`
}

type ReleaseRules struct {
	Rules []ReleaseRule `json:"rules"`
}

type Options struct {
	Reader io.Reader
}

// Map returns a flat map corresponding to the release rules with commit types as keys and release types as values.
func (r *ReleaseRules) Map() map[string]string {
	m := make(map[string]string, len(r.Rules))
	for _, rule := range r.Rules {
		m[rule.CommitType] = rule.ReleaseType
	}
	return m
}

// Init initialize a new set of release rules with the given options if any.
func Init(opts *Options) (rr *ReleaseRules, err error) {
	if opts == nil || opts.Reader == nil {
		reader := strings.NewReader(Default)
		rr, err = Parse(reader)
		return rr, err
	}

	rr, err = Parse(opts.Reader)

	return rr, err
}

// Parse reads a buffer a returns the corresponding release rules.
func Parse(reader io.Reader) (*ReleaseRules, error) {
	var rules ReleaseRules
	existingType := make(map[string]string)

	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	bufReader := bytes.NewReader(buf)

	decoder := json.NewDecoder(bufReader)
	err = decoder.Decode(&rules)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON into rules: %w", err)
	}

	for _, rule := range rules.Rules {
		if _, ok := validCommitTypes[rule.CommitType]; !ok {
			return nil, ErrInvalidCommitType
		}

		if _, ok := validReleaseTypes[rule.ReleaseType]; !ok {
			return nil, ErrInvalidReleaseType
		}

		if _, ok := existingType[rule.CommitType]; ok {
			return nil, ErrDuplicateReleaseRule
		}

		existingType[rule.CommitType] = rule.ReleaseType
	}

	if len(rules.Rules) == 0 {
		return nil, ErrNoRules
	}

	return &rules, nil
}

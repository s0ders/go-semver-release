// Package rule provides functions to deal with release rule.
package rule

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

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

type ReleaseRule struct {
	CommitType  string `json:"type"`
	ReleaseType string `json:"release"`
}

type ReleaseRules struct {
	Rules []ReleaseRule `json:"rule"`
}

var Default = ReleaseRules{Rules: []ReleaseRule{
	{"feat", "minor"},
	{"fix", "patch"},
	{"perf", "patch"},
	{"revert", "patch"},
}}

type Options struct {
	Reader io.Reader
}

type OptionFunc func(*Options)

func WithReader(reader io.Reader) OptionFunc {
	return func(o *Options) {
		o.Reader = reader
	}
}

// Map returns a flat map corresponding to the release rule with commit types as keys and release types as values.
func (r *ReleaseRules) Map() map[string]string {
	m := make(map[string]string, len(r.Rules))
	for _, rule := range r.Rules {
		m[rule.CommitType] = rule.ReleaseType
	}
	return m
}

// Init initialize a new set of release rule with the given options if any.
func Init(options ...OptionFunc) (ReleaseRules, error) {

	opts := &Options{}

	for _, optionFunc := range options {
		optionFunc(opts)
	}

	if opts.Reader == nil {
		return Default, nil
	}

	return Parse(opts.Reader)
}

// Parse reads a buffer a returns the corresponding release rule.
func Parse(reader io.Reader) (ReleaseRules, error) {
	var rules ReleaseRules
	existingType := make(map[string]string)

	buf, err := io.ReadAll(reader)
	if err != nil {
		return ReleaseRules{}, fmt.Errorf("reading rule file: %w", err)
	}

	bufReader := bytes.NewReader(buf)

	decoder := json.NewDecoder(bufReader)
	err = decoder.Decode(&rules)
	if err != nil {
		return ReleaseRules{}, fmt.Errorf("decoding JSON into rule: %w", err)
	}

	for _, rule := range rules.Rules {
		if _, ok := validCommitTypes[rule.CommitType]; !ok {
			return ReleaseRules{}, ErrInvalidCommitType
		}

		if _, ok := validReleaseTypes[rule.ReleaseType]; !ok {
			return ReleaseRules{}, ErrInvalidReleaseType
		}

		if _, ok := existingType[rule.CommitType]; ok {
			return ReleaseRules{}, ErrDuplicateReleaseRule
		}

		existingType[rule.CommitType] = rule.ReleaseType
	}

	if len(rules.Rules) == 0 {
		return ReleaseRules{}, ErrNoRules
	}

	return rules, nil
}

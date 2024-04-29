// Package rules provides functions to deal with release rules.
package rules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// TODO: make into a map or struct directly ?
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
	"major": {},
	"minor": {},
	"patch": {},
}

type Reader struct {
	logger *slog.Logger
	reader io.Reader
}

// TODO: remove validator
type ReleaseRule struct {
	CommitType  string `json:"type"`
	ReleaseType string `json:"release"`
}

type ReleaseRules struct {
	Rules []ReleaseRule `json:"rules" validate:"required"`
}

func (r *ReleaseRules) Map() map[string]string {
	m := make(map[string]string)
	for _, rule := range r.Rules {
		m[rule.CommitType] = rule.ReleaseType
	}
	return m
}

func New(logger *slog.Logger) *Reader {
	return &Reader{
		logger: logger,
	}
}

// TODO: pass an io.Reader directly ?
func (r *Reader) Read(path string) (rr *Reader, err error) {
	if len(path) == 0 {
		r.reader = strings.NewReader(Default)
		return r, nil
	}

	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open rules file: %w", err)
	}

	defer func() {
		err = reader.Close()
		return
	}()

	r.reader = reader

	return r, nil
}

func (r *Reader) Parse() (*ReleaseRules, error) {
	var rules ReleaseRules
	existingType := make(map[string]string)

	buf, err := io.ReadAll(r.reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	bufReader := bytes.NewReader(buf)

	decoder := json.NewDecoder(bufReader)
	err = decoder.Decode(&rules)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON into rules")
	}

	if len(rules.Rules) == 0 {
		return nil, fmt.Errorf("no release rules found")
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

	return &rules, nil
}

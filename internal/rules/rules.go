// Package rules provides functions to deal with release rules.
package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-playground/validator/v10"
)

const Default = `{
	"rules": [
		{"type": "feat", "release": "minor"},
		{"type": "fix", "release": "patch"},
		{"type": "perf", "release": "patch"},
		{"type": "revert", "release": "patch"}
	]
}`

type Reader struct {
	logger *slog.Logger
	reader io.Reader
}

// TODO: remove validator
type ReleaseRule struct {
	CommitType  string `json:"type" yaml:"type" validate:"required,oneof=build chore ci docs feat fix perf refactor revert style test"`
	ReleaseType string `json:"release" yaml:"release" validate:"required,oneof=major minor patch"`
}

type ReleaseRules struct {
	Rules []ReleaseRule `json:"rules" yaml:"rules" validate:"required"`
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

	switch {
	case isJSON(buf):
		decoder := json.NewDecoder(bufReader)
		err = decoder.Decode(&rules)
		if err != nil {
			return nil, fmt.Errorf("failed to decode JSON into rules")
		}
	case isYAML(buf):
		decoder := yaml.NewDecoder(bufReader)
		err = decoder.Decode(&rules)
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML into rules")
		}
	default:
		return nil, fmt.Errorf("failed to detect if rules are JSON or YAML")
	}

	validate := validator.New()
	if err := validate.Struct(rules); err != nil {
		return nil, fmt.Errorf("failed to validate release rules: %w", err)
	}

	for _, rule := range rules.Rules {

		existingRuleType, commitTypeAlreadyAssigned := existingType[rule.CommitType]

		if commitTypeAlreadyAssigned {
			return nil, fmt.Errorf("a release rule already exist for commit type %s (%s), remove one to avoid conflict", rule.CommitType, existingRuleType)
		}

		if err := validate.Struct(rule); err != nil {
			return nil, fmt.Errorf("failed to validate release rules: %w", err)
		}
		existingType[rule.CommitType] = rule.ReleaseType
	}

	return &rules, nil
}

func isJSON(b []byte) bool {
	var raw json.RawMessage
	return json.Unmarshal(b, &raw) == nil
}

func isYAML(b []byte) bool {
	var raw map[any]any
	return yaml.Unmarshal(b, &raw) == nil
}

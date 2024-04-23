package releaserules

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

const DefaultRules = `{
	"releaseRules": [
		{"type": "feat", "release": "minor"},
		{"type": "perf", "release": "minor"},
		{"type": "fix", "release": "patch"}
	]
}`

type ReleaseRuleReader struct {
	logger *slog.Logger
	reader io.Reader
}

type ReleaseRule struct {
	CommitType  string `json:"type" validate:"required,oneof=build chore ci docs feat fix perf refactor revert style test"`
	ReleaseType string `json:"release" validate:"required,oneof=major minor patch"`
}

type ReleaseRules struct {
	Rules []ReleaseRule `json:"releaseRules" validate:"required"`
}

func (r *ReleaseRules) Map() map[string]string {
	m := make(map[string]string)
	for _, rule := range r.Rules {
		m[rule.CommitType] = rule.ReleaseType
	}
	return m
}

func New(logger *slog.Logger) *ReleaseRuleReader {
	return &ReleaseRuleReader{
		logger: logger,
	}
}

// TODO: pass an io.Reader directly ?
func (r *ReleaseRuleReader) Read(path string) (*ReleaseRuleReader, error) {
	if len(path) == 0 {
		r.reader = strings.NewReader(DefaultRules)
		return r, nil
	}

	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open rules file: %w", err)
	}

	return r.setReader(reader), nil
}

func (r *ReleaseRuleReader) setReader(reader io.Reader) *ReleaseRuleReader {
	r.reader = reader
	return r
}

func (r *ReleaseRuleReader) Parse() (*ReleaseRules, error) {
	var releaseRules ReleaseRules
	existingType := make(map[string]string)

	decoder := json.NewDecoder(r.reader)
	err := decoder.Decode(&releaseRules)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(releaseRules); err != nil {
		return nil, fmt.Errorf("failed to validate release rules: %w", err)
	}

	for _, rule := range releaseRules.Rules {

		existingRuleType, commitTypeAlreadyAssigned := existingType[rule.CommitType]

		if commitTypeAlreadyAssigned {
			return nil, fmt.Errorf("a release rule already exist for commit type %s (%s), remove one to avoid conflict", rule.CommitType, existingRuleType)
		}

		if err := validate.Struct(rule); err != nil {
			return nil, fmt.Errorf("failed to validate release rules: %w", err)
		}
		existingType[rule.CommitType] = rule.ReleaseType
	}

	return &releaseRules, nil
}

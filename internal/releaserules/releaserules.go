package releaserules

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/s0ders/go-semver-release/internal/helpers"
)

type ReleaseRuleReader struct {
	logger *log.Logger
	reader io.Reader
}

type ReleaseRule struct {
	CommitType  string `json:"type" validate:"required,oneof=build chore ci docs feat fix perf refactor revert style test"`
	ReleaseType string `json:"release" validate:"required,oneof=major minor patch"`
}

type ReleaseRules struct {
	Rules []ReleaseRule `json:"releaseRules" validate:"required"`
}

func NewReleaseRuleReader() *ReleaseRuleReader {
	logger := log.New(os.Stdout, fmt.Sprintf("%-20s ", "[releas-rule-reader]"), log.Default().Flags())
	return &ReleaseRuleReader{
		logger: logger,
	}
}

func (r *ReleaseRuleReader) Read(path string) *ReleaseRuleReader {
	if len(path) == 0 {
		r.reader = strings.NewReader(helpers.DefaultReleaseRules)
		return r
	}

	reader, err := os.Open(path)
	if err != nil {
		r.logger.Fatalf("failed to open rules file: %s", err)
	}

	r.reader = reader
	return r
}

func (r *ReleaseRuleReader) Parse() (*ReleaseRules, error) {
	var releaseRules ReleaseRules
	existingType := make(map[string]string)

	decoder := json.NewDecoder(r.reader)
	decoder.Decode(&releaseRules)

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

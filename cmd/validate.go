package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/s0ders/go-semver-release/v7/internal/rule"
)

type ValidationResult struct {
	Errors   []string
	Warnings []string
}

func (v *ValidationResult) AddError(format string, args ...interface{}) {
	v.Errors = append(v.Errors, fmt.Sprintf(format, args...))
}

func (v *ValidationResult) AddWarning(format string, args ...interface{}) {
	v.Warnings = append(v.Warnings, fmt.Sprintf(format, args...))
}

func (v *ValidationResult) HasErrors() bool {
	return len(v.Errors) > 0
}

type configFile struct {
	Branches []branchConfig   `yaml:"branches"`
	Monorepo []monorepoConfig `yaml:"monorepo"`
	Rules    rulesConfig      `yaml:"rules"`
}

type branchConfig struct {
	Name       string `yaml:"name"`
	Prerelease bool   `yaml:"prerelease"`
}

type monorepoConfig struct {
	Name  string   `yaml:"name"`
	Path  string   `yaml:"path"`
	Paths []string `yaml:"paths"`
}

type rulesConfig struct {
	Minor []string `yaml:"minor"`
	Patch []string `yaml:"patch"`
}

func NewValidateCmd() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate <config-file>",
		Short: "Validate a configuration file",
		Long:  "Validate a configuration file for syntax and semantic errors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := args[0]

			result, err := validateConfigFile(configPath)
			if err != nil {
				return err
			}

			printValidationResult(cmd, configPath, result)

			if result.HasErrors() {
				os.Exit(1)
			}

			return nil
		},
	}

	return validateCmd
}

func validateConfigFile(path string) (*ValidationResult, error) {
	result := &ValidationResult{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config configFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	validateBranches(config.Branches, result)
	validateMonorepo(config.Monorepo, result)
	validateRules(config.Rules, result)

	return result, nil
}

func validateBranches(branches []branchConfig, result *ValidationResult) {
	if len(branches) == 0 {
		result.AddWarning("no branches configured")
		return
	}

	hasStable := false
	hasPrerelease := false
	seenNames := make(map[string]bool)

	for i, b := range branches {
		if b.Name == "" {
			result.AddError("branches[%d]: name is required", i)
			continue
		}

		if seenNames[b.Name] {
			result.AddError("branches[%d]: duplicate branch name %q", i, b.Name)
		}
		seenNames[b.Name] = true

		if b.Prerelease {
			hasPrerelease = true
		} else {
			hasStable = true
		}
	}

	if hasPrerelease && !hasStable {
		result.AddWarning("prerelease branches configured but no stable branch defined")
	}
}

func validateMonorepo(monorepo []monorepoConfig, result *ValidationResult) {
	if len(monorepo) == 0 {
		return
	}

	seenNames := make(map[string]bool)

	for i, m := range monorepo {
		if m.Name == "" {
			result.AddError("monorepo[%d]: name is required", i)
			continue
		}

		if seenNames[m.Name] {
			result.AddError("monorepo[%d]: duplicate project name %q", i, m.Name)
		}
		seenNames[m.Name] = true

		hasPath := m.Path != ""
		hasPaths := len(m.Paths) > 0

		if hasPath && hasPaths {
			result.AddError("monorepo[%d]: project %q has both \"path\" and \"paths\" set (mutually exclusive)", i, m.Name)
		}

		if !hasPath && !hasPaths {
			result.AddWarning("monorepo[%d]: project %q has no path configured", i, m.Name)
		}
	}
}

func validateRules(rules rulesConfig, result *ValidationResult) {
	seenCommitTypes := make(map[string]string)

	for _, commitType := range rules.Minor {
		validateCommitType(commitType, "minor", seenCommitTypes, result)
	}

	for _, commitType := range rules.Patch {
		validateCommitType(commitType, "patch", seenCommitTypes, result)
	}
}

func validateCommitType(commitType, releaseType string, seen map[string]string, result *ValidationResult) {
	if _, valid := rule.ValidCommitTypes[commitType]; !valid {
		suggestion := suggestCommitType(commitType)
		if suggestion != "" {
			result.AddWarning("rules.%s: unknown commit type %q (did you mean %q?)", releaseType, commitType, suggestion)
		} else {
			result.AddWarning("rules.%s: unknown commit type %q", releaseType, commitType)
		}
		return
	}

	if existingRelease, exists := seen[commitType]; exists {
		result.AddError("rules: commit type %q is mapped to both %q and %q", commitType, existingRelease, releaseType)
		return
	}

	seen[commitType] = releaseType
}

func suggestCommitType(input string) string {
	input = strings.ToLower(input)

	suggestions := map[string]string{
		"feature":  "feat",
		"features": "feat",
		"bugfix":   "fix",
		"bug":      "fix",
		"fixes":    "fix",
		"document": "docs",
		"doc":      "docs",
		"testing":  "test",
		"tests":    "test",
		"perform":  "perf",
		"styles":   "style",
		"refact":   "refactor",
		"builds":   "build",
		"chores":   "chore",
	}

	if suggestion, ok := suggestions[input]; ok {
		return suggestion
	}

	return ""
}

func printValidationResult(cmd *cobra.Command, path string, result *ValidationResult) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Validating %s...\n\n", path)

	if !result.HasErrors() && len(result.Warnings) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\u2713 Configuration valid")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n0 errors, 0 warnings\n")
		return
	}

	for _, err := range result.Errors {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\u2717 %s\n", err)
	}

	for _, warn := range result.Warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\u26a0 %s\n", warn)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%d error(s), %d warning(s)\n", len(result.Errors), len(result.Warnings))
}

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/s0ders/go-semver-release/v8/internal/rule"
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

// rawConfig uses interface{} to accept any YAML structure for lenient validation
type rawConfig struct {
	Branches interface{} `yaml:"branches"`
	Monorepo interface{} `yaml:"monorepo"`
	Rules    interface{} `yaml:"rules"`
}

func NewValidateCmd() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate <CONFIGURATION_FILE_PATH>",
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

	var config rawConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	validateBranches(config.Branches, result)
	validateMonorepo(config.Monorepo, result)
	validateRules(config.Rules, result)

	return result, nil
}

func validateBranches(branches interface{}, result *ValidationResult) {
	if branches == nil {
		result.AddWarning("no branches configured")
		return
	}

	branchList, ok := branches.([]interface{})
	if !ok {
		result.AddError("branches: expected an array, got %T", branches)
		return
	}

	if len(branchList) == 0 {
		result.AddWarning("no branches configured")
		return
	}

	hasStable := false
	hasPrereleaseFlag := false
	seenNames := make(map[string]bool)

	for i, item := range branchList {
		branch, ok := item.(map[string]interface{})
		if !ok {
			// Check if it's a string (common mistake)
			if str, isStr := item.(string); isStr {
				result.AddError("branches[%d]: expected object with \"name\" key, got string %q (use \"- name: %s\" instead)", i, str, str)
			} else {
				result.AddError("branches[%d]: expected object with \"name\" key, got %T", i, item)
			}
			continue
		}

		name, hasName := branch["name"]
		if !hasName {
			result.AddError("branches[%d]: \"name\" key is required", i)
			continue
		}

		nameStr, ok := name.(string)
		if !ok {
			result.AddError("branches[%d]: \"name\" must be a string, got %T", i, name)
			continue
		}

		if nameStr == "" {
			result.AddError("branches[%d]: \"name\" cannot be empty", i)
			continue
		}

		if seenNames[nameStr] {
			result.AddError("branches[%d]: duplicate branch name %q", i, nameStr)
		}
		seenNames[nameStr] = true

		// Check if prerelease is set to true
		if pr, ok := branch["prerelease"].(bool); ok && pr {
			hasPrereleaseFlag = true
		} else {
			hasStable = true
		}
	}

	if hasPrereleaseFlag && !hasStable {
		result.AddWarning("prerelease branches configured but no stable branch defined")
	}
}

func validateMonorepo(monorepo interface{}, result *ValidationResult) {
	if monorepo == nil {
		return
	}

	monorepoList, ok := monorepo.([]interface{})
	if !ok {
		result.AddError("monorepo: expected an array, got %T", monorepo)
		return
	}

	if len(monorepoList) == 0 {
		return
	}

	seenNames := make(map[string]bool)

	for i, item := range monorepoList {
		project, ok := item.(map[string]interface{})
		if !ok {
			if str, isStr := item.(string); isStr {
				result.AddError("monorepo[%d]: expected object with \"name\" key, got string %q", i, str)
			} else {
				result.AddError("monorepo[%d]: expected object with \"name\" and \"path\" keys, got %T", i, item)
			}
			continue
		}

		name, hasName := project["name"]
		if !hasName {
			result.AddError("monorepo[%d]: \"name\" key is required", i)
			continue
		}

		nameStr, ok := name.(string)
		if !ok {
			result.AddError("monorepo[%d]: \"name\" must be a string, got %T", i, name)
			continue
		}

		if nameStr == "" {
			result.AddError("monorepo[%d]: \"name\" cannot be empty", i)
			continue
		}

		if seenNames[nameStr] {
			result.AddError("monorepo[%d]: duplicate project name %q", i, nameStr)
		}
		seenNames[nameStr] = true

		path, hasPath := project["path"]
		paths, hasPaths := project["paths"]

		pathSet := hasPath && path != nil && path != ""
		pathsSet := hasPaths && paths != nil

		if pathsSet {
			if pathsList, ok := paths.([]interface{}); ok {
				pathsSet = len(pathsList) > 0
			} else {
				pathsSet = false
			}
		}

		if pathSet && pathsSet {
			result.AddError("monorepo[%d]: project %q has both \"path\" and \"paths\" set (mutually exclusive)", i, nameStr)
		}

		if !pathSet && !pathsSet {
			result.AddWarning("monorepo[%d]: project %q has no path configured", i, nameStr)
		}
	}
}

func validateRules(rules interface{}, result *ValidationResult) {
	if rules == nil {
		return
	}

	rulesMap, ok := rules.(map[string]interface{})
	if !ok {
		result.AddError("rules: expected an object with \"minor\" and \"patch\" keys, got %T", rules)
		return
	}

	seenCommitTypes := make(map[string]string)

	if minor, hasMinor := rulesMap["minor"]; hasMinor {
		validateRulesList(minor, "minor", seenCommitTypes, result)
	}

	if patch, hasPatch := rulesMap["patch"]; hasPatch {
		validateRulesList(patch, "patch", seenCommitTypes, result)
	}
}

func validateRulesList(rulesList interface{}, releaseType string, seen map[string]string, result *ValidationResult) {
	if rulesList == nil {
		return
	}

	list, ok := rulesList.([]interface{})
	if !ok {
		result.AddError("rules.%s: expected an array of commit types, got %T", releaseType, rulesList)
		return
	}

	for i, item := range list {
		commitType, ok := item.(string)
		if !ok {
			result.AddError("rules.%s[%d]: expected string, got %T", releaseType, i, item)
			continue
		}

		validateCommitType(commitType, releaseType, seen, result)
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

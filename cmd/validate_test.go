package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfigFile_ValidConfig(t *testing.T) {
	content := `
branches:
  - name: main
  - name: rc
    prerelease: true
rules:
  minor:
    - feat
  patch:
    - fix
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Empty(t, result.Warnings)
}

func TestValidateConfigFile_InvalidYAML(t *testing.T) {
	content := `
branches:
  - name: [invalid
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	_, err := validateConfigFile(path)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid YAML syntax")
}

func TestValidateConfigFile_BranchesAsStrings(t *testing.T) {
	content := `
branches:
  - main
  - rc
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 2)
	assert.Contains(t, result.Errors[0], "expected object with \"name\" key")
	assert.Contains(t, result.Errors[0], "use \"- name: main\" instead")
}

func TestValidateConfigFile_MissingBranchName(t *testing.T) {
	content := `
branches:
  - name: main
  - prerelease: true
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "\"name\" key is required")
}

func TestValidateConfigFile_DuplicateBranchName(t *testing.T) {
	content := `
branches:
  - name: main
  - name: main
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "duplicate branch name")
}

func TestValidateConfigFile_PrereleaseWithoutStable(t *testing.T) {
	content := `
branches:
  - name: rc
    prerelease: true
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "no stable branch defined")
}

func TestValidateConfigFile_MonorepoPathAndPathsExclusive(t *testing.T) {
	content := `
monorepo:
  - name: api
    path: ./api/
    paths:
      - ./lib/
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "mutually exclusive")
}

func TestValidateConfigFile_MonorepoMissingName(t *testing.T) {
	content := `
monorepo:
  - path: ./api/
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "\"name\" key is required")
}

func TestValidateConfigFile_MonorepoNoPath(t *testing.T) {
	content := `
branches:
  - name: main
monorepo:
  - name: api
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "no path configured")
}

func TestValidateConfigFile_UnknownCommitType(t *testing.T) {
	content := `
branches:
  - name: main
rules:
  minor:
    - feature
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "unknown commit type")
	assert.Contains(t, result.Warnings[0], "did you mean")
}

func TestValidateConfigFile_DuplicateCommitTypeInRules(t *testing.T) {
	content := `
rules:
  minor:
    - feat
  patch:
    - feat
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "mapped to both")
}

func TestValidateConfigFile_EmptyConfig(t *testing.T) {
	content := ``
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "no branches configured")
}

func TestSuggestCommitType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature", "feat"},
		{"features", "feat"},
		{"bugfix", "fix"},
		{"bug", "fix"},
		{"document", "docs"},
		{"testing", "test"},
		{"unknown", ""},
		{"FEATURE", "feat"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := suggestCommitType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateConfigFile_BranchesNotArray(t *testing.T) {
	content := `
branches: "not-an-array"
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "expected an array")
}

func TestValidateConfigFile_BranchNameNotString(t *testing.T) {
	content := `
branches:
  - name: 123
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "must be a string")
}

func TestValidateConfigFile_BranchNameEmpty(t *testing.T) {
	content := `
branches:
  - name: ""
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "cannot be empty")
}

func TestValidateConfigFile_MonorepoNotArray(t *testing.T) {
	content := `
monorepo: "not-an-array"
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "expected an array")
}

func TestValidateConfigFile_MonorepoAsStrings(t *testing.T) {
	content := `
monorepo:
  - api
  - web
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "expected object with \"name\" key")
}

func TestValidateConfigFile_MonorepoNameNotString(t *testing.T) {
	content := `
monorepo:
  - name: 123
    path: ./api/
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "must be a string")
}

func TestValidateConfigFile_MonorepoNameEmpty(t *testing.T) {
	content := `
monorepo:
  - name: ""
    path: ./api/
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "cannot be empty")
}

func TestValidateConfigFile_MonorepoDuplicateName(t *testing.T) {
	content := `
monorepo:
  - name: api
    path: ./api/
  - name: api
    path: ./api2/
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "duplicate project name")
}

func TestValidateConfigFile_RulesNotObject(t *testing.T) {
	content := `
rules:
  - feat
  - fix
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "expected an object")
}

func TestValidateConfigFile_RulesMinorNotArray(t *testing.T) {
	content := `
rules:
  minor: feat
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "expected an array of commit types")
}

func TestValidateConfigFile_RulesCommitTypeNotString(t *testing.T) {
	content := `
rules:
  minor:
    - 123
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0], "expected string")
}

func TestValidateConfigFile_UnknownCommitTypeWithoutSuggestion(t *testing.T) {
	content := `
branches:
  - name: main
rules:
  minor:
    - xyz123
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	result, err := validateConfigFile(path)

	assert.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "unknown commit type")
	assert.NotContains(t, result.Warnings[0], "did you mean")
}

func TestValidateConfigFile_FileNotFound(t *testing.T) {
	_, err := validateConfigFile("/nonexistent/path/config.yaml")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestValidateCmd_Integration(t *testing.T) {
	content := `
branches:
  - name: main
rules:
  minor:
    - feat
  patch:
    - fix
`
	path := createTempConfig(t, content)
	defer func() {
		_ = os.Remove(path)
	}()

	cmd := NewValidateCmd()
	cmd.SetArgs([]string{path})

	err := cmd.Execute()

	assert.NoError(t, err)
}

func TestPrintValidationResult_Valid(t *testing.T) {
	cmd := NewValidateCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &ValidationResult{}
	printValidationResult(cmd, "test.yaml", result)

	output := buf.String()
	assert.Contains(t, output, "Validating test.yaml...")
	assert.Contains(t, output, "Configuration valid")
	assert.Contains(t, output, "0 errors, 0 warnings")
}

func TestPrintValidationResult_WithErrors(t *testing.T) {
	cmd := NewValidateCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &ValidationResult{
		Errors: []string{"error 1", "error 2"},
	}
	printValidationResult(cmd, "test.yaml", result)

	output := buf.String()
	assert.Contains(t, output, "error 1")
	assert.Contains(t, output, "error 2")
	assert.Contains(t, output, "2 error(s), 0 warning(s)")
}

func TestPrintValidationResult_WithWarnings(t *testing.T) {
	cmd := NewValidateCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &ValidationResult{
		Warnings: []string{"warning 1"},
	}
	printValidationResult(cmd, "test.yaml", result)

	output := buf.String()
	assert.Contains(t, output, "warning 1")
	assert.Contains(t, output, "0 error(s), 1 warning(s)")
}

func TestPrintValidationResult_WithErrorsAndWarnings(t *testing.T) {
	cmd := NewValidateCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &ValidationResult{
		Errors:   []string{"error 1"},
		Warnings: []string{"warning 1", "warning 2"},
	}
	printValidationResult(cmd, "test.yaml", result)

	output := buf.String()
	assert.Contains(t, output, "error 1")
	assert.Contains(t, output, "warning 1")
	assert.Contains(t, output, "warning 2")
	assert.Contains(t, output, "1 error(s), 2 warning(s)")
}

func createTempConfig(t *testing.T, content string) string {
	t.Helper()

	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, "test-config-*.yaml")

	f, err := os.CreateTemp(tmpDir, "test-config-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("closing temp file: %v", err)
	}

	_ = path // suppress unused warning
	return f.Name()
}

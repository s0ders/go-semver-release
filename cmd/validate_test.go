package cmd

import (
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

package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const noBranchesWarning = "no branches configured"

func TestValidateConfigFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		// wantErrors and wantWarnings list, in order, a substring of each expected error and warning.
		wantErrors   []string
		wantWarnings []string
	}{
		{
			name: "valid configuration",
			content: `
branches:
  - name: main
  - name: rc
    prerelease: true
monorepo:
  - name: api
    path: ./api/
  - name: web
    paths: [./web/, ./shared/]
rules:
  minor: [feat]
  patch: [fix]
`,
		},
		{name: "empty configuration", wantWarnings: []string{noBranchesWarning}},

		// Branches
		{
			name:       "branches as strings",
			content:    "branches:\n  - main\n  - rc\n",
			wantErrors: []string{`branches[0]: expected object with "name" key, got string "main" (use "- name: main" instead)`, `branches[1]`},
		},
		{name: "branches not an array", content: `branches: "main"`, wantErrors: []string{"branches: expected an array"}},
		{name: "missing branch name", content: "branches:\n  - name: main\n  - prerelease: true\n", wantErrors: []string{`branches[1]: "name" key is required`}},
		{name: "branch name not a string", content: "branches:\n  - name: 123\n", wantErrors: []string{`"name" must be a string`}},
		{name: "empty branch name", content: "branches:\n  - name: \"\"\n", wantErrors: []string{`"name" cannot be empty`}},
		{name: "duplicate branch name", content: "branches:\n  - name: main\n  - name: main\n", wantErrors: []string{`duplicate branch name "main"`}},
		{
			name:         "prerelease without stable branch",
			content:      "branches:\n  - name: rc\n    prerelease: true\n",
			wantWarnings: []string{"no stable branch defined"},
		},

		// Monorepo
		{
			name:         "monorepo as strings",
			content:      "monorepo:\n  - api\n  - web\n",
			wantErrors:   []string{`monorepo[0]: expected object with "name" key, got string "api"`, `monorepo[1]`},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "monorepo not an array",
			content:      `monorepo: "api"`,
			wantErrors:   []string{"monorepo: expected an array"},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "missing project name",
			content:      "monorepo:\n  - path: ./api/\n",
			wantErrors:   []string{`monorepo[0]: "name" key is required`},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "project name not a string",
			content:      "monorepo:\n  - name: 123\n    path: ./api/\n",
			wantErrors:   []string{`"name" must be a string`},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "empty project name",
			content:      "monorepo:\n  - name: \"\"\n    path: ./api/\n",
			wantErrors:   []string{`"name" cannot be empty`},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "duplicate project name",
			content:      "monorepo:\n  - name: api\n    path: ./api/\n  - name: api\n    path: ./api2/\n",
			wantErrors:   []string{`monorepo[1]: duplicate project name "api"`},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "path and paths are mutually exclusive",
			content:      "monorepo:\n  - name: api\n    path: ./api/\n    paths: [./lib/]\n",
			wantErrors:   []string{"mutually exclusive"},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "project without path",
			content:      "branches:\n  - name: main\nmonorepo:\n  - name: api\n",
			wantWarnings: []string{`project "api" has no path configured`},
		},
		{
			name:       "path not a string",
			content:    "branches:\n  - name: main\nmonorepo:\n  - name: api\n    path: 3\n",
			wantErrors: []string{`monorepo[0]: "path" must be a string, got int`},
		},
		{
			name:         "paths not an array",
			content:      "branches:\n  - name: main\nmonorepo:\n  - name: api\n    paths: ./api/\n",
			wantErrors:   []string{`monorepo[0]: "paths" must be an array of strings, got string`},
			wantWarnings: []string{"no path configured"},
		},
		{
			name:       "paths item not a string",
			content:    "branches:\n  - name: main\nmonorepo:\n  - name: api\n    paths: [./api/, 3]\n",
			wantErrors: []string{"monorepo[0].paths[1]: expected string, got int"},
		},

		// Rules
		{
			name:         "rules not an object",
			content:      "rules:\n  - feat\n",
			wantErrors:   []string{`rules: expected an object with "minor" and "patch" keys`},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "unknown release type",
			content:      "rules:\n  major: [feat]\n  minor: [fix]\n",
			wantErrors:   []string{`rules: unknown release type "major"`},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "release type not an array",
			content:      "rules:\n  minor: feat\n",
			wantErrors:   []string{"rules.minor: expected an array of commit types, got string"},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "commit type not a string",
			content:      "rules:\n  minor:\n    - 123\n",
			wantErrors:   []string{"rules.minor[0]: expected string, got int"},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:         "commit type mapped twice",
			content:      "rules:\n  minor: [feat]\n  patch: [feat]\n",
			wantErrors:   []string{`commit type "feat" is mapped to both "minor" and "patch"`},
			wantWarnings: []string{noBranchesWarning},
		},
		{
			name:       "unknown commit type with suggestion",
			content:    "branches:\n  - name: main\nrules:\n  minor: [feature]\n",
			wantErrors: []string{`rules.minor: unknown commit type "feature" (did you mean "feat"?)`},
		},
		{
			name:       "unknown commit type without suggestion",
			content:    "branches:\n  - name: main\nrules:\n  minor: [xyz123]\n",
			wantErrors: []string{`rules.minor: unknown commit type "xyz123"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validateConfigFile(writeFile(t, "config.yaml", tc.content))
			require.NoError(t, err)

			assertMessages(t, "error", tc.wantErrors, result.Errors)
			assertMessages(t, "warning", tc.wantWarnings, result.Warnings)
			assert.Equal(t, len(tc.wantErrors) > 0, result.HasErrors())
		})
	}

	t.Run("no suggestion for unrelated commit type", func(t *testing.T) {
		result, err := validateConfigFile(writeFile(t, "config.yaml", "branches:\n  - name: main\nrules:\n  minor: [xyz123]\n"))
		require.NoError(t, err)
		require.Len(t, result.Errors, 1)
		assert.NotContains(t, result.Errors[0], "did you mean")
	})
}

// assertMessages checks that got holds exactly one message per wanted substring, in the same order.
func assertMessages(t *testing.T, kind string, want, got []string) {
	t.Helper()

	if !assert.Len(t, got, len(want), "%ss: %q", kind, got) {
		return
	}

	for i := range want {
		assert.Contains(t, got[i], want[i], "%s %d", kind, i)
	}
}

func TestValidateConfigFile_UnreadableFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "missing file", path: filepath.Join(t.TempDir(), "missing.yaml"), wantErr: "reading config file"},
		{name: "invalid YAML", path: writeFile(t, "config.yaml", "branches:\n  - name: [invalid\n"), wantErr: "invalid YAML syntax"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateConfigFile(tc.path)
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
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
			assert.Equal(t, tc.expected, suggestCommitType(tc.input))
		})
	}
}

func TestValidateCmd(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantErr    error
		wantOutput []string
	}{
		{
			name:       "valid configuration",
			content:    "branches:\n  - name: main\nrules:\n  minor: [feat]\n  patch: [fix]\n",
			wantOutput: []string{"Configuration valid"},
		},
		{
			name:       "invalid configuration",
			content:    "branches:\n  - name: main\n  - name: main\n",
			wantErr:    ErrInvalidConfiguration,
			wantOutput: []string{`duplicate branch name "main"`, "1 error(s), 0 warning(s)"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, "config.yaml", tc.content)

			th := NewTestHelper(t)
			out, err := th.ExecuteCommand("validate", path)

			assert.ErrorIs(t, err, tc.wantErr)
			for _, want := range tc.wantOutput {
				assert.Contains(t, string(out), want)
			}
			assert.NotContains(t, string(out), "Usage:", "validation errors should not print the command usage")
		})
	}
}

func TestPrintValidationResult(t *testing.T) {
	tests := []struct {
		name   string
		result ValidationResult
		want   []string
	}{
		{
			name:   "valid",
			result: ValidationResult{},
			want:   []string{"Validating test.yaml...", "Configuration valid", "0 errors, 0 warnings"},
		},
		{
			name:   "errors",
			result: ValidationResult{Errors: []string{"error 1", "error 2"}},
			want:   []string{"error 1", "error 2", "2 error(s), 0 warning(s)"},
		},
		{
			name:   "warnings",
			result: ValidationResult{Warnings: []string{"warning 1"}},
			want:   []string{"warning 1", "0 error(s), 1 warning(s)"},
		},
		{
			name:   "errors and warnings",
			result: ValidationResult{Errors: []string{"error 1"}, Warnings: []string{"warning 1", "warning 2"}},
			want:   []string{"error 1", "warning 1", "warning 2", "1 error(s), 2 warning(s)"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewValidateCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			printValidationResult(cmd, "test.yaml", &tc.result)

			for _, want := range tc.want {
				assert.Contains(t, buf.String(), want)
			}
		})
	}
}

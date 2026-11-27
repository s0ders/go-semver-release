package ci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/s0ders/go-semver-release/v8/internal/semver"
)

// setGitHubOutput points GITHUB_OUTPUT to a new file of a temporary directory, for the duration of the test.
func setGitHubOutput(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "output")
	t.Setenv("GITHUB_OUTPUT", path)

	return path
}

func TestCI_GenerateGitHubOutput(t *testing.T) {
	version := &semver.Version{Major: 1, Minor: 2, Patch: 3}

	tests := []struct {
		name    string
		branch  string
		options []OptionFunc
		want    string
	}{
		{
			name:    "release",
			branch:  "main",
			options: []OptionFunc{WithNewRelease(true), WithTagPrefix("v")},
			want:    "\nMAIN_SEMVER=v1.2.3\nMAIN_NEW_RELEASE=true\n",
		},
		{
			name:    "release with project",
			branch:  "main",
			options: []OptionFunc{WithNewRelease(true), WithTagPrefix("v"), WithProject("foo")},
			want:    "\nMAIN_SEMVER=v1.2.3\nMAIN_NEW_RELEASE=true\nMAIN_PROJECT=foo\n",
		},
		{
			name:   "no release and no prefix",
			branch: "main",
			want:   "\nMAIN_SEMVER=1.2.3\nMAIN_NEW_RELEASE=false\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := setGitHubOutput(t)

			require.NoError(t, GenerateGitHubOutput(version, tc.branch, tc.options...))

			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(got))
		})
	}
}

func TestCI_GenerateGitHubOutput_AppendsToExistingOutput(t *testing.T) {
	path := setGitHubOutput(t)
	require.NoError(t, os.WriteFile(path, []byte("EXISTING=1"), 0o644))

	require.NoError(t, GenerateGitHubOutput(&semver.Version{Major: 1}, "main"))
	require.NoError(t, GenerateGitHubOutput(&semver.Version{Major: 2}, "rc"))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "EXISTING=1\nMAIN_SEMVER=1.0.0\nMAIN_NEW_RELEASE=false\n\nRC_SEMVER=2.0.0\nRC_NEW_RELEASE=false\n", string(got))
}

func TestCI_GenerateGitHubOutput_NoEnvVar(t *testing.T) {
	// t.Setenv registers the restoration of the variable, which is then unset for the test duration.
	t.Setenv("GITHUB_OUTPUT", "")
	require.NoError(t, os.Unsetenv("GITHUB_OUTPUT"))

	assert.NoError(t, GenerateGitHubOutput(&semver.Version{}, "main"), "should not have tried to generate an output")
}

func TestCI_GenerateGitHubOutput_ReadOnlyOutput(t *testing.T) {
	path := setGitHubOutput(t)
	require.NoError(t, os.WriteFile(path, nil, 0o444))

	err := GenerateGitHubOutput(&semver.Version{Major: 1, Minor: 2, Patch: 3}, "main")
	assert.ErrorContains(t, err, "opening ci file")
}

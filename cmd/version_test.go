package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v8/internal/appcontext"
)

func TestCmd_Version(t *testing.T) {
	assert := assert.New(t)
	actual := new(bytes.Buffer)
	ctx := appcontext.New()

	rootCmd := NewRootCommand(ctx)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	assert.NoError(err, "local command executed with error")

	output := actual.String()
	assert.Contains(output, "Version:")
	assert.Contains(output, "Commit:")
}

func TestGetVersionInfo_WithLdflags(t *testing.T) {
	// Save original values
	origVersion := cmdVersion
	origCommit := buildCommitHash
	defer func() {
		cmdVersion = origVersion
		buildCommitHash = origCommit
	}()

	// Set ldflags values
	cmdVersion = "v1.2.3"
	buildCommitHash = "abc123def456"

	version, commit, _ := getVersionInfo()

	assert.Equal(t, "v1.2.3", version)
	assert.Equal(t, "abc123def456", commit)
}

func TestGetVersionInfo_WithoutLdflags(t *testing.T) {
	// Save original values
	origVersion := cmdVersion
	origCommit := buildCommitHash
	defer func() {
		cmdVersion = origVersion
		buildCommitHash = origCommit
	}()

	// Clear ldflags values
	cmdVersion = ""
	buildCommitHash = ""

	version, commit, _ := getVersionInfo()

	// Should get values from debug.ReadBuildInfo() or return "unknown"
	assert.NotEmpty(t, version)
	assert.NotEmpty(t, commit)
}

func TestCmd_Version_OutputFormat(t *testing.T) {
	// Save original values
	origVersion := cmdVersion
	origBuild := buildNumber
	origCommit := buildCommitHash
	defer func() {
		cmdVersion = origVersion
		buildNumber = origBuild
		buildCommitHash = origCommit
	}()

	cmdVersion = "v2.0.0"
	buildNumber = "12345"
	buildCommitHash = "deadbeef"

	actual := new(bytes.Buffer)
	ctx := appcontext.New()

	rootCmd := NewRootCommand(ctx)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := actual.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	assert.Len(t, lines, 3)
	assert.Equal(t, "Version: v2.0.0", lines[0])
	assert.Equal(t, "Build: 12345", lines[1])
	assert.Equal(t, "Commit: deadbeef", lines[2])
}

func TestCmd_Version_NoBuildNumber(t *testing.T) {
	// Save original values
	origVersion := cmdVersion
	origBuild := buildNumber
	origCommit := buildCommitHash
	defer func() {
		cmdVersion = origVersion
		buildNumber = origBuild
		buildCommitHash = origCommit
	}()

	cmdVersion = "v3.0.0"
	buildNumber = ""
	buildCommitHash = "abc123"

	actual := new(bytes.Buffer)
	ctx := appcontext.New()

	rootCmd := NewRootCommand(ctx)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := actual.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should only have 2 lines (no Build line)
	assert.Len(t, lines, 2)
	assert.Equal(t, "Version: v3.0.0", lines[0])
	assert.Equal(t, "Commit: abc123", lines[1])
}

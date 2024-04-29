package ci

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/s0ders/go-semver-release/v2/internal/semver"
	"github.com/stretchr/testify/assert"
)

func TestCI_New(t *testing.T) {
	assert := assert.New(t)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	output := New(logger)

	assert.Equal(logger, output.logger, "logger should be equal")
}

func TestCI_GenerateGitHub(t *testing.T) {
	assert := assert.New(t)

	outputDir, err := os.MkdirTemp("./", "output-*")
	assert.NoError(err, "should create temp directory")

	defer func(path string) {
		err := os.RemoveAll(outputDir)
		assert.NoError(err, "should have been able to remove temporary directory")
	}(outputDir)

	outputFilePath := filepath.Join(outputDir, "output")

	outputFile, err := os.OpenFile(outputFilePath, os.O_RDONLY|os.O_CREATE, 0o666)
	assert.NoError(err, "should have been able to create output file")

	defer func() {
		err := outputFile.Close()
		assert.NoError(err, "should have been able to close output file")
	}()

	outputPath := filepath.Join(outputDir, "output")

	err = os.Setenv("GITHUB_OUTPUT", outputPath)
	assert.NoError(err, "should have been able to set GITHUB_OUTPUT")

	defer func() {
		err := os.Unsetenv("GITHUB_OUTPUT")
		assert.NoError(err, "should have been able to unset GITHUB_OUTPUT")
	}()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	output := New(logger)

	version, err := semver.New(1, 2, 3, "")
	assert.NoError(err, "should have been able to create version")

	err = output.GenerateGitHub("v", version, true)
	assert.NoError(err, "should have been able to generate GitHub output")

	writtenOutput, err := os.ReadFile(outputPath)
	assert.NoError(err, "should have been able to read output file")

	want := "\nSEMVER=v1.2.3\nNEW_RELEASE=true\n"
	got := string(writtenOutput)

	assert.Equal(want, got, "output should match")
}

func TestCI_NoOutputEnvVar(t *testing.T) {
	assert := assert.New(t)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	output := New(logger)

	err := output.GenerateGitHub("", nil, false)

	assert.NoError(err, "should not have tried to generate an output")
}

func TestCI_ReadOnlyOutput(t *testing.T) {
	assert := assert.New(t)

	outputDir, err := os.MkdirTemp("./", "output-*")
	assert.NoError(err, "should create temp directory")

	defer func(path string) {
		err := os.RemoveAll(outputDir)
		assert.NoError(err, "should have been able to remove temporary directory")
	}(outputDir)

	outputFilePath := filepath.Join(outputDir, "output")

	outputFile, err := os.OpenFile(outputFilePath, os.O_RDONLY|os.O_CREATE, 0o444)
	assert.NoError(err, "should have been able to create output file")

	defer func(outputFile *os.File) {
		err := outputFile.Close()
		assert.NoError(err, "should have been able to close output file")
	}(outputFile)

	outputPath := filepath.Join(outputDir, "output")

	err = os.Setenv("GITHUB_OUTPUT", outputPath)
	assert.NoError(err, "should have been able to set GITHUB_OUTPUT")

	defer func() {
		err := os.Unsetenv("GITHUB_OUTPUT")
		assert.NoError(err, "should have been able to unset GITHUB_OUTPUT")
	}()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	output := New(logger)

	version, err := semver.New(1, 2, 3, "")
	assert.NoError(err, "should have been able to create version")

	err = output.GenerateGitHub("v", version, true)
	assert.Error(err, "should have failed since output file is readonly")
}

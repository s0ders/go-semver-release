package ci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s0ders/go-semver-release/v2/internal/semver"
	"github.com/stretchr/testify/assert"
)

func TestCI_GenerateGitHub(t *testing.T) {
	assert := assert.New(t)

	outputDir, err := os.MkdirTemp("./", "output-*")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}

	defer func(path string) {
		err = os.RemoveAll(outputDir)
		if err != nil {
			t.Fatalf("failed to remove temporary directory: %s", err)
		}
	}(outputDir)

	outputFilePath := filepath.Join(outputDir, "output")

	outputFile, err := os.OpenFile(outputFilePath, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatalf("failed to create output file: %s", err)
	}

	defer func() {
		err = outputFile.Close()
		if err != nil {
			t.Fatalf("failed to create temporary directory: %s", err)
		}
	}()

	outputPath := filepath.Join(outputDir, "output")

	err = os.Setenv("GITHUB_OUTPUT", outputPath)
	if err != nil {
		t.Fatalf("failed to set GITHUB_OUTPUT env. var.: %s", err)
	}

	defer func() {
		err = os.Unsetenv("GITHUB_OUTPUT")
		if err != nil {
			t.Fatalf("failed unset GITHUB_OUTPUT env. var.: %s", err)
		}
	}()

	version := &semver.Semver{Major: 1, Minor: 2, Patch: 3}

	err = GenerateGitHubOutput("v", version, true)
	if err != nil {
		t.Fatalf("failed to create github output: %s", err)
	}

	writtenOutput, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %s", err)
	}

	want := "\nSEMVER=v1.2.3\nNEW_RELEASE=true\n"
	got := string(writtenOutput)

	assert.Equal(want, got, "output should match")
}

func TestCI_NoOutputEnvVar(t *testing.T) {
	assert := assert.New(t)

	err := GenerateGitHubOutput("", nil, false)
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

	version := &semver.Semver{Major: 1, Minor: 2, Patch: 3}

	err = GenerateGitHubOutput("v", version, true)
	assert.Error(err, "should have failed since output file is readonly")
}

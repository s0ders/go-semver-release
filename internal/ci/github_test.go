package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v5/internal/semver"
)

func TestCI_GenerateGitHub_HappyScenario(t *testing.T) {
	assert := assertion.New(t)

	err := setup()
	checkErr(t, "setting up test", err)

	defer func() {
		err = teardown()
		checkErr(t, "tearing down test", err)
	}()

	version := &semver.Semver{Major: 1, Minor: 2, Patch: 3}

	err = GenerateGitHubOutput(version, "main", WithNewRelease(true), WithTagPrefix("v"))
	if err != nil {
		t.Fatalf("creating github output: %s", err)
	}

	outputPath := os.Getenv("GITHUB_OUTPUT")

	writtenOutput, err := os.ReadFile(outputPath)
	checkErr(t, "reading output file", err)

	want := "\nMAIN_SEMVER=v1.2.3\nMAIN_NEW_RELEASE=true\n"
	got := string(writtenOutput)

	assert.Equal(want, got, "output should match")
}

func TestCI_GenerateGitHub_HappyScenarioWithProject(t *testing.T) {
	assert := assertion.New(t)

	err := setup()
	checkErr(t, "setting up test", err)

	defer func() {
		err = teardown()
		checkErr(t, "tearing down test", err)
	}()

	version := &semver.Semver{Major: 1, Minor: 2, Patch: 3}

	err = GenerateGitHubOutput(version, "main", WithNewRelease(true), WithTagPrefix("v"), WithProject("foo"))
	if err != nil {
		t.Fatalf("creating github output: %s", err)
	}

	outputPath := os.Getenv("GITHUB_OUTPUT")

	writtenOutput, err := os.ReadFile(outputPath)
	checkErr(t, "reading output file", err)

	want := "\nMAIN_SEMVER=v1.2.3\nMAIN_NEW_RELEASE=true\nMAIN_PROJECT=foo\n"
	got := string(writtenOutput)

	assert.Equal(want, got, "output should match")
}

func TestCI_GenerateGitHub_NoEnvVar(t *testing.T) {
	assert := assertion.New(t)

	err := GenerateGitHubOutput(&semver.Semver{}, "main")
	assert.NoError(err, "should not have tried to generate an output")
}

func TestCI_GenerateGitHub_ReadOnlyOutput(t *testing.T) {
	assert := assertion.New(t)

	err := setup()
	checkErr(t, "setting up test", err)

	defer func() {
		err = teardown()
		checkErr(t, "tearing down test", err)
	}()

	filePath := os.Getenv("GITHUB_OUTPUT")

	err = os.Chmod(filePath, 0444)
	checkErr(t, "changing output file permissions", err)

	version := &semver.Semver{Major: 1, Minor: 2, Patch: 3}

	err = GenerateGitHubOutput(version, "main")
	assert.Error(err, "should have failed since output file is readonly")
}

func setup() error {
	dirPath, err := os.MkdirTemp("", "output-*")
	if err != nil {
		return fmt.Errorf("creating temporary directory: %w", err)
	}

	filePath := filepath.Join(dirPath, "output")

	err = os.WriteFile(filePath, []byte(""), 0644)
	if err != nil {
		return fmt.Errorf("setting up output file: %w", err)
	}

	err = os.Setenv("GITHUB_OUTPUT", filePath)
	if err != nil {
		return fmt.Errorf("setting GITHUB_OUTPUT env. var.: %w", err)
	}

	return nil
}

func teardown() error {
	path, ok := os.LookupEnv("GITHUB_OUTPUT")
	if !ok {
		return nil
	}

	dirPath := filepath.Dir(path)

	err := os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("removing directory: %w", err)
	}

	err = os.Unsetenv("GITHUB_OUTPUT")
	if err != nil {
		return fmt.Errorf("unsetting GITHUB_OUTPUT env. var.: %w", err)
	}

	return nil
}

func checkErr(t *testing.T, msg string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", msg, err.Error())
	}
}

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/s0ders/go-semver-release/v2/internal/rule"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v2/internal/tag"
)

type cmdOutput struct {
	Message     string `json:"message"`
	NewVersion  string `json:"new-version"`
	NextVersion string `json:"next-version"`
	NewRelease  bool   `json:"new-release"`
}

var sampleCommitFile = "not_a_real_file.txt"

func TestLocalCmd_Release(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"pattern": "master"}})

	commits := []string{
		"fix",      // 0.0.1
		"feat!",    // 1.0.0 (breaking change)
		"feat",     // 1.1.0
		"fix",      // 1.1.1
		"fix",      // 1.1.2
		"chores",   // 1.1.2
		"refactor", // 1.1.2
		"test",     // 1.1.2
		"ci",       // 1.1.2
		"feat",     // 1.2.0
		"perf",     // 1.2.1
		"revert",   // 1.2.2
		"style",    // 1.2.2
	}

	flags := map[string]string{
		"tag-prefix": "v",
	}

	buf := new(bytes.Buffer)

	repository, path := setup(t, buf, flags, commits)

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "1.2.2"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		NewVersion: expectedVersion,
		NewRelease: true,
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(buf.Bytes(), &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	exists, err := tag.Exists(repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists, "tag not found")
}

func TestLocalCmd_ReleaseWithBuildMetadata(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"pattern": "master"}})

	metadata := "foobarbaz"
	buf := new(bytes.Buffer)

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
		"feat",  // 1.1.0
		"fix",   // 1.1.1
	}

	flags := map[string]string{
		"build-metadata": metadata,
	}

	repository, path := setup(t, buf, flags, commits)

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "1.1.1" + "+" + metadata
	expectedOut := cmdOutput{
		Message:    "new release found",
		NewVersion: expectedVersion,
		NewRelease: true,
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(buf.Bytes(), &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	exists, err := tag.Exists(repository, expectedVersion)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists)
}

func TestLocalCmd_Prerelease(t *testing.T) {
	assert := assertion.New(t)

	prereleaseID := "alpha"
	configSetBranches([]map[string]string{{"pattern": "master", "prerelease": "true", "prerelease-identifier": prereleaseID}})

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
		"feat",  // 1.1.0
		"fix",   // 1.1.1
	}

	buf := new(bytes.Buffer)

	repository, path := setup(t, buf, nil, commits)

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "1.1.1" + "-" + prereleaseID
	expectedOut := cmdOutput{
		Message:    "new release found",
		NewVersion: expectedVersion,
		NewRelease: true,
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(buf.Bytes(), &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	exists, err := tag.Exists(repository, expectedVersion)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists)
}

func TestLocalCmd_ReleaseWithDryRun(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"pattern": "master"}})

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
	}

	flags := map[string]string{
		"dry-run": "true",
	}

	actual := new(bytes.Buffer)

	repository, path := setup(t, actual, flags, commits)

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "1.0.0"
	expectedTag := expectedVersion
	expectedOut := cmdOutput{
		Message:     "new release found, dry-run is enabled",
		NextVersion: expectedVersion,
		NewRelease:  true,
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(actual.Bytes(), &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	exists, err := tag.Exists(repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(false, exists, "tag should not exist, running in dry-run mode")
}

func TestLocalCmd_NoRelease(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"pattern": "master"}})

	actual := new(bytes.Buffer)

	_, path := setup(t, actual, nil, []string{})

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedOut := cmdOutput{
		Message:    "no new release",
		NewRelease: false,
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(actual.Bytes(), &actualOut)
	checkErr(t, err, "removing temporary directory")

	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")
}

func TestLocalCmd_ReadOnlyGitHubOutput(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"pattern": "master"}})

	outputDir, err := os.MkdirTemp("./", "output-*")
	if err != nil {
		t.Fatalf("creating temporary GitHub directory: %s", err)
	}

	defer func() {
		err = os.RemoveAll(outputDir)
		if err != nil {
			t.Fatalf("removing temporary GitHub directory: %s", err)
		}
	}()

	outputFilePath := filepath.Join(outputDir, "output")

	outputFile, err := os.OpenFile(outputFilePath, os.O_RDONLY|os.O_CREATE, 0o444)
	if err != nil {
		t.Fatalf("creating GitHub output file: %s", err)
	}

	defer func() {
		err = outputFile.Close()
		if err != nil {
			t.Fatalf("closing GitHub output file: %s", err)
		}
	}()

	outputPath := filepath.Join(outputDir, "output")

	err = os.Setenv("GITHUB_OUTPUT", outputPath)
	if err != nil {
		t.Fatalf("setting GITHUB_OUTPUT env. var.: %s", err)
	}

	defer func() {
		err = os.Unsetenv("GITHUB_OUTPUT")
		if err != nil {
			t.Fatalf("unsetting GITHUB_OUTPUT env. var.: %s", err)
		}
	}()

	_, repositoryPath, err := sampleRepository()
	if err != nil {
		t.Fatalf("creating sample repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		assert.NoError(err, "removing sample repository")
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", repositoryPath})

	err = resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to write GitHub output to read-only file")
}

func TestLocalCmd_InvalidRepositoryPath(t *testing.T) {
	assert := assertion.New(t)

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", "./does/not/exist"})

	err := resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to open inexisting Git repository")
}

func TestLocalCmd_InvalidArmoredKeyPath(t *testing.T) {
	assert := assertion.New(t)

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", ".", "--gpg-key-path", "./fake.asc"})

	err := resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to open inexisting armored GPG key")
}

func TestLocalCmd_InvalidArmoredKeyContent(t *testing.T) {
	assert := assertion.New(t)

	gpgKeyDir, err := os.MkdirTemp("./", "gpg-*")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}

	defer func(path string) {
		err = os.RemoveAll(gpgKeyDir)
		if err != nil {
			t.Fatalf("failed to remove temporary directory: %s", err)
		}
	}(gpgKeyDir)

	keyFilePath := filepath.Join(gpgKeyDir, "key.asc")

	keyFile, err := os.Create(keyFilePath)
	if err != nil {
		t.Fatalf("failed to create output file: %s", err)
	}

	defer func() {
		err = keyFile.Close()
		if err != nil {
			t.Fatalf("failed to create temporary directory: %s", err)
		}
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", ".", "--gpg-key-path", keyFilePath})

	err = resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to read armored key ring from empty file")
}

func TestLocalCmd_RepositoryWithNoHead(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"pattern": "master"}})

	tempDirPath, err := os.MkdirTemp("", "tag-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}

	defer func() {
		err = os.RemoveAll(tempDirPath)
		if err != nil {
			t.Fatalf("removing temp dir: %v", err)
		}
	}()

	_, err = git.PlainInit(tempDirPath, false)
	if err != nil {
		t.Fatalf("initializing repository: %v", err)
	}

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", tempDirPath})

	err = resetFlags(localCmd)
	assert.NoError(err, "resetting command flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to compute new semver of repository with no HEAD")
}

func TestLocalCmd_InvalidCustomRules(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"pattern": "master"}})
	configSetRules(map[string][]string{"minor": {"feat"}, "patch": {"feat", "fix"}})

	_, repositoryPath, err := sampleRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(repositoryPath)
		checkErr(t, err, "removing sample repository")
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", repositoryPath})

	err = resetFlags(localCmd)
	checkErr(t, err, "resetting flags")

	err = rootCmd.Execute()
	assert.ErrorIs(err, rule.ErrDuplicateReleaseRule, "should have failed parsing invalid custom rule")
}

func TestLocalCmd_CustomRules(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"pattern": "master"}})
	configSetRules(map[string][]string{"minor": {"feat", "fix"}})

	commits := []string{
		"fix",  // 0.1.0 (with custom rule)
		"feat", // 0.2.0
	}

	buf := new(bytes.Buffer)

	repository, path := setup(t, buf, nil, commits)

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "0.2.0"
	expectedTag := expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		NewVersion: expectedVersion,
		NewRelease: true,
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(buf.Bytes(), &actualOut)
	assert.NoError(err, "failed to unmarshal json")

	// Check that the JSON output is correct
	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	// Check that the tag was actually created on the repository
	exists, err := tag.Exists(repository, expectedTag)
	assert.NoError(err, "failed to check if tag exists")

	assert.Equal(true, exists, "tag should exist")
}

func sampleRepository() (*git.Repository, string, error) {
	dir, err := os.MkdirTemp("", "localcmd-test-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	repository, err := git.PlainInit(dir, false)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize git repository: %s", err)
	}

	tempFilePath := filepath.Join(dir, sampleCommitFile)

	commitFile, err := os.Create(tempFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create sample commit file: %s", err)
	}

	defer func() {
		_ = commitFile.Close()
	}()

	_, err = commitFile.Write([]byte("first line"))
	if err != nil {
		return nil, "", err
	}

	worktree, err := repository.Worktree()
	if err != nil {
		return nil, "", fmt.Errorf("could not get worktree: %w", err)
	}

	_, err = worktree.Add(sampleCommitFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to add sample commit file to worktree: %w", err)
	}

	_, err = worktree.Commit("first commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create commit: %w", err)
	}
	return repository, dir, nil
}

func sampleCommit(repository *git.Repository, repositoryPath string, commitType string) (err error) {
	worktree, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("could not get worktree: %w", err)
	}

	commitFilePath := filepath.Join(repositoryPath, sampleCommitFile)

	err = os.WriteFile(commitFilePath, []byte("data to modify file"), 0o666)
	if err != nil {
		return fmt.Errorf("failed to open sample commit file: %w", err)
	}

	_, err = worktree.Add(sampleCommitFile)
	if err != nil {
		return fmt.Errorf("failed to add sample commit file to worktree: %w", err)
	}

	commitMessage := fmt.Sprintf("%s: this a test commit", commitType)

	_, err = worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return nil
}

func setup(t *testing.T, buf io.Writer, flags map[string]string, commits []string) (*git.Repository, string) {
	repository, path, err := sampleRepository()
	checkErr(t, err, "creating sample repository")

	if len(commits) != 0 {
		for _, commit := range commits {
			err = sampleCommit(repository, path, commit)
			checkErr(t, err, "creating sample commit")
		}
	}

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"local", path})

	err = resetFlags(localCmd)
	checkErr(t, err, "resetting flags")

	for k, v := range flags {
		err = localCmd.Flags().Set(k, v)
		checkErr(t, err, "setting "+k)
	}

	err = rootCmd.Execute()
	checkErr(t, err, "executing command")

	return repository, path
}

func resetFlags(cmd *cobra.Command) (err error) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		err = f.Value.Set(f.DefValue)
		if err != nil {
			return
		}
	})

	return err
}

func checkErr(t *testing.T, err error, message string) {
	if err != nil {
		t.Fatalf("%s: %s", message, err)
	}
}

func configSetBranches(branches []map[string]string) {
	viper.Set("branches", branches)
}

func configSetRules(rules map[string][]string) {
	viper.Set("rules", rules)
}

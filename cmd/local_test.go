package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/s0ders/go-semver-release/v2/internal/tag"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
)

type cmdOutput struct {
	Message     string `json:"message"`
	NewVersion  string `json:"new-version"`
	NextVersion string `json:"next-version"`
	NewRelease  bool   `json:"new-release"`
}

var sampleCommitFile = "not_a_real_file.txt"

func TestLocalCmd_Release(t *testing.T) {
	assert := assert.New(t)

	// Setting up sample Git repository
	repository, repositoryPath, err := sampleRepository()
	assert.NoError(err, "failed to create sample repository")

	defer func() {
		err = os.RemoveAll(repositoryPath)
		assert.NoError(err, "failed to remove repository")
	}()

	commitTypes := []string{
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

	for _, commitType := range commitTypes {
		err = sampleCommit(repository, repositoryPath, commitType)
		assert.NoError(err, "failed to create sample commit")
	}

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", repositoryPath})

	err = resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = localCmd.Flags().Set("release-branch", "main")
	assert.NoError(err, "failed to set --release-branch")

	err = localCmd.Flags().Set("tag-prefix", "v")
	assert.NoError(err, "failed to set --tag-prefix")

	err = rootCmd.Execute()
	assert.NoError(err, "local command executed with error")

	expectedVersion := "1.2.2"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		NewVersion: expectedVersion,
		NewRelease: true,
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(actual.Bytes(), &actualOut)
	assert.NoError(err, "failed to unmarshal json")

	// Check that the JSON output is correct
	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	// Check that the tag was actually created on the repository
	exists, err := tag.TagExists(repository, expectedTag)
	assert.NoError(err, "failed to check if tag exists")

	assert.Equal(true, exists, "tag should exist")
}

func TestLocalCmd_ReleaseWithDryRun(t *testing.T) {
	assert := assert.New(t)

	// Setting up sample Git repository
	repository, repositoryPath, err := sampleRepository()
	assert.NoError(err, "failed to create sample repository")

	defer func() {
		err = os.RemoveAll(repositoryPath)
		assert.NoError(err, "failed to remove repository")
	}()

	commitTypes := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
	}

	for _, commitType := range commitTypes {
		err = sampleCommit(repository, repositoryPath, commitType)
		assert.NoError(err, "failed to create sample commit")
	}

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", repositoryPath})

	err = resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = localCmd.Flags().Set("dry-run", "true")
	assert.NoError(err, "failed to set --dry-run")

	err = localCmd.Flags().Set("release-branch", "main")
	assert.NoError(err, "failed to set --release-branch")

	err = localCmd.Flags().Set("tag-prefix", "v")
	assert.NoError(err, "failed to set --tag-prefix")

	err = rootCmd.Execute()
	assert.NoError(err, "local command executed with error")

	expectedVersion := "1.0.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:     "new release found, dry-run is enabled",
		NextVersion: expectedVersion,
		NewRelease:  true,
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(actual.Bytes(), &actualOut)
	assert.NoError(err, "failed to unmarshal json")

	// Check that the JSON output is correct
	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	// Check that the tag was actually created on the repository
	exists, err := tag.TagExists(repository, expectedTag)
	assert.NoError(err, "failed to check if tag exists")

	assert.Equal(false, exists, "tag should not exist, running in dry-run mode")
}

func TestLocalCmd_NoRelease(t *testing.T) {
	assert := assert.New(t)

	// Setting up sample Git repository
	_, repositoryPath, err := sampleRepository()
	assert.NoError(err, "failed to create sample repository")

	defer func() {
		err = os.RemoveAll(repositoryPath)
		assert.NoError(err, "failed to remove repository")
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", repositoryPath})

	err = resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = localCmd.Flags().Set("release-branch", "main")
	assert.NoError(err, "failed to set --release-branch")

	err = localCmd.Flags().Set("tag-prefix", "v")
	assert.NoError(err, "failed to set --tag-prefix")

	err = rootCmd.Execute()
	assert.NoError(err, "local command executed with error")

	expectedOut := cmdOutput{
		Message:    "no new release",
		NewRelease: false,
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(actual.Bytes(), &actualOut)
	assert.NoError(err, "failed to unmarshal json")

	// Check that the JSON output is correct
	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")
}

func TestLocalCmd_Verbose(t *testing.T) {
	assert := assert.New(t)

	// Setting up sample Git repository
	_, repositoryPath, err := sampleRepository()
	assert.NoError(err, "failed to create sample repository")

	defer func() {
		err = os.RemoveAll(repositoryPath)
		assert.NoError(err, "failed to remove repository")
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", repositoryPath})

	err = resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = rootCmd.PersistentFlags().Set("verbose", "true")
	assert.NoError(err, "failed to set --verbose")

	err = localCmd.Flags().Set("release-branch", "main")
	assert.NoError(err, "failed to set --release-branch")

	err = localCmd.Flags().Set("tag-prefix", "v")
	assert.NoError(err, "failed to set --tag-prefix")

	err = rootCmd.Execute()
	assert.NoError(err, "local command executed with error")
}

func TestLocalCmd_CustomRules(t *testing.T) {
	assert := assert.New(t)

	// Setting up sample Git repository
	repository, repositoryPath, err := sampleRepository()
	assert.NoError(err, "failed to create sample repository")

	defer func() {
		err = os.RemoveAll(repositoryPath)
		assert.NoError(err, "failed to remove repository")
	}()

	commitTypes := []string{
		"fix",  // 0.1.0 (with custom rules)
		"feat", // 0.2.0
	}

	for _, commitType := range commitTypes {
		err = sampleCommit(repository, repositoryPath, commitType)
		assert.NoError(err, "failed to create sample commit")
	}

	tempRulesDir, err := os.MkdirTemp("", "rules-*")
	assert.NoError(err, "failed to create temp. dir.")

	defer func() {
		err = os.RemoveAll(tempRulesDir)
		assert.NoError(err, "failed to remove temp. dir.")
	}()

	customRulesPath := filepath.Join(tempRulesDir, "custom.json")

	customRules, err := os.Create(customRulesPath)
	assert.NoError(err, "failed to create empty rule file")

	customRulesJSON := `
{
    "rules": [
        {"type": "feat",   "release": "minor"},
        {"type": "fix",    "release": "minor"}
    ]
}
`

	_, err = customRules.Write([]byte(customRulesJSON))
	assert.NoError(err, "failed to write empty rule file")

	defer func() {
		err = customRules.Close()
		assert.NoError(err, "failed to close empty rule file")
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", repositoryPath})

	err = resetFlags(localCmd)
	assert.NoError(err, "failed to reset localCmd flags")

	err = localCmd.Flags().Set("rules-path", customRulesPath)
	assert.NoError(err, "failed to set --rules-path")

	err = rootCmd.Execute()
	assert.NoError(err, "local command executed with error")

	expectedVersion := "0.2.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		NewVersion: expectedVersion,
		NewRelease: true,
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(actual.Bytes(), &actualOut)
	assert.NoError(err, "failed to unmarshal json")

	// Check that the JSON output is correct
	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	// Check that the tag was actually created on the repository
	exists, err := tag.TagExists(repository, expectedTag)
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

// sampleCommit modifies the sample commit files with the same line of text
// and creates a new commit to the given repository with the given commit type.
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

func resetFlags(cmd *cobra.Command) (err error) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		err = f.Value.Set(f.DefValue)
		if err != nil {
			return
		}
	})

	return err
}

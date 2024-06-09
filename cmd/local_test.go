package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v2/internal/gittest"
	"github.com/s0ders/go-semver-release/v2/internal/rule"
	"github.com/s0ders/go-semver-release/v2/internal/tag"
)

type cmdOutput struct {
	Message    string `json:"message"`
	Branch     string `json:"branch"`
	Version    string `json:"version"`
	NewRelease bool   `json:"new-release"`
}

func TestLocalCmd_Release(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"name": "master"}})

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
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(buf.Bytes(), &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")

	exists, err := tag.Exists(repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists, "tag not found")
}

func TestLocalCmd_MultiBranchRelease(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{
		{"name": "master"},
		{"name": "rc", "prerelease": "true"},
	})

	buf := new(bytes.Buffer)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = testRepository.Remove()
		checkErr(t, err, "removing repository")
	}()

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"local", testRepository.Path})

	// Create commits on master
	masterCommits := []string{
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

	if len(masterCommits) != 0 {
		for _, commit := range masterCommits {
			_, err = testRepository.AddCommit(commit)
			checkErr(t, err, "creating sample commit on master")
		}
	}

	// Create branch rc and its commits
	head, err := testRepository.Head()
	checkErr(t, err, "fetching head")

	rcRef := plumbing.NewHashReference("refs/heads/rc", head.Hash())

	err = testRepository.Storer.SetReference(rcRef)
	checkErr(t, err, "creating branch rc")

	worktree, err := testRepository.Worktree()
	checkErr(t, err, "fetching worktree")

	branchCoOpts := git.CheckoutOptions{
		Branch: rcRef.Name(),
		Force:  true,
	}

	err = worktree.Checkout(&branchCoOpts)
	checkErr(t, err, "checking out to branch rc")

	rcCommits := []string{
		"feat!", // 2.0.0
		"feat",  // 2.1.0
		"perf",  // 2.1.1
	}

	for _, commit := range rcCommits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit on rc")
	}

	// Executing command
	err = rootCmd.Execute()
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    "new release found",
			Version:    "1.2.2",
			NewRelease: true,
			Branch:     "master",
		},
		{
			Message:    "new release found",
			Version:    "2.1.1-rc",
			NewRelease: true,
			Branch:     "rc",
		},
	}

	scanner := bufio.NewScanner(buf)

	for scanner.Scan() {
		rawOutput := scanner.Bytes()

		actualOutput := cmdOutput{}

		err = json.Unmarshal(rawOutput, &actualOutput)
		checkErr(t, err, "unmarshalling output")

		assert.Equal(expectedOutputs[i], actualOutput)
		i++
	}

	err = scanner.Err()
	checkErr(t, err, "scanning error")
}

func TestLocalCmd_ReleaseWithBuildMetadata(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"name": "master"}})

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
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
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

	configSetBranches([]map[string]string{{"name": "master", "prerelease": "true"}})

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

	expectedVersion := "1.1.1-master"
	expectedOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
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

	configSetBranches([]map[string]string{{"name": "master"}})

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
		Message:    "dry-run enabled, new release found",
		Branch:     "master",
		Version:    expectedVersion,
		NewRelease: true,
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

	configSetBranches([]map[string]string{{"name": "master"}})

	actual := new(bytes.Buffer)

	_, path := setup(t, actual, nil, []string{})

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedOut := cmdOutput{
		Message:    "no new release",
		NewRelease: false,
		Branch:     "master",
		Version:    "0.0.0",
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(actual.Bytes(), &actualOut)
	checkErr(t, err, "removing temporary directory")

	assert.Equal(expectedOut, actualOut, "localCmd output should be equal")
}

func TestLocalCmd_ReadOnlyGitHubOutput(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"name": "master"}})

	outputDir, err := os.MkdirTemp("./", "output-*")
	checkErr(t, err, "creating output directory")

	defer func() {
		err = os.RemoveAll(outputDir)
		checkErr(t, err, "removing output directory")
	}()

	outputFilePath := filepath.Join(outputDir, "output")

	outputFile, err := os.OpenFile(outputFilePath, os.O_RDONLY|os.O_CREATE, 0o444)
	checkErr(t, err, "creating output file")

	defer func() {
		err = outputFile.Close()
		checkErr(t, err, "closing output file")
	}()

	outputPath := filepath.Join(outputDir, "output")

	err = os.Setenv("GITHUB_OUTPUT", outputPath)
	checkErr(t, err, "setting GITHUB_OUTPUT environment variable")

	defer func() {
		err = os.Unsetenv("GITHUB_OUTPUT")
		checkErr(t, err, "unsetting GITHUB_OUTPUT environment variable")
	}()

	testRepository, err := gittest.NewRepository()
	if err != nil {
		t.Fatalf("creating sample repository: %s", err)
	}

	defer func() {
		err = testRepository.Remove()
		assert.NoError(err, "removing sample repository")
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", testRepository.Path})

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

	defer func() {
		err = os.RemoveAll(gpgKeyDir)
		if err != nil {
			t.Fatalf("failed to remove temporary directory: %s", err)
		}
	}()

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

	configSetBranches([]map[string]string{{"name": "master"}})

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

	configSetBranches([]map[string]string{{"name": "master"}})
	configSetRules(map[string][]string{"minor": {"feat"}, "patch": {"feat", "fix"}})

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing sample repository")
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"local", testRepository.Path})

	err = resetFlags(localCmd)
	checkErr(t, err, "resetting flags")

	err = rootCmd.Execute()
	assert.ErrorIs(err, rule.ErrDuplicateReleaseRule, "should have failed parsing invalid custom rule")
}

func TestLocalCmd_CustomRules(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"name": "master"}})
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
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
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

func setup(t *testing.T, buf io.Writer, flags map[string]string, commits []string) (*git.Repository, string) {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	if len(commits) != 0 {
		for _, commit := range commits {
			_, err = testRepository.AddCommit(commit)
			checkErr(t, err, "creating sample commit")
		}
	}

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"local", testRepository.Path})

	err = resetFlags(localCmd)
	checkErr(t, err, "resetting flags")

	for k, v := range flags {
		err = localCmd.Flags().Set(k, v)
		checkErr(t, err, "setting "+k)
	}

	err = rootCmd.Execute()
	checkErr(t, err, "executing command")

	return testRepository.Repository, testRepository.Path
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

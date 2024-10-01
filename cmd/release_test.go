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
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v5/internal/gittest"
	"github.com/s0ders/go-semver-release/v5/internal/rule"
	"github.com/s0ders/go-semver-release/v5/internal/tag"
)

type cmdOutput struct {
	Message    string `json:"message"`
	Branch     string `json:"branch"`
	Version    string `json:"version"`
	Project    string `json:"project"`
	NewRelease bool   `json:"new-release"`
}

func TestReleaseCmd_ConfigureRules_DefaultRules(t *testing.T) {
	assert := assertion.New(t)

	rules, err := configureRules()
	checkErr(t, err, "configuring rules")

	assert.Equal(rule.Default, rules)
}

func TestReleaseCmd_ConfigureBranches_NoBranches(t *testing.T) {
	assert := assertion.New(t)

	_, err := configureBranches()
	assert.ErrorContains(err, "missing branches key in configuration")
}

func TestReleaseCmd_ConfigureProjects_NoProjects(t *testing.T) {
	assert := assertion.New(t)

	_, err := configureProjects()
	assert.ErrorContains(err, "missing projects key in configuration")
}

func TestReleaseCmd_SemVerConfigFile(t *testing.T) {
	assert := assertion.New(t)

	taggerName := "My CI Robot"
	taggerEmail := "my-robot@release.ci"

	// Create configuration file
	configurationContent := []byte(`
git-name: ` + taggerName + `
git-email: ` + taggerEmail + `
tag-prefix: v
branches:
  - name: master
  - name: alpha
    prerelease: true
rules:
  minor:
    - feat
  patch:
    - fix
    - perf
    - revert
`)

	configurationFileDirectory, err := os.MkdirTemp("", "*")
	checkErr(t, err, "creating configuration file")

	defer func() {
		err = os.RemoveAll(configurationFileDirectory)
		checkErr(t, err, "removing configuration file")
	}()

	configurationFilePath := filepath.Join(configurationFileDirectory, "config.yml")

	err = os.WriteFile(configurationFilePath, configurationContent, 0644)
	checkErr(t, err, "writing configuration file")

	// Create test repository
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

	alphaCommits := []string{
		"fix",  // 1.2.3-alpha
		"feat", // 1.3.0-alpha
	}

	buf := new(bytes.Buffer)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	for _, commit := range masterCommits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit")
	}

	// Creating alpha branch and associated commits
	err = testRepository.CheckoutBranch("alpha")
	checkErr(t, err, "checking out alpha branch")

	for _, commit := range alphaCommits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit")
	}

	// Executing command
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	err = rootCmd.PersistentFlags().Set("config", configurationFilePath)
	checkErr(t, err, "setting root command config flag")

	rootCmd.SetArgs([]string{"release", testRepository.Path})

	err = resetFlags(releaseCmd)
	checkErr(t, err, "resetting local command flags")

	err = rootCmd.Execute()
	checkErr(t, err, "executing command")

	expectedMasterVersion := "1.2.2"
	expectedMasterTag := "v" + expectedMasterVersion
	expectedMasterOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedMasterVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualMasterOut := cmdOutput{}

	expectedAlphaVersion := "1.3.0-alpha"
	expectedAlphaTag := "v" + expectedAlphaVersion
	expectedAlphaOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedAlphaVersion,
		NewRelease: true,
		Branch:     "alpha",
	}
	actualAlphaOut := cmdOutput{}

	outputs := make([]string, 0, 2)

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		outputs = append(outputs, scanner.Text())
	}

	// Checking master
	err = json.Unmarshal([]byte(outputs[0]), &actualMasterOut)
	checkErr(t, err, "unmarshalling master output")

	assert.Equal(expectedMasterOut, actualMasterOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedMasterTag)
	checkErr(t, err, "checking if master tag exists")

	assert.Equal(true, exists, "master tag not found")

	expectedTagRef, err := testRepository.Tag(expectedMasterTag)
	checkErr(t, err, "getting master tag ref")

	expectedTagObj, err := testRepository.TagObject(expectedTagRef.Hash())
	checkErr(t, err, "getting master tag object")

	assert.Equal(taggerName, expectedTagObj.Tagger.Name)
	assert.Equal(taggerEmail, expectedTagObj.Tagger.Email)

	// Checking alpha
	err = json.Unmarshal([]byte(outputs[1]), &actualAlphaOut)
	checkErr(t, err, "unmarshalling alpha output")

	assert.Equal(expectedAlphaOut, actualAlphaOut, "releaseCmd output should be equal")

	exists, err = tag.Exists(testRepository.Repository, expectedAlphaTag)
	checkErr(t, err, "checking if alpha tag exists")

	assert.Equal(true, exists, "alpha tag not found")
}

func TestReleaseCmd_Release(t *testing.T) {
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

	buf := new(bytes.Buffer)

	repository, path := setup(t, buf, commits)

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

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists, "tag not found")
}

func TestReleaseCmd_RemoteRelease(t *testing.T) {
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

	buf := new(bytes.Buffer)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	for _, commit := range commits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit")
	}

	err = resetPersistentFlags(rootCmd)
	checkErr(t, err, "resetting root command flags")

	err = resetFlags(releaseCmd)
	checkErr(t, err, "resetting release command flags")

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err = rootCmd.PersistentFlags().Set("remote", "true")
	checkErr(t, err, "setting remote flag")

	err = rootCmd.PersistentFlags().Set("remote-name", "origin")
	checkErr(t, err, "setting remote-name flag")

	err = rootCmd.PersistentFlags().Set("access-token", "")
	checkErr(t, err, "setting access-token flag")

	rootCmd.SetArgs([]string{"release", testRepository.Path})

	err = rootCmd.Execute()
	checkErr(t, err, "executing command")

	expectedVersion := "1.2.2"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(buf.Bytes(), &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists, "tag not found")
}

func TestReleaseCmd_MultiBranchRelease(t *testing.T) {
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

	err = resetPersistentFlags(rootCmd)
	checkErr(t, err, "resetting root command flags")

	err = resetFlags(releaseCmd)
	checkErr(t, err, "resetting release command flags")

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"release", testRepository.Path})

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

func TestReleaseCmd_ReleaseWithBuildMetadata(t *testing.T) {
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

	repository, path := setup(t, buf, commits, WithReleaseFlags(flags))

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "1.1.1" + "+" + metadata
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

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists)
}

func TestReleaseCmd_Prerelease(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"name": "master", "prerelease": "true"}})

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
		"feat",  // 1.1.0
		"fix",   // 1.1.1
	}

	buf := new(bytes.Buffer)

	repository, path := setup(t, buf, commits)

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "1.1.1-master"
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

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists)
}

func TestReleaseCmd_ReleaseWithDryRun(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"name": "master"}})

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
	}

	releaseFlags := map[string]string{
		"dry-run": "true",
	}

	actual := new(bytes.Buffer)

	repository, path := setup(t, actual, commits, WithReleaseFlags(releaseFlags))

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "1.0.0"
	expectedTag := expectedVersion
	expectedOut := cmdOutput{
		Message:    "dry-run enabled, next release found",
		Branch:     "master",
		Version:    expectedVersion,
		NewRelease: true,
	}
	actualOut := cmdOutput{}

	err := json.Unmarshal(actual.Bytes(), &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(false, exists, "tag should not exist, running in dry-run mode")
}

func TestReleaseCmd_NoRelease(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"name": "master"}})

	actual := new(bytes.Buffer)

	_, path := setup(t, actual, []string{})

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

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")
}

func TestReleaseCmd_ReadOnlyGitHubOutput(t *testing.T) {
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
	rootCmd.SetArgs([]string{"release", testRepository.Path})

	err = resetFlags(releaseCmd)
	assert.NoError(err, "failed to reset releaseCmd flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to write GitHub output to read-only file")
}

func TestReleaseCmd_InvalidRepositoryPath(t *testing.T) {
	assert := assertion.New(t)

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"release", "./does/not/exist"})

	err := resetFlags(releaseCmd)
	assert.NoError(err, "failed to reset releaseCmd flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to open inexisting Git repository")
}

func TestReleaseCmd_InvalidArmoredKeyPath(t *testing.T) {
	assert := assertion.New(t)

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"release", ".", "--gpg-key-path", "./fake.asc"})

	err := resetFlags(releaseCmd)
	assert.NoError(err, "failed to reset releaseCmd flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to open inexisting armored GPG key")
}

func TestReleaseCmd_InvalidArmoredKeyContent(t *testing.T) {
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
	rootCmd.SetArgs([]string{"release", ".", "--gpg-key-path", keyFilePath})

	err = resetFlags(releaseCmd)
	assert.NoError(err, "failed to reset releaseCmd flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to read armored key ring from empty file")
}

func TestReleaseCmd_RepositoryWithNoHead(t *testing.T) {
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
	rootCmd.SetArgs([]string{"release", tempDirPath})

	err = resetFlags(releaseCmd)
	assert.NoError(err, "resetting command flags")

	err = rootCmd.Execute()
	assert.Error(err, "should have failed trying to compute new semver of repository with no HEAD")
}

func TestReleaseCmd_InvalidCustomRules(t *testing.T) {
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
	rootCmd.SetArgs([]string{"release", testRepository.Path})

	err = resetFlags(releaseCmd)
	checkErr(t, err, "resetting flags")

	err = rootCmd.Execute()
	assert.ErrorIs(err, rule.ErrDuplicateReleaseRule, "should have failed parsing invalid custom rule")
}

func TestReleaseCmd_CustomRules(t *testing.T) {
	assert := assertion.New(t)

	configSetBranches([]map[string]string{{"name": "master"}})
	configSetRules(map[string][]string{"minor": {"feat", "fix"}})

	commits := []string{
		"fix",  // 0.1.0 (with custom rule)
		"feat", // 0.2.0
	}

	buf := new(bytes.Buffer)

	repository, path := setup(t, buf, commits)

	defer func() {
		err := os.RemoveAll(path)
		checkErr(t, err, "removing repository")
	}()

	expectedVersion := "0.2.0"
	expectedTag := "v" + expectedVersion
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
	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	// Check that the tag was actually created on the repository
	exists, err := tag.Exists(repository, expectedTag)
	assert.NoError(err, "failed to check if tag exists")

	assert.Equal(true, exists, "tag should exist")
}

func TestReleaseCmd_Monorepo(t *testing.T) {
	assert := assertion.New(t)

	configSetMonorepo()
	configSetBranches([]map[string]string{{"name": "master"}})
	configSetProjects([]map[string]string{{"name": "foo", "path": "foo"}, {"name": "bar", "path": "bar"}})
	configSetRules(map[string][]string{"minor": {"feat"}, "patch": {"fix"}})

	buf := new(bytes.Buffer)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = testRepository.Remove()
		checkErr(t, err, "removing repository")
	}()

	err = resetPersistentFlags(rootCmd)
	checkErr(t, err, "resetting root command flags")

	err = resetFlags(releaseCmd)
	checkErr(t, err, "resetting release command flags")

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"release", testRepository.Path})

	// "foo" commits
	_, err = testRepository.AddCommitWithSpecificFile("feat", "./foo/foo.txt")
	checkErr(t, err, "adding commit")
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./foo/foo2.txt")
	checkErr(t, err, "adding commit")

	// "bar" commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./bar/foo.txt")
	checkErr(t, err, "adding commit")
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/foo2.txt")
	checkErr(t, err, "adding commit")
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/foo2.txt")
	checkErr(t, err, "adding commit")

	// Executing command
	err = rootCmd.Execute()
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    "new release found",
			Version:    "0.1.1",
			NewRelease: true,
			Branch:     "master",
			Project:    "foo",
		},
		{
			Message:    "new release found",
			Version:    "1.0.2",
			NewRelease: true,
			Branch:     "master",
			Project:    "bar",
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

type CommandFlagOptions func()

func WithReleaseFlags(flags map[string]string) CommandFlagOptions {
	return func() {
		for k, v := range flags {
			_ = releaseCmd.Flags().Set(k, v)
		}
	}
}

func setup(t *testing.T, buf io.Writer, commits []string, commandFlagOptions ...CommandFlagOptions) (*git.Repository, string) {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	if len(commits) != 0 {
		for _, commit := range commits {
			_, err = testRepository.AddCommit(commit)
			checkErr(t, err, "creating sample commit")
		}
	}

	err = resetPersistentFlags(rootCmd)
	checkErr(t, err, "resetting root command flags")

	err = resetFlags(releaseCmd)
	checkErr(t, err, "resetting release command flags")

	for _, option := range commandFlagOptions {
		option()
	}

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"release", testRepository.Path})

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

func resetPersistentFlags(cmd *cobra.Command) (err error) {
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		err = f.Value.Set(f.DefValue)
		if err != nil {
			return
		}
	})

	return err
}

func checkErr(t *testing.T, err error, message string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", message, err)
	}
}

func configSetBranches(branches []map[string]string) {
	viperInstance.Set("branches", branches)
}

func configSetRules(rules map[string][]string) {
	viperInstance.Set("rules", rules)
}

func configSetMonorepo() {
	viperInstance.Set("monorepo", true)
}

func configSetProjects(projects []map[string]string) {
	viperInstance.Set("projects", projects)
}

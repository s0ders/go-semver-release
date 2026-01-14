package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v6/internal/appcontext"
	"github.com/s0ders/go-semver-release/v6/internal/gittest"
	"github.com/s0ders/go-semver-release/v6/internal/tag"
)

type cmdOutput struct {
	Message    string `json:"message"`
	Branch     string `json:"branch"`
	Version    string `json:"version"`
	Project    string `json:"project"`
	NewRelease bool   `json:"new-release"`
}

func TestReleaseCmd_ConfigurationAsEnvironmentVariable(t *testing.T) {
	assert := assertion.New(t)
	th := NewTestHelper(t)

	err := th.SetFlag(BranchesConfiguration, `[{"name": "master"}]`)
	checkErr(t, err, "setting branches configuration")

	testRepository := NewTestRepository(t, []string{})

	accessToken := "secret"
	err = os.Setenv("GO_SEMVER_RELEASE_ACCESS_TOKEN", accessToken)
	checkErr(t, err, "setting environment variable")

	_, err = th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	assert.Equal(accessToken, th.Ctx.AccessToken, "access token flag value should be equal to environment variable value")
}

// TODO: add monorepo config key

func TestReleaseCmd_ConfigurationAsFile(t *testing.T) {
	assert := assertion.New(t)

	taggerName := "My CI Robot"
	taggerEmail := "my-robot@release.ci"

	// Create a configuration file
	cfgContent := []byte(`
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

	cfgFileDirectory, err := os.MkdirTemp("", "*")
	checkErr(t, err, "creating configuration file")

	defer func() {
		err = os.RemoveAll(cfgFileDirectory)
		checkErr(t, err, "removing configuration file")
	}()

	cfgFilePath := filepath.Join(cfgFileDirectory, "config.yml")

	err = os.WriteFile(cfgFilePath, cfgContent, 0o644)
	checkErr(t, err, "writing configuration file")

	// Create a test repository

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	masterCommits := []string{
		"fix",
		"feat",
		"feat!", // 1.0.0
		"fix",
	}

	alphaCommits := []string{
		"fix",
		"feat",
		// TODO: process and tag each branch individually before processing the next one, this way, prerelease branch see "main" tags, the correct version below should be 2.0.0-alpha not 1.0.0-alpha because alpha should have seen the 1.0.0 tag
		"feat!", // 1.0.0-alpha
		"fix",
	}

	for _, commit := range masterCommits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit")
	}

	// Creating the alpha branch and associated commits
	err = testRepository.CheckoutBranch("alpha")
	checkErr(t, err, "checking out alpha branch")

	for _, commit := range alphaCommits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit")
	}

	th := NewTestHelper(t)
	err = th.SetFlag("config", cfgFilePath)
	checkErr(t, err, "setting flags")

	releaseOutput, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "running release command")

	expectedMasterVersion := "1.0.0"
	expectedMasterTag := "v" + expectedMasterVersion
	expectedAlphaVersion := "1.0.0-alpha.1"
	expectedAlphaTag := "v" + expectedAlphaVersion

	expectedOutputs := []cmdOutput{
		{
			Message:    "new release found",
			Version:    expectedAlphaVersion,
			NewRelease: true,
			Branch:     "alpha",
		},
		{
			Message:    "new release found",
			Version:    expectedMasterVersion,
			NewRelease: true,
			Branch:     "master",
		},
	}
	actualOutput := cmdOutput{}

	outputs := make([]string, 0, 2)

	scanner := bufio.NewScanner(bytes.NewReader(releaseOutput))
	for scanner.Scan() {
		outputs = append(outputs, scanner.Text())
	}

	// Checking master
	err = json.Unmarshal([]byte(outputs[0]), &actualOutput)
	checkErr(t, err, "unmarshalling master output")

	assert.Contains(expectedOutputs, actualOutput, "releaseCmd output should be equal")

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
	err = json.Unmarshal([]byte(outputs[1]), &actualOutput)
	checkErr(t, err, "unmarshalling alpha output")

	assert.Contains(expectedOutputs, actualOutput, "releaseCmd output should be equal")

	exists, err = tag.Exists(testRepository.Repository, expectedAlphaTag)
	checkErr(t, err, "checking if alpha tag exists")

	assert.Equal(true, exists, "alpha tag not found")
}

func TestReleaseCmd_ConfigurationAsFlags(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",   // 0.1.0
		"feat!", // 1.0.0 (breaking change)
		"feat",
		"fix",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "master"}]`,
		RulesConfiguration:    `{"minor": ["feat", "fix"]}`,
	})
	checkErr(t, err, "setting flags")

	output, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "1.0.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(output, &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	checkErr(t, err, "checking if master tag exists")

	assert.Equal(true, exists, "master tag not found")
}

func TestReleaseCmd_LocalRelease(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
		"feat",
		"fix",
		"fix",
		"chores",
		"refactor",
		"test",
		"ci",
		"feat",
		"perf",
		"revert",
		"style",
	}

	testRepository := NewTestRepository(t, commits)

	defer func() {
		err := os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	th := NewTestHelper(t)
	err := th.SetFlag(BranchesConfiguration, `[{"name": "master"}]`)
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "1.0.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(out, &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists, "tag not found")
}

func TestReleaseCmd_RemoteRelease(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
		"feat",
		"fix",
		"fix",
		"chores",
		"refactor",
		"test",
		"ci",
		"feat",
		"perf",
		"revert",
		"style",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BranchesConfiguration:    `[{"name": "master"}]`,
		RemoteNameConfiguration:  "origin",
		AccessTokenConfiguration: "",
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "1.0.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(out, &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists, "tag not found")
}

func TestReleaseCmd_MultiBranchRelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	hash, err := testRepository.AddCommit("feat!")
	checkErr(t, err, "creating sample commit on master")

	err = testRepository.AddTag("1.0.0", hash)
	checkErr(t, err, "creating sample tag")

	// TODO: add tag 1.0.0 on master
	// Create commits on master
	masterCommits := []string{
		"fix",   // 1.0.1
		"feat!", // 2.0.0 (breaking change)
		"feat",
		"fix",
		"fix",
		"chores",
		"refactor",
		"test",
		"ci",
		"feat",
		"perf",
		"revert",
		"style",
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
		"feat",
		"feat!", // 2.0.0-rc
		"perf",
	}

	for _, commit := range rcCommits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit on rc")
	}

	th := NewTestHelper(t)
	err = th.SetFlag(BranchesConfiguration, `[{"name": "master"}, {"name": "rc", "prerelease": true}]`)
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    "new release found",
			Version:    "2.0.0",
			NewRelease: true,
			Branch:     "master",
		},
		{
			Message:    "new release found",
			Version:    "2.0.0-rc.1",
			NewRelease: true,
			Branch:     "rc",
		},
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))

	for scanner.Scan() {
		rawOutput := scanner.Bytes()

		actualOutput := cmdOutput{}

		err = json.Unmarshal(rawOutput, &actualOutput)
		checkErr(t, err, "unmarshalling output")

		assert.Contains(expectedOutputs, actualOutput)
		i++
	}

	err = scanner.Err()
	checkErr(t, err, "scanning error")
}

func TestReleaseCmd_ReleaseWithMetadata(t *testing.T) {
	assert := assertion.New(t)
	metadata := "foobarbaz"

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
		"feat",
		"fix",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BuildMetadataConfiguration: metadata,
		BranchesConfiguration:      `[{"name": "master"}]`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "1.0.0" + "+" + metadata
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(out, &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists)
}

func TestReleaseCmd_PrereleaseBranch(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
		"feat",
		"fix",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlag(BranchesConfiguration, `[{"name": "master", "prerelease": true}]`)
	checkErr(t, err, "setting flags")
	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "1.0.0-master.1"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(out, &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(true, exists)
}

func TestReleaseCmd_DryRunRelease(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",   // 0.0.1
		"feat!", // 1.0.0 (breaking change)
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "master"}]`,
		DryRunConfiguration:   `true`,
	})
	checkErr(t, err, "setting flags")
	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "1.0.0"
	expectedTag := expectedVersion
	expectedOut := cmdOutput{
		Message:    "dry-run enabled, next release found",
		Branch:     "master",
		Version:    expectedVersion,
		NewRelease: true,
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(out, &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	checkErr(t, err, "checking if tag exists")

	assert.Equal(false, exists, "tag should not exist, running in dry-run mode")
}

func TestReleaseCmd_ReleaseNoNewVersion(t *testing.T) {
	assert := assertion.New(t)

	testRepository := NewTestRepository(t, []string{})

	th := NewTestHelper(t)
	err := th.SetFlag(BranchesConfiguration, `[{"name": "master"}]`)
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedOut := cmdOutput{
		Message:    "no new release",
		NewRelease: false,
		Branch:     "master",
		Version:    "0.0.0",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(out, &actualOut)
	checkErr(t, err, "removing temporary directory")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")
}

func TestReleaseCmd_ReadOnlyGitHubOutput(t *testing.T) {
	assert := assertion.New(t)

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

	testRepository := NewTestRepository(t, []string{})

	th := NewTestHelper(t)
	err = th.SetFlag(BranchesConfiguration, `[{"name": "master"}]`)
	checkErr(t, err, "setting flags")

	_, err = th.ExecuteCommand("release", testRepository.Path)
	assert.ErrorContains(err, "opening ci file", "should have failed trying to write GitHub output to read-only file")
}

func TestReleaseCmd_InvalidRepositoryPath(t *testing.T) {
	assert := assertion.New(t)

	th := NewTestHelper(t)
	_ = th.SetFlag(BranchesConfiguration, `[{"name": "master"}]`)
	_, err := th.ExecuteCommand("release", "./does/not/exist")

	assert.ErrorContains(err, "cloning Git repository", "should have failed trying to open inexisting Git repository")
}

func TestReleaseCmd_RepositoryWithNoHead(t *testing.T) {
	assert := assertion.New(t)

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

	th := NewTestHelper(t)
	err = th.SetFlag(BranchesConfiguration, `[{"name": "master"}]`)
	checkErr(t, err, "setting flags")

	_, err = th.ExecuteCommand("release", tempDirPath)

	assert.Error(err, "should have failed trying to compute new semver of repository with no HEAD")
}

func TestReleaseCmd_CustomRules(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",  // 0.1.0 (with custom rule)
		"feat", // 0.1.0 -> single bump per release
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "master"}]`,
		RulesConfiguration:    `{"minor": ["feat", "fix"]}`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "0.1.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    "new release found",
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "master",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(out, &actualOut)
	assert.NoError(err, "failed to unmarshal json")

	// Check that the JSON output is correct
	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	// Check that the tag was actually created on the repository
	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	assert.NoError(err, "failed to check if tag exists")

	assert.Equal(true, exists, "tag should exist")
}

func TestReleaseCmd_Monorepo(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = testRepository.Remove()
		checkErr(t, err, "removing repository")
	}()

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

	th := NewTestHelper(t)
	err = th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "master"}]`,
		MonorepoConfiguration: `[{"name": "foo", "path": "foo"}, {"name": "bar", "path": "bar"}]`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    "new release found",
			Version:    "0.1.0",
			NewRelease: true,
			Branch:     "master",
			Project:    "foo",
		},
		{
			Message:    "new release found",
			Version:    "1.0.0",
			NewRelease: true,
			Branch:     "master",
			Project:    "bar",
		},
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))

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

func TestReleaseCmd_Monorepo_MixedRelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = testRepository.Remove()
		checkErr(t, err, "removing repository")
	}()

	// "bar" commits, "foo" has no applicable commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./bar/foo.txt")
	checkErr(t, err, "adding commit")
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/foo2.txt")
	checkErr(t, err, "adding commit")
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/foo2.txt")
	checkErr(t, err, "adding commit")

	th := NewTestHelper(t)
	err = th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "master"}]`,
		MonorepoConfiguration: `[{"name": "foo", "path": "foo"}, {"name": "bar", "path": "bar"}]`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    "no new release",
			Version:    "0.0.0",
			NewRelease: false,
			Branch:     "master",
			Project:    "foo",
		},
		{
			Message:    "new release found",
			Version:    "1.0.0",
			NewRelease: true,
			Branch:     "master",
			Project:    "bar",
		},
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))

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
	assert.Equal(len(expectedOutputs), i)
}

func TestReleaseCmd_Monorepo_ProjectWithPaths(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = testRepository.Remove()
		checkErr(t, err, "removing repository")
	}()

	// "bar" commits, "foo" has no applicable commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./bar/foo.txt")
	checkErr(t, err, "adding commit")
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar2/foo2.txt")
	checkErr(t, err, "adding commit")
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar2/foo2.txt")
	checkErr(t, err, "adding commit")

	th := NewTestHelper(t)
	err = th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "master"}]`,
		MonorepoConfiguration: `[{"name": "bar", "paths": ["bar", "bar2"]}]`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    "new release found",
			Version:    "1.0.0",
			NewRelease: true,
			Branch:     "master",
			Project:    "bar",
		},
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))

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
	assert.Equal(len(expectedOutputs), i)
}

func TestReleaseCmd_Monorepo_ExclusivePathAndPaths(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = testRepository.Remove()
		checkErr(t, err, "removing repository")
	}()

	// "bar" commits, "foo" has no applicable commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./bar/foo.txt")
	checkErr(t, err, "adding commit")

	th := NewTestHelper(t)
	err = th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "master"}]`,
		MonorepoConfiguration: `[{"name": "bar", "path": "bar", "paths": ["./bar/", "./bar2/"]}]`,
	})

	// TODO: refine to target specific monorepo.ErrExclusive...
	assert.Error(err, "should have failed trying to set exclusive path and paths")
}

func TestReleaseCmd_InvalidArmoredKeyPath(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

	ctx.GPGKeyPath = "./does/not/exist"

	_, err := configureGPGKey(ctx)

	assert.ErrorContains(err, "reading armored key", "should have failed trying to open non existing armored GPG key")
}

func TestReleaseCmd_InvalidArmoredKeyContent(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

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

	ctx.GPGKeyPath = keyFilePath

	_, err = configureGPGKey(ctx)
	assert.ErrorContains(err, "loading armored key", "should have failed trying to read armored key ring from empty file")
}

// Test utilities
func NewTestRepository(t *testing.T, commits []string) *gittest.TestRepository {
	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	for _, commit := range commits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit")
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(testRepository.Path)
	})

	return testRepository
}

type TestHelper struct {
	Ctx *appcontext.AppContext
	Cmd *cobra.Command
}

// NewTestHelper creates a new TestHelper with a fresh AppContext and Command
func NewTestHelper(t *testing.T) *TestHelper {
	ctx := &appcontext.AppContext{
		Viper: viper.New(),
	}
	cmd := NewRootCommand(ctx)
	return &TestHelper{
		Ctx: ctx,
		Cmd: cmd,
	}
}

// SetFlag sets a flag value for the test
func (th *TestHelper) SetFlag(name string, value string) error {
	return th.Cmd.PersistentFlags().Set(name, value)
}

// SetFlags sets multiple flag values for the test
func (th *TestHelper) SetFlags(flags map[string]string) error {
	for name, value := range flags {
		if err := th.SetFlag(name, value); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteCommand executes the command with given arguments
func (th *TestHelper) ExecuteCommand(args ...string) ([]byte, error) {
	th.Cmd.SetArgs(args)
	return ExecuteCommand(th.Cmd, args...)
}

// ExecuteCommand is a helper function to execute a command and capture its output
func ExecuteCommand(cmd *cobra.Command, args ...string) ([]byte, error) {
	output := new(bytes.Buffer)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return output.Bytes(), err
}

func checkErr(t *testing.T, err error, message string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", message, err)
	}
}

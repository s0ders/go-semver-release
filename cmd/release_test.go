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
	"github.com/s0ders/go-semver-release/v6/internal/branch"
	"github.com/s0ders/go-semver-release/v6/internal/gittest"
	"github.com/s0ders/go-semver-release/v6/internal/monorepo"
	"github.com/s0ders/go-semver-release/v6/internal/rule"
	"github.com/s0ders/go-semver-release/v6/internal/tag"
)

type cmdOutput struct {
	Message    string `json:"message"`
	Branch     string `json:"branch"`
	Version    string `json:"version"`
	Project    string `json:"project"`
	NewRelease bool   `json:"new-release"`
	Error      string `json:"error"`
}

var (
	taggerName  string = "My CI Robot"
	taggerEmail string = "my-robot@release.ci"
)

func TestReleaseCmd_ConfigurationAsEnvironmentVariable(t *testing.T) {
	assert := assertion.New(t)
	th := NewTestHelper(t)

	err := th.SetFlag(BranchesConfiguration, `[{"name": "main"}]`)
	checkErr(t, err, "setting branches configuration")

	testRepository := NewTestRepository(t, []string{})

	accessToken := "secret"
	err = os.Setenv("GO_SEMVER_RELEASE_ACCESS_TOKEN", accessToken)
	checkErr(t, err, "setting environment variable")

	_, err = th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	assert.Equal(accessToken, th.Ctx.AccessTokenFlag, "access token flag value should be equal to environment variable value")
}

func TestReleaseCmd_ConfigurationAsFile(t *testing.T) {
	// Create configuration file
	cfgContent := []byte(`
git-name: ` + taggerName + `
git-email: ` + taggerEmail + `
tag-prefix: v
branches:
  - name: main
  - name: beta
    prerelease: true
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

	// Create test steps
	type e = []*cmdOutput
	steps := []gittest.Step{
		gittest.NewCommitStep("main", "fix"),
		gittest.NewCallbackStep("", e{
			{
				Message:    MessageNewRelease,
				Branch:     "main",
				Version:    "0.1.0",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "beta",
				Version:    "0.0.0",
				Project:    "",
				NewRelease: false,
				Error:      "remote branch \"refs/remotes/origin/beta\" not found: reference not found",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "alpha",
				Version:    "0.0.0",
				Project:    "",
				NewRelease: false,
				Error:      "remote branch \"refs/remotes/origin/alpha\" not found: reference not found",
			},
		}),

		gittest.NewCommitStep("main", "feat"),
		gittest.NewCommitStep("main", "fix"),
		gittest.NewCallbackStep("", e{
			{
				Message:    MessageNewRelease,
				Branch:     "main",
				Version:    "0.2.0",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "beta",
				Version:    "0.0.0",
				Project:    "",
				NewRelease: false,
				Error:      "remote branch \"refs/remotes/origin/beta\" not found: reference not found",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "alpha",
				Version:    "0.0.0",
				Project:    "",
				NewRelease: false,
				Error:      "remote branch \"refs/remotes/origin/alpha\" not found: reference not found",
			},
		}),

		gittest.NewCommitStep("main", "fix"),
		gittest.NewCommitStep("beta", "feat!"),
		gittest.NewCallbackStep("", e{
			{
				Message:    MessageNewRelease,
				Branch:     "main",
				Version:    "0.2.1",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
			{
				Message:    MessageNewRelease,
				Branch:     "beta",
				Version:    "1.0.0-beta.1",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "alpha",
				Version:    "0.0.0",
				Project:    "",
				NewRelease: false,
				Error:      "remote branch \"refs/remotes/origin/alpha\" not found: reference not found",
			},
		}),

		gittest.NewCommitStep("main", "chores"),
		gittest.NewCommitStep("beta", "refactor"),
		gittest.NewCommitStep("main", "test"),
		gittest.NewCommitStep("beta", "ci"),
		gittest.NewCallbackStep("", e{
			{
				Message:    MessageNoNewRelease,
				Branch:     "main",
				Version:    "0.2.1",
				Project:    "",
				NewRelease: false,
				Error:      "",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "beta",
				Version:    "1.0.0-beta.1",
				Project:    "",
				NewRelease: false,
				Error:      "",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "alpha",
				Version:    "0.0.0",
				Project:    "",
				NewRelease: false,
				Error:      "remote branch \"refs/remotes/origin/alpha\" not found: reference not found",
			},
		}),

		gittest.NewCommitStep("alpha", "perf"),
		gittest.NewCallbackStep("main", e{
			{
				Message:    MessageNoNewRelease,
				Branch:     "main",
				Version:    "0.2.1",
				Project:    "",
				NewRelease: false,
				Error:      "",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "beta",
				Version:    "1.0.0-beta.1",
				Project:    "",
				NewRelease: false,
				Error:      "",
			},
			{
				Message:    MessageNewRelease,
				Branch:     "alpha",
				Version:    "1.0.1-alpha.1",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
		}),

		gittest.NewCommitStep("main", "revert"),
		gittest.NewCommitStep("main", "style"),
		gittest.NewCommitStep("alpha", "feat"),
		gittest.NewCommitStep("beta", "fix"),
		gittest.NewCallbackStep("", e{
			{
				Message:    MessageNewRelease,
				Branch:     "main",
				Version:    "0.2.2",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
			{
				Message:    MessageNewRelease,
				Branch:     "beta",
				Version:    "1.0.0-beta.2",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
			{
				Message:    MessageNewRelease,
				Branch:     "alpha",
				Version:    "1.1.0-alpha.1",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
		}),

		gittest.NewMergeStep("main", "beta", false),
		gittest.NewCallbackStep("", e{
			{
				Message:    MessageNewRelease,
				Branch:     "main",
				Version:    "1.0.0",
				Project:    "",
				NewRelease: true,
				Error:      "",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "beta",
				Version:    "1.0.0-beta.2",
				Project:    "",
				NewRelease: false,
				Error:      "",
			},
			{
				Message:    MessageNoNewRelease,
				Branch:     "alpha",
				Version:    "1.1.0-alpha.1",
				Project:    "",
				NewRelease: false,
				Error:      "",
			},
		}),
	}

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	th := NewTestHelper(t)
	err = th.SetFlag("config", cfgFilePath)
	checkErr(t, err, "setting flags")

	var i int
	err = gittest.ExecuteSteps(testRepository, steps, func(expected e) error {
		releaseOutput, err := th.ExecuteCommand("release", testRepository.Path)
		checkErr(t, err, "running release command")

		checkRelease(t, testRepository, i, releaseOutput, expected)
		i++
		return nil
	})
	checkErr(t, err, "execute test steps")
}

func TestReleaseCmd_ConfigurationAsFlags(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",
		"feat!",
		"feat",
		"fix",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "main"}]`,
		RulesConfiguration:    `{"minor": ["feat", "fix"]}`,
	})
	checkErr(t, err, "setting flags")

	output, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "0.1.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    MessageNewRelease,
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "main",
	}
	actualOut := cmdOutput{}

	err = json.Unmarshal(output, &actualOut)
	checkErr(t, err, "unmarshalling output")

	assert.Equal(expectedOut, actualOut, "releaseCmd output should be equal")

	exists, err := tag.Exists(testRepository.Repository, expectedTag)
	checkErr(t, err, "checking if main tag exists")

	assert.Equal(true, exists, "main tag not found")
}

func TestReleaseCmd_LocalRelease(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",
		"feat!",
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
	err := th.SetFlag(BranchesConfiguration, `[{"name": "main"}]`)
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "0.1.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    MessageNewRelease,
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "main",
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
		"fix",
		"feat!",
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
		BranchesConfiguration:    `[{"name": "main"}]`,
		RemoteNameConfiguration:  "origin",
		AccessTokenConfiguration: "",
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "0.1.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    MessageNewRelease,
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "main",
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

	// Create commits on main
	mainCommits := []string{
		"fix",
		"feat!",
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

	if len(mainCommits) != 0 {
		for _, commit := range mainCommits {
			_, err = testRepository.AddCommit(commit)
			checkErr(t, err, "creating sample commit on main")
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
		"feat!",
		"feat",
		"perf",
	}

	for _, commit := range rcCommits {
		_, err = testRepository.AddCommit(commit)
		checkErr(t, err, "creating sample commit on rc")
	}

	th := NewTestHelper(t)
	err = th.SetFlag(BranchesConfiguration, `[{"name": "main"}, {"name": "rc", "prerelease": true}]`)
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    MessageNewRelease,
			Version:    "0.1.0",
			NewRelease: true,
			Branch:     "main",
		},
		{
			Message:    MessageNewRelease,
			Version:    "1.0.0-rc.1",
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

		assert.Equal(expectedOutputs[i], actualOutput)
		i++
	}

	err = scanner.Err()
	checkErr(t, err, "scanning error")
}

func TestReleaseCmd_ReleaseWithMetadata(t *testing.T) {
	assert := assertion.New(t)
	metadata := "foobarbaz"

	commits := []string{
		"fix",
		"feat!",
		"feat",
		"fix",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BuildMetadataConfiguration: metadata,
		BranchesConfiguration:      `[{"name": "main"}]`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "0.1.0" + "+" + metadata
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    MessageNewRelease,
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "main",
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
		"fix",
		"feat!",
		"feat",
		"fix",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlag(BranchesConfiguration, `[{"name": "main", "prerelease": true}]`)
	checkErr(t, err, "setting flags")
	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "0.1.0-main.1"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    MessageNewRelease,
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "main",
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
		"fix",
		"feat!",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "main"}]`,
		DryRunConfiguration:   `true`,
	})
	checkErr(t, err, "setting flags")
	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "0.1.0"
	expectedTag := expectedVersion
	expectedOut := cmdOutput{
		Message:    MessageDryRun,
		Branch:     "main",
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
	err := th.SetFlag(BranchesConfiguration, `[{"name": "main"}]`)
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedOut := cmdOutput{
		Message:    MessageNoNewRelease,
		NewRelease: false,
		Branch:     "main",
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
	err = th.SetFlag(BranchesConfiguration, `[{"name": "main"}]`)
	checkErr(t, err, "setting flags")

	_, err = th.ExecuteCommand("release", testRepository.Path)
	assert.ErrorContains(err, "opening ci file", "should have failed trying to write GitHub output to read-only file")
}

func TestReleaseCmd_InvalidRepositoryPath(t *testing.T) {
	assert := assertion.New(t)

	th := NewTestHelper(t)
	_ = th.SetFlag(BranchesConfiguration, `[{"name": "main"}]`)
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
	err = th.SetFlag(BranchesConfiguration, `[{"name": "main"}]`)
	checkErr(t, err, "setting flags")

	_, err = th.ExecuteCommand("release", tempDirPath)

	assert.Error(err, "should have failed trying to compute new semver of repository with no HEAD")
}

func TestReleaseCmd_CustomRules(t *testing.T) {
	assert := assertion.New(t)

	commits := []string{
		"fix",
		"feat",
	}

	testRepository := NewTestRepository(t, commits)

	th := NewTestHelper(t)
	err := th.SetFlags(map[string]string{
		BranchesConfiguration: `[{"name": "main"}]`,
		RulesConfiguration:    `{"minor": ["feat", "fix"]}`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	expectedVersion := "0.1.0"
	expectedTag := "v" + expectedVersion
	expectedOut := cmdOutput{
		Message:    MessageNewRelease,
		Version:    expectedVersion,
		NewRelease: true,
		Branch:     "main",
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
		BranchesConfiguration: `[{"name": "main"}]`,
		MonorepoConfiguration: `[{"name": "foo", "path": "foo"}, {"name": "bar", "path": "bar"}]`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    MessageNewRelease,
			Version:    "0.1.0",
			NewRelease: true,
			Branch:     "main",
			Project:    "foo",
		},
		{
			Message:    MessageNewRelease,
			Version:    "0.1.0",
			NewRelease: true,
			Branch:     "main",
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
		BranchesConfiguration: `[{"name": "main"}]`,
		MonorepoConfiguration: `[{"name": "foo", "path": "foo"}, {"name": "bar", "path": "bar"}]`,
	})
	checkErr(t, err, "setting flags")

	out, err := th.ExecuteCommand("release", testRepository.Path)
	checkErr(t, err, "executing command")

	i := 0
	expectedOutputs := []cmdOutput{
		{
			Message:    MessageNoNewRelease,
			Version:    "0.0.0",
			NewRelease: false,
			Branch:     "main",
			Project:    "foo",
		},
		{
			Message:    MessageNewRelease,
			Version:    "0.1.0",
			NewRelease: true,
			Branch:     "main",
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

func TestReleaseCmd_ConfigureRules_DefaultRules(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

	rules, err := configureRules(ctx)
	checkErr(t, err, "configuring rules")

	assert.Equal(rule.Default, rules)
}

func TestReleaseCmd_ConfigureBranches_NoBranches(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

	_, err := configureBranches(ctx)
	assert.ErrorIs(err, branch.ErrNoBranch)
}

func TestReleaseCmd_ConfigureProjects_NoProjects(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

	projects, err := configureProjects(ctx)
	checkErr(t, err, "configuring projects")

	assert.Nil(projects, "no monorepo configuration, should have gotten nil")
}

func TestReleaseCmd_InvalidCustomRules(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

	ctx.RulesFlag = map[string][]string{
		"minor": {"feat"},
		"patch": {"feat"},
	}

	_, err := configureRules(ctx)
	assert.ErrorIs(err, rule.ErrDuplicateReleaseRule, "should have failed parsing invalid custom rule")
}

func TestReleaseCmd_InvalidBranch(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

	ctx.BranchesFlag = []map[string]any{{"prerelease": true}}

	_, err := configureBranches(ctx)
	assert.ErrorIs(err, branch.ErrNoName, "should have failed parsing branch with no name")
}

func TestReleaseCmd_InvalidMonorepoProjects(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

	ctx.MonorepositoryFlag = []map[string]string{{"path": "foo"}}

	_, err := configureProjects(ctx)
	assert.ErrorIs(err, monorepo.ErrNoName, "should have failed parsing project with no name")
}

func TestReleaseCmd_InvalidArmoredKeyPath(t *testing.T) {
	assert := assertion.New(t)
	ctx := appcontext.New()

	ctx.GPGKeyPathFlag = "./does/not/exist"

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

	ctx.GPGKeyPathFlag = keyFilePath

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

func checkRelease(t *testing.T, r *gittest.TestRepository, i int, output []byte, expected []*cmdOutput) {
	assert := assertion.New(t)

	parts := make([]string, 0, len(expected))
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		parts = append(parts, scanner.Text())
	}

	for j, part := range parts {
		expectedOutput := *expected[j]
		actualOutput := cmdOutput{}
		err := json.Unmarshal([]byte(part), &actualOutput)
		checkErr(t, err, "unmarshalling main output")

		assert.Equal(expectedOutput, actualOutput, "releaseCmd output should be equal [%d:%d]", i, j)

		if actualOutput.NewRelease {
			expectedTag := "v" + actualOutput.Version

			exists, err := tag.Exists(r.Repository, expectedTag)
			checkErr(t, err, "checking if main tag exists")

			assert.Equal(true, exists, "main tag not found [%d:%d]", i, j)

			expectedTagRef, err := r.Tag(expectedTag)
			checkErr(t, err, "getting main tag ref")

			expectedTagObj, err := r.TagObject(expectedTagRef.Hash())
			checkErr(t, err, "getting main tag object")

			assert.Equal(taggerName, expectedTagObj.Tagger.Name)
			assert.Equal(taggerEmail, expectedTagObj.Tagger.Email)
		}
	}
}

func checkErr(t *testing.T, err error, message string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", message, err)
	}
}

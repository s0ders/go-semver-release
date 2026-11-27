package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/s0ders/go-semver-release/v8/internal/appcontext"
	"github.com/s0ders/go-semver-release/v8/internal/ci"
	"github.com/s0ders/go-semver-release/v8/internal/gittest"
)

func TestReleaseCmd_Release(t *testing.T) {
	allCommitTypes := []string{"fix", "feat!", "feat", "fix", "fix", "chores", "refactor", "test", "ci", "feat", "perf", "revert", "style"}

	tests := []struct {
		name        string
		commits     []string
		flags       map[string]string
		wantVersion string
		wantRelease bool
		wantMessage string
		wantTags    []string
	}{
		{
			name:        "default rules",
			commits:     allCommitTypes,
			wantVersion: "1.0.0", wantRelease: true, wantMessage: MessageNewRelease,
			wantTags: []string{"v1.0.0"},
		},
		{
			name:        "custom rules",
			commits:     []string{"fix", "feat"},
			flags:       map[string]string{RulesConfiguration: `{"minor": ["feat", "fix"]}`},
			wantVersion: "0.1.0", wantRelease: true, wantMessage: MessageNewRelease,
			wantTags: []string{"v0.1.0"},
		},
		{
			name:        "build metadata",
			commits:     []string{"fix", "feat!"},
			flags:       map[string]string{BuildMetadataConfiguration: "foobarbaz"},
			wantVersion: "1.0.0+foobarbaz", wantRelease: true, wantMessage: MessageNewRelease,
			wantTags: []string{"v1.0.0+foobarbaz"},
		},
		{
			name:        "custom tag prefix",
			commits:     []string{"feat"},
			flags:       map[string]string{TagPrefixConfiguration: "release-"},
			wantVersion: "0.1.0", wantRelease: true, wantMessage: MessageNewRelease,
			wantTags: []string{"release-0.1.0"},
		},
		{
			name:        "prerelease branch",
			commits:     []string{"fix", "feat!", "feat"},
			flags:       map[string]string{BranchesConfiguration: `[{"name": "master", "prerelease": true}]`},
			wantVersion: "1.0.0-master.1", wantRelease: true, wantMessage: MessageNewRelease,
			wantTags: []string{"v1.0.0-master.1"},
		},
		{
			name:        "dry-run does not tag",
			commits:     []string{"fix", "feat!"},
			flags:       map[string]string{DryRunConfiguration: "true"},
			wantVersion: "1.0.0", wantRelease: true, wantMessage: MessageDryRun,
		},
		{
			name:        "no new release",
			wantVersion: "0.0.0", wantMessage: MessageNoNewRelease,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.New(t)
			repo.MustCommits(t, tc.commits...)

			th := NewTestHelper(t)
			th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})
			th.MustSetFlags(t, tc.flags)

			output := th.MustRelease(t, repo.Path)

			require.Len(t, output.Releases, 1)
			release := output.Releases[0]
			assert.Equal(t, tc.wantVersion, release.Version)
			assert.Equal(t, "master", release.Branch)
			assert.Equal(t, tc.wantRelease, release.NewRelease)
			assert.Equal(t, tc.wantMessage, release.Message)

			assert.Equal(t, sorted(tc.wantTags), tagNames(t, repo), "tags pushed to the remote")
		})
	}
}

func TestReleaseCmd_LightweightTags(t *testing.T) {
	repo := gittest.New(t)
	repo.MustCommit(t, "feat")

	th := NewTestHelper(t)
	th.MustSetFlags(t, map[string]string{
		BranchesConfiguration:        `[{"name": "master"}]`,
		LightweightTagsConfiguration: "true",
	})
	th.MustRelease(t, repo.Path)

	ref, err := repo.Tag("v0.1.0")
	require.NoError(t, err)

	_, err = repo.TagObject(ref.Hash())
	assert.ErrorIs(t, err, plumbing.ErrObjectNotFound, "a lightweight tag has no tag object")
}

func TestReleaseCmd_GitHubOutput(t *testing.T) {
	repo := gittest.New(t)
	repo.MustCommit(t, "feat")

	th := NewTestHelper(t)
	th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})
	th.MustRelease(t, repo.Path)

	got, err := os.ReadFile(os.Getenv("GITHUB_OUTPUT"))
	require.NoError(t, err)
	assert.Equal(t, "\nMASTER_SEMVER=v0.1.0\nMASTER_NEW_RELEASE=true\n", string(got))
}

func TestReleaseCmd_MultiBranchRelease(t *testing.T) {
	repo := gittest.New(t)

	repo.MustTag(t, "1.0.0", repo.MustCommit(t, "feat!"))
	repo.MustCommits(t, "fix", "feat!", "feat", "fix", "chores", "perf")

	repo.MustCheckoutBranch(t, "rc")
	repo.MustCommits(t, "feat", "feat!", "perf")

	th := NewTestHelper(t)
	th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}, {"name": "rc", "prerelease": true}]`})

	output := th.MustRelease(t, repo.Path)

	assert.Equal(t, 2, output.Summary.ReleaseCount)
	assert.Equal(t, map[string]string{"master": "2.0.0", "rc": "2.0.0-rc.1"}, versionsBy(output, func(r ci.ReleaseOutput) string { return r.Branch }))
	assert.Equal(t, []string{"1.0.0", "v2.0.0", "v2.0.0-rc.1"}, tagNames(t, repo))
}

type fileCommit struct {
	path       string
	commitType string
}

func TestReleaseCmd_Monorepo(t *testing.T) {
	tests := []struct {
		name         string
		monorepo     string
		commits      []fileCommit
		wantVersions map[string]string
		wantTags     []string
	}{
		{
			name:         "every project released in a single push",
			monorepo:     `[{"name": "foo", "path": "foo"}, {"name": "bar", "path": "bar"}]`,
			commits:      []fileCommit{{"foo/a.txt", "feat"}, {"foo/b.txt", "fix"}, {"bar/a.txt", "feat!"}, {"bar/b.txt", "fix"}},
			wantVersions: map[string]string{"foo": "0.1.0", "bar": "1.0.0"},
			wantTags:     []string{"bar-v1.0.0", "foo-v0.1.0"},
		},
		{
			name:         "project without release",
			monorepo:     `[{"name": "foo", "path": "foo"}, {"name": "bar", "path": "bar"}]`,
			commits:      []fileCommit{{"bar/a.txt", "feat!"}, {"bar/b.txt", "fix"}},
			wantVersions: map[string]string{"foo": "0.0.0", "bar": "1.0.0"},
			wantTags:     []string{"bar-v1.0.0"},
		},
		{
			name:         "project with several paths",
			monorepo:     `[{"name": "bar", "paths": ["bar", "bar2"]}]`,
			commits:      []fileCommit{{"bar/a.txt", "fix"}, {"bar2/b.txt", "feat"}},
			wantVersions: map[string]string{"bar": "0.1.0"},
			wantTags:     []string{"bar-v0.1.0"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.New(t)
			for _, c := range tc.commits {
				repo.MustCommitFile(t, c.commitType, c.path)
			}

			th := NewTestHelper(t)
			th.MustSetFlags(t, map[string]string{
				BranchesConfiguration: `[{"name": "master"}]`,
				MonorepoConfiguration: tc.monorepo,
			})

			output := th.MustRelease(t, repo.Path)

			assert.Equal(t, tc.wantVersions, versionsBy(output, func(r ci.ReleaseOutput) string { return r.Project }))
			assert.Equal(t, sorted(tc.wantTags), tagNames(t, repo), "tags pushed to the remote")
		})
	}
}

// TestReleaseCmd_PrereleasePromotion checks that a prerelease merged into a stable branch, without any new release
// commit, is promoted by tagging the prerelease commit.
func TestReleaseCmd_PrereleasePromotion(t *testing.T) {
	repo := gittest.New(t)

	rcCommit := repo.MustCommit(t, "feat")
	repo.MustTag(t, "v1.0.0-rc.3", rcCommit)
	repo.MustCommit(t, "chore")

	th := NewTestHelper(t)
	th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})

	output := th.MustRelease(t, repo.Path)

	require.Len(t, output.Releases, 1)
	assert.Equal(t, "1.0.0", output.Releases[0].Version)
	assert.True(t, output.Releases[0].NewRelease)

	tagObject := annotatedTag(t, repo, "v1.0.0")
	assert.Equal(t, rcCommit, tagObject.Target, "promoted version should tag the prerelease commit")
}

func TestReleaseCmd_SecondRunIsNoop(t *testing.T) {
	repo := gittest.New(t)
	repo.MustCommit(t, "feat")

	for i, wantRelease := range []bool{true, false} {
		th := NewTestHelper(t)
		th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})

		output := th.MustRelease(t, repo.Path)

		require.Len(t, output.Releases, 1)
		assert.Equal(t, "0.1.0", output.Releases[0].Version, "run %d", i+1)
		assert.Equal(t, wantRelease, output.Releases[0].NewRelease, "run %d", i+1)
	}

	assert.Equal(t, []string{"v0.1.0"}, tagNames(t, repo))
}

// TestReleaseCmd_ExistingTagIsNoop checks that a computed tag already existing on the remote, but not reachable from
// the released branch, is left untouched instead of failing the release.
func TestReleaseCmd_ExistingTagIsNoop(t *testing.T) {
	repo := gittest.New(t)

	repo.MustCheckoutBranch(t, "other")
	otherCommit := repo.MustCommit(t, "feat")
	repo.MustTag(t, "v0.1.0", otherCommit)

	repo.MustCheckout(t, "master")
	repo.MustCommit(t, "feat")

	th := NewTestHelper(t)
	th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})

	output := th.MustRelease(t, repo.Path)

	require.Len(t, output.Releases, 1)
	assert.Equal(t, "0.1.0", output.Releases[0].Version)
	assert.Equal(t, otherCommit, annotatedTag(t, repo, "v0.1.0").Target, "existing tag should not be moved")
}

func TestReleaseCmd_ConfigurationFile(t *testing.T) {
	cfgFilePath := writeFile(t, "config.yml", `
git-name: My CI Robot
git-email: my-robot@release.ci
tag-prefix: v
branches:
  - name: master
  - name: alpha
    prerelease: true
monorepo:
  - name: foo
    path: ./foo/
rules:
  minor:
    - feat
  patch:
    - fix
`)

	repo := gittest.New(t)
	repo.MustCommitFile(t, "feat!", "foo/a.txt")
	repo.MustCommitFile(t, "feat", "bar/a.txt")
	repo.MustCheckoutBranch(t, "alpha")
	repo.MustCommitFile(t, "feat", "foo/b.txt")

	th := NewTestHelper(t)
	th.MustSetFlags(t, map[string]string{"config": cfgFilePath})

	output := th.MustRelease(t, repo.Path)

	assert.Equal(t, map[string]string{"master": "1.0.0", "alpha": "1.0.0-alpha.1"}, versionsBy(output, func(r ci.ReleaseOutput) string { return r.Branch }))
	for _, release := range output.Releases {
		assert.Equal(t, "foo", release.Project)
	}

	tagObject := annotatedTag(t, repo, "foo-v1.0.0")
	assert.Equal(t, "My CI Robot", tagObject.Tagger.Name)
	assert.Equal(t, "my-robot@release.ci", tagObject.Tagger.Email)
}

// TestReleaseCmd_ConfigurationPrecedence checks the documented precedence: flags, then environment variables, then
// the configuration file.
func TestReleaseCmd_ConfigurationPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		file    bool
		env     bool
		flag    bool
		wantTag string
	}{
		{name: "default", wantTag: "v0.1.0"},
		{name: "file", file: true, wantTag: "file-0.1.0"},
		{name: "environment over file", file: true, env: true, wantTag: "env-0.1.0"},
		{name: "flag over environment and file", file: true, env: true, flag: true, wantTag: "flag-0.1.0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.New(t)
			repo.MustCommit(t, "feat")

			th := NewTestHelper(t)
			th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})

			if tc.file {
				th.MustSetFlags(t, map[string]string{"config": writeFile(t, "config.yml", "tag-prefix: file-\n")})
			}
			if tc.env {
				t.Setenv("GO_SEMVER_RELEASE_TAG_PREFIX", "env-")
			}
			if tc.flag {
				th.MustSetFlags(t, map[string]string{TagPrefixConfiguration: "flag-"})
			}

			th.MustRelease(t, repo.Path)

			assert.Equal(t, []string{tc.wantTag}, tagNames(t, repo))
		})
	}
}

func TestReleaseCmd_ConfigurationAsEnvironmentVariable(t *testing.T) {
	t.Setenv("GO_SEMVER_RELEASE_ACCESS_TOKEN", "secret")

	repo := gittest.New(t)

	th := NewTestHelper(t)
	th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})
	th.MustRelease(t, repo.Path)

	assert.Equal(t, "secret", th.Ctx.AccessToken)
}

func TestReleaseCmd_Errors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, th *TestHelper) (repositoryPath string)
		wantErr string
	}{
		{
			name: "invalid repository path",
			setup: func(t *testing.T, th *TestHelper) string {
				return filepath.Join(t.TempDir(), "does-not-exist")
			},
			wantErr: "cloning Git repository",
		},
		{
			name: "empty repository",
			setup: func(t *testing.T, th *TestHelper) string {
				path := t.TempDir()
				_, err := git.PlainInit(path, false)
				require.NoError(t, err)
				return path
			},
			wantErr: "remote repository is empty",
		},
		{
			name: "read-only GitHub output",
			setup: func(t *testing.T, th *TestHelper) string {
				path := filepath.Join(t.TempDir(), "output")
				require.NoError(t, os.WriteFile(path, nil, 0o444))
				t.Setenv("GITHUB_OUTPUT", path)
				return gittest.New(t).Path
			},
			wantErr: "opening ci file",
		},
		{
			name: "missing GPG key",
			setup: func(t *testing.T, th *TestHelper) string {
				th.MustSetFlags(t, map[string]string{GPGPathConfiguration: filepath.Join(t.TempDir(), "missing.asc")})
				return gittest.New(t).Path
			},
			wantErr: "reading armored key",
		},
		{
			name: "invalid GPG key",
			setup: func(t *testing.T, th *TestHelper) string {
				th.MustSetFlags(t, map[string]string{GPGPathConfiguration: writeFile(t, "key.asc", "")})
				return gittest.New(t).Path
			},
			wantErr: "loading armored key",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			th := NewTestHelper(t)
			th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})

			_, err := th.ExecuteCommand("release", tc.setup(t, th))
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// Test utilities

type TestHelper struct {
	Ctx        *appcontext.AppContext
	Cmd        *cobra.Command
	ReleaseCmd *cobra.Command
}

// NewTestHelper creates a new TestHelper with a fresh AppContext and Command. GitHub output is redirected to a
// temporary file so that tests running in GitHub Actions never write to the workflow's own output.
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	t.Setenv("GITHUB_OUTPUT", filepath.Join(t.TempDir(), "github_output"))

	ctx := &appcontext.AppContext{
		Viper: viper.New(),
	}
	cmd := NewRootCommand(ctx)

	releaseCmd, _, err := cmd.Find([]string{"release"})
	require.NoError(t, err)

	return &TestHelper{
		Ctx:        ctx,
		Cmd:        cmd,
		ReleaseCmd: releaseCmd,
	}
}

// MustSetFlags sets the given flags on the release subcommand.
func (th *TestHelper) MustSetFlags(t *testing.T, flags map[string]string) {
	t.Helper()

	for name, value := range flags {
		require.NoError(t, th.ReleaseCmd.Flags().Set(name, value), "setting flag %q", name)
	}
}

// ExecuteCommand executes the root command with the given arguments and returns its output.
func (th *TestHelper) ExecuteCommand(args ...string) ([]byte, error) {
	output := new(bytes.Buffer)
	th.Cmd.SetOut(output)
	th.Cmd.SetErr(output)
	th.Cmd.SetArgs(args)

	err := th.Cmd.Execute()

	return output.Bytes(), err
}

// MustRelease runs the release command on the given repository and returns its parsed JSON output.
func (th *TestHelper) MustRelease(t *testing.T, repositoryPath string) *ci.JSONOutput {
	t.Helper()

	out, err := th.ExecuteCommand("release", repositoryPath)
	require.NoError(t, err, "running release command, output: %s", out)

	var result ci.JSONOutput
	require.NoError(t, json.Unmarshal(out, &result), "parsing JSON output: %s", out)

	return &result
}

// versionsBy indexes released versions by the given key (e.g., branch or project).
func versionsBy(output *ci.JSONOutput, key func(ci.ReleaseOutput) string) map[string]string {
	versions := make(map[string]string, len(output.Releases))
	for _, r := range output.Releases {
		versions[key(r)] = r.Version
	}

	return versions
}

// tagNames returns the sorted names of every tag of the repository.
func tagNames(t *testing.T, repo *gittest.TestRepository) []string {
	t.Helper()

	iter, err := repo.Tags()
	require.NoError(t, err)

	names := []string{}
	require.NoError(t, iter.ForEach(func(ref *plumbing.Reference) error {
		names = append(names, ref.Name().Short())
		return nil
	}))

	return sorted(names)
}

func annotatedTag(t *testing.T, repo *gittest.TestRepository, name string) *object.Tag {
	t.Helper()

	ref, err := repo.Tag(name)
	require.NoError(t, err)

	tagObject, err := repo.TagObject(ref.Hash())
	require.NoError(t, err)

	return tagObject
}

func sorted(s []string) []string {
	if s == nil {
		return []string{}
	}

	s = slices.Clone(s)
	slices.Sort(s)

	return s
}

// writeFile writes the given content to a new file of a temporary directory and returns its path.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

func TestReleaseCmd_RemovesLocalClone(t *testing.T) {
	tempDir := t.TempDir()
	for _, env := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(env, tempDir)
	}

	repo := gittest.New(t)
	repo.MustCommit(t, "feat")

	th := NewTestHelper(t)
	th.MustSetFlags(t, map[string]string{BranchesConfiguration: `[{"name": "master"}]`})
	th.MustRelease(t, repo.Path)

	clones, err := filepath.Glob(filepath.Join(tempDir, "go-semver-release-*"))
	require.NoError(t, err)
	assert.Empty(t, clones, "the local clone should be removed once the release is done")
}

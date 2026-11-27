package parser

import (
	"encoding/json"
	"io"
	"slices"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/s0ders/go-semver-release/v8/internal/appcontext"
	"github.com/s0ders/go-semver-release/v8/internal/branch"
	"github.com/s0ders/go-semver-release/v8/internal/gittest"
	"github.com/s0ders/go-semver-release/v8/internal/monorepo"
	"github.com/s0ders/go-semver-release/v8/internal/rule"
	"github.com/s0ders/go-semver-release/v8/internal/semver"
)

// newTestContext returns an application context with the default release rules and a single stable "master" branch.
func newTestContext(t *testing.T) *appcontext.AppContext {
	t.Helper()

	ctx := &appcontext.AppContext{
		RemoteName:  "origin",
		BranchesCfg: []branch.Item{{Name: "master"}},
		Logger:      zerolog.New(io.Discard),
	}

	defaultRules, err := json.Marshal(rule.Default)
	require.NoError(t, err)
	require.NoError(t, ctx.RulesCfg.Set(string(defaultRules)))

	return ctx
}

// reachableFrom returns the hash set of commits reachable from the given local branch.
func reachableFrom(t *testing.T, repo *gittest.TestRepository, branchName string) map[plumbing.Hash]struct{} {
	t.Helper()

	reachable, err := BuildReachableCommits(repo.Repository, repo.MustBranchRef(t, branchName))
	require.NoError(t, err)

	return reachable.HashSet
}

// historyStep is one step of a scripted commit history: an optional commit followed by an optional tag on the
// latest commit (HEAD of the repository when no commit was made yet).
type historyStep struct {
	commit string
	tag    string
}

// buildHistory replays the given steps on the repository and returns the hash of each step's commit, or of HEAD at
// that point for tag-only steps.
func buildHistory(t *testing.T, repo *gittest.TestRepository, steps []historyStep) []plumbing.Hash {
	t.Helper()

	head, err := repo.Head()
	require.NoError(t, err)

	last := head.Hash()
	hashes := make([]plumbing.Hash, len(steps))

	for i, step := range steps {
		if step.commit != "" {
			last = repo.MustCommit(t, step.commit)
		}
		if step.tag != "" {
			repo.MustTag(t, step.tag, last)
		}
		hashes[i] = last
	}

	return hashes
}

func TestParser_ParseCommitMessage(t *testing.T) {
	tests := []struct {
		message      string
		conventional bool
		commitType   string
		breaking     bool
	}{
		{message: "feat: implemented foo", conventional: true, commitType: "feat"},
		{message: "fix(foo.js): fixed foo", conventional: true, commitType: "fix"},
		{message: "chore(api): fixed doc typos", conventional: true, commitType: "chore"},
		{message: "test(../tests/): implemented unit tests", conventional: true, commitType: "test"},
		{message: "ci(ci.yaml): added stages to pipeline", conventional: true, commitType: "ci"},
		{message: "feat!: dropped the v1 API", conventional: true, commitType: "feat", breaking: true},
		{message: "fix(foo.js)!: fixed foo", conventional: true, commitType: "fix", breaking: true},
		{message: "chore(docs): fixed typos BREAKING CHANGE: deleted some APIs", conventional: true, commitType: "chore", breaking: true},
		{message: "fix: renamed flag\n\nBREAKING CHANGE: --foo is now --bar", conventional: true, commitType: "fix", breaking: true},
		{message: "Merge branch 'main' into feature"},
		{message: "feature: not a conventional type"},
		{message: "Feat: types are case sensitive"},
		{message: "feat:missing space"},
		{message: "feat: "},
	}

	p := New(newTestContext(t))

	for _, tc := range tests {
		t.Run(tc.message, func(t *testing.T) {
			commit := &object.Commit{
				Hash:    plumbing.ComputeHash(plumbing.CommitObject, []byte(tc.message)),
				Message: tc.message,
			}

			got := p.parseCommitMessage(commit)

			assert.Equal(t, commitKind{conventional: tc.conventional, commitType: tc.commitType, breaking: tc.breaking}, got)
		})
	}
}

func TestParser_FetchLatestSemverTag(t *testing.T) {
	tests := []struct {
		name    string
		project monorepo.Item
		setup   func(t *testing.T, repo *gittest.TestRepository)
		want    string
	}{
		{
			name:  "no tag",
			setup: func(t *testing.T, repo *gittest.TestRepository) {},
		},
		{
			name: "highest version wins regardless of creation order",
			setup: func(t *testing.T, repo *gittest.TestRepository) {
				head := repo.MustCommit(t, "fix")
				for _, v := range []string{"2.0.0", "2.0.1", "3.0.0", "2.5.0", "0.0.2", "0.1.0", "1.0.0"} {
					repo.MustTag(t, v, head)
				}
			},
			want: "3.0.0",
		},
		{
			name: "lightweight and annotated tags are both considered",
			setup: func(t *testing.T, repo *gittest.TestRepository) {
				repo.MustLightweightTag(t, "v1.0.0", repo.MustCommit(t, "fix"))
				repo.MustTag(t, "v2.0.0", repo.MustCommit(t, "feat"))
				repo.MustLightweightTag(t, "v3.0.0", repo.MustCommit(t, "feat"))
			},
			want: "v3.0.0",
		},
		{
			name: "non-semver tags are ignored",
			setup: func(t *testing.T, repo *gittest.TestRepository) {
				head := repo.MustCommit(t, "fix")
				repo.MustTag(t, "1.0.0", head)
				repo.MustTag(t, "latest", head)
				repo.MustTag(t, "v2", head)
			},
			want: "1.0.0",
		},
		{
			name: "tags not pointing to a commit are ignored",
			setup: func(t *testing.T, repo *gittest.TestRepository) {
				head := repo.MustCommit(t, "fix")
				repo.MustTag(t, "1.0.0", head)

				commit, err := repo.CommitObject(head)
				require.NoError(t, err)
				require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference("refs/tags/9.0.0", commit.TreeHash)))
			},
			want: "1.0.0",
		},
		{
			name: "tags unreachable from the branch are ignored",
			setup: func(t *testing.T, repo *gittest.TestRepository) {
				repo.MustTag(t, "1.0.0", repo.MustCommit(t, "feat"))
				repo.MustCheckoutBranch(t, "other")
				repo.MustTag(t, "9.9.9", repo.MustCommit(t, "feat"))
				repo.MustCheckout(t, "master")
			},
			want: "1.0.0",
		},
		{
			name:    "project only sees its own tags",
			project: monorepo.Item{Name: "foo", Path: "foo"},
			setup: func(t *testing.T, repo *gittest.TestRepository) {
				head := repo.MustCommit(t, "fix")
				repo.MustTag(t, "foo-1.0.0", head)
				repo.MustTag(t, "bar-2.0.0", head)
				repo.MustTag(t, "3.0.0", head)
			},
			want: "foo-1.0.0",
		},
		{
			name:    "project tags with a tag prefix and a prerelease",
			project: monorepo.Item{Name: "foo", Path: "foo"},
			setup: func(t *testing.T, repo *gittest.TestRepository) {
				head := repo.MustCommit(t, "fix")
				repo.MustTag(t, "foo-v1.0.0", head)
				repo.MustTag(t, "foo-v1.1.0-rc.1", head)
			},
			want: "foo-v1.1.0-rc.1",
		},
		{
			name:    "project does not claim tags of a project sharing its name as prefix",
			project: monorepo.Item{Name: "api", Path: "api"},
			setup: func(t *testing.T, repo *gittest.TestRepository) {
				head := repo.MustCommit(t, "fix")
				repo.MustTag(t, "api-1.0.0", head)
				repo.MustTag(t, "api-gateway-2.0.0", head)
				repo.MustTag(t, "api-gateway-v3.0.0", head)
			},
			want: "api-1.0.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.New(t)
			tc.setup(t, repo)

			latest, err := New(newTestContext(t)).FetchLatestSemverTag(repo.Repository, tc.project, reachableFrom(t, repo, "master"))
			require.NoError(t, err)

			if tc.want == "" {
				assert.Nil(t, latest)
				return
			}

			require.NotNil(t, latest)
			assert.Equal(t, tc.want, latest.Name)
		})
	}
}

func TestParser_FetchLatestSemverTag_DivergentBranches(t *testing.T) {
	tests := []struct {
		name    string
		project monorepo.Item
		prefix  string
	}{
		{name: "repository"},
		{name: "monorepo project", project: monorepo.Item{Name: "foo", Path: "foo"}, prefix: "foo-"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.New(t)

			repo.MustTag(t, tc.prefix+"1.0.0", repo.MustCommitFile(t, "feat", "foo/a.txt"))

			repo.MustCheckoutBranch(t, "release-1.x")
			repo.MustTag(t, tc.prefix+"1.0.1", repo.MustCommitFile(t, "fix", "foo/b.txt"))

			repo.MustCheckout(t, "master")
			repo.MustTag(t, tc.prefix+"2.0.0", repo.MustCommitFile(t, "feat!", "foo/c.txt"))

			p := New(newTestContext(t))

			for branchName, want := range map[string]string{"master": "2.0.0", "release-1.x": "1.0.1"} {
				latest, err := p.FetchLatestSemverTag(repo.Repository, tc.project, reachableFrom(t, repo, branchName))
				require.NoError(t, err)
				require.NotNil(t, latest)
				assert.Equal(t, tc.prefix+want, latest.Name, "branch %q", branchName)
			}
		})
	}
}

func TestParser_FetchLatestStableTag(t *testing.T) {
	repo := gittest.New(t)
	p := New(newTestContext(t))

	latest, err := p.FetchLatestStableTag(repo.Repository, monorepo.Item{}, reachableFrom(t, repo, "master"))
	require.NoError(t, err)
	assert.Nil(t, latest, "no stable tag should be found in an untagged repository")

	repo.MustTag(t, "1.0.0", repo.MustCommit(t, "feat"))
	repo.MustTag(t, "1.0.5", repo.MustCommit(t, "fix"))
	repo.MustTag(t, "1.1.0-rc.1", repo.MustCommit(t, "feat"))

	latest, err = p.FetchLatestStableTag(repo.Repository, monorepo.Item{}, reachableFrom(t, repo, "master"))
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, "1.0.5", latest.String(), "prereleases should be skipped")
}

func TestParser_FetchLatestPrereleaseTag(t *testing.T) {
	repo := gittest.New(t)
	for _, v := range []string{"1.0.0-rc.1", "1.0.0-rc.3", "1.0.0-rc.2", "1.0.0-beta.1", "1.0.0-beta.2", "1.1.0-rc.5", "1.0.0"} {
		repo.MustTag(t, v, repo.MustCommit(t, "fix"))
	}

	tests := []struct {
		core  semver.Version
		label string
		want  string
	}{
		{core: semver.Version{Major: 1}, label: "rc", want: "1.0.0-rc.3"},
		{core: semver.Version{Major: 1}, label: "beta", want: "1.0.0-beta.2"},
		{core: semver.Version{Major: 1, Minor: 1}, label: "rc", want: "1.1.0-rc.5"},
		{core: semver.Version{Major: 1, Minor: 1}, label: "beta"},
		{core: semver.Version{Major: 2}, label: "rc"},
	}

	p := New(newTestContext(t))
	reachable := reachableFrom(t, repo, "master")

	for _, tc := range tests {
		t.Run(tc.core.String()+"-"+tc.label, func(t *testing.T) {
			latest, err := p.FetchLatestPrereleaseTag(repo.Repository, monorepo.Item{}, reachable, &tc.core, tc.label)
			require.NoError(t, err)

			if tc.want == "" {
				assert.Nil(t, latest)
				return
			}

			require.NotNil(t, latest)
			assert.Equal(t, tc.want, latest.String())
		})
	}
}

func TestParser_ComputeNewSemver(t *testing.T) {
	const noReleaseCommit = -1

	tests := []struct {
		name       string
		prerelease bool
		metadata   string
		history    []historyStep
		want       string
		newRelease bool
		// releaseCommit is the index of the history step whose commit should be released, or noReleaseCommit.
		releaseCommit int
	}{
		// Stable branch, untagged repository
		{name: "no commit", want: "0.0.0", releaseCommit: noReleaseCommit},
		{name: "patch", history: []historyStep{{commit: "fix"}}, want: "0.0.1", newRelease: true},
		{name: "minor", history: []historyStep{{commit: "feat"}}, want: "0.1.0", newRelease: true},
		{name: "major", history: []historyStep{{commit: "feat!"}}, want: "1.0.0", newRelease: true},
		{name: "major from footer", history: []historyStep{{commit: "fix: renamed flag\n\nBREAKING CHANGE: --foo is now --bar"}}, want: "1.0.0", newRelease: true},
		{name: "build metadata", metadata: "build.42", history: []historyStep{{commit: "feat"}}, want: "0.1.0+build.42", newRelease: true},
		{
			name:          "commit types without release rule",
			history:       []historyStep{{commit: "chore"}, {commit: "docs"}, {commit: "ci"}},
			want:          "0.0.0",
			releaseCommit: noReleaseCommit,
		},
		{
			name:          "non-conventional commits",
			history:       []historyStep{{commit: "Update README"}},
			want:          "0.0.0",
			releaseCommit: noReleaseCommit,
		},

		// Stable branch, tagged repository: a single bump per release, the highest one wins
		{
			name:    "multiple patches bump once",
			history: []historyStep{{commit: "feat!", tag: "1.0.0"}, {commit: "fix"}, {commit: "fix"}, {commit: "fix"}},
			want:    "1.0.1", newRelease: true, releaseCommit: 3,
		},
		{
			name:    "minor wins over patch",
			history: []historyStep{{commit: "feat!", tag: "1.0.0"}, {commit: "fix"}, {commit: "feat"}, {commit: "fix"}},
			want:    "1.1.0", newRelease: true, releaseCommit: 3,
		},
		{
			name:    "major wins over minor and patch",
			history: []historyStep{{commit: "feat!", tag: "1.0.0"}, {commit: "fix"}, {commit: "feat!"}, {commit: "feat"}},
			want:    "2.0.0", newRelease: true, releaseCommit: 3,
		},
		{
			name:    "released commit is the latest release commit",
			history: []historyStep{{commit: "feat!", tag: "1.0.0"}, {commit: "feat"}, {commit: "chore"}},
			want:    "1.1.0", newRelease: true, releaseCommit: 1,
		},
		{
			name:          "no release commit since latest tag",
			history:       []historyStep{{commit: "feat!", tag: "1.0.0"}, {commit: "chore"}},
			want:          "1.0.0",
			releaseCommit: noReleaseCommit,
		},

		// Stable branch, promotion of a prerelease (e.g., after merging a prerelease branch)
		{
			name:    "prerelease is promoted",
			history: []historyStep{{commit: "feat", tag: "1.0.0-rc.3"}},
			want:    "1.0.0", newRelease: true, releaseCommit: 0,
		},
		{
			name:    "prerelease is promoted on the prerelease commit despite non-release commits",
			history: []historyStep{{commit: "feat", tag: "1.0.0-rc.3"}, {commit: "chore"}},
			want:    "1.0.0", newRelease: true, releaseCommit: 0,
		},
		{
			name:    "prerelease is bumped and promoted",
			history: []historyStep{{commit: "feat", tag: "1.0.0-rc.3"}, {commit: "fix"}},
			want:    "1.0.1", newRelease: true, releaseCommit: 1,
		},

		// Prerelease branch
		{
			name:       "first prerelease",
			prerelease: true,
			history:    []historyStep{{commit: "feat"}},
			want:       "0.1.0-master.1", newRelease: true,
		},
		{
			name:       "prerelease number is bumped on patch",
			prerelease: true,
			history:    []historyStep{{commit: "feat", tag: "0.1.0-master.1"}, {commit: "fix"}},
			want:       "0.1.0-master.2", newRelease: true, releaseCommit: 1,
		},
		{
			name:       "new prerelease series on minor",
			prerelease: true,
			history:    []historyStep{{commit: "feat", tag: "0.1.0-master.1"}, {commit: "feat"}},
			want:       "0.2.0-master.1", newRelease: true, releaseCommit: 1,
		},
		{
			name:          "prerelease without new commit",
			prerelease:    true,
			history:       []historyStep{{commit: "feat", tag: "0.1.0-master.1"}},
			want:          "0.1.0-master.1",
			releaseCommit: noReleaseCommit,
		},
		{
			name:       "prerelease after stable minor",
			prerelease: true,
			history:    []historyStep{{commit: "feat!", tag: "1.0.0"}, {commit: "feat"}},
			want:       "1.1.0-master.1", newRelease: true, releaseCommit: 1,
		},
		{
			name:       "prerelease after stable major",
			prerelease: true,
			history:    []historyStep{{commit: "feat", tag: "1.0.0"}, {commit: "feat!"}},
			want:       "2.0.0-master.1", newRelease: true, releaseCommit: 1,
		},
		{
			name:       "existing prerelease of the next version is bumped when the latest tag has another label",
			prerelease: true,
			history: []historyStep{
				{commit: "feat!", tag: "1.0.0"},
				{commit: "fix", tag: "1.0.1-master.1"},
				{commit: "fix", tag: "1.0.1-rc.1"},
				{commit: "fix"},
			},
			want: "1.0.1-master.2", newRelease: true, releaseCommit: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.New(t)
			hashes := buildHistory(t, repo, tc.history)

			ctx := newTestContext(t)
			ctx.BuildMetadata = tc.metadata
			ctx.BranchesCfg[0].Prerelease = tc.prerelease

			output, err := New(ctx).ComputeNewSemver(repo.Repository, monorepo.Item{}, ctx.BranchesCfg[0])
			require.NoError(t, err)

			assert.Equal(t, tc.want, output.Semver.String(), "version")
			assert.Equal(t, tc.newRelease, output.NewRelease, "new release")
			assert.Equal(t, "master", output.Branch, "branch")

			wantCommit := plumbing.ZeroHash
			if tc.releaseCommit != noReleaseCommit && len(hashes) > 0 {
				wantCommit = hashes[tc.releaseCommit]
			}
			assert.Equal(t, wantCommit, output.CommitHash, "released commit")
		})
	}
}

func TestParser_ComputeNewSemver_UnknownReleaseType(t *testing.T) {
	repo := gittest.New(t)
	repo.MustCommit(t, "feat")

	ctx := newTestContext(t)
	ctx.RulesCfg = map[string]string{"feat": "unknown", "fix": "patch"}

	_, err := New(ctx).ComputeNewSemver(repo.Repository, monorepo.Item{}, ctx.BranchesCfg[0])
	assert.ErrorContains(t, err, "unknown release type")
}

func TestParser_ComputeNewSemver_UninitializedRepository(t *testing.T) {
	repository, err := git.PlainInit(t.TempDir(), false)
	require.NoError(t, err)

	ctx := newTestContext(t)

	_, err = New(ctx).ComputeNewSemver(repository, monorepo.Item{}, ctx.BranchesCfg[0])
	assert.ErrorIs(t, err, plumbing.ErrReferenceNotFound)
}

func TestParser_Run(t *testing.T) {
	repo := gittest.New(t)
	repo.MustCommit(t, "feat")

	output, err := New(newTestContext(t)).Run(repo.MustClone(t).Repository)
	require.NoError(t, err)

	require.Len(t, output, 1)
	assert.Equal(t, "0.1.0", output[0].Semver.String())
	assert.Equal(t, "master", output[0].Branch)
	assert.Empty(t, output[0].Project.Name)
	assert.True(t, output[0].NewRelease)
}

func TestParser_Run_Monorepo(t *testing.T) {
	tests := []struct {
		name string
		tags bool
		want map[string]string
	}{
		{name: "untagged projects", want: map[string]string{"foo": "1.0.0", "bar": "0.1.0", "baz": "0.0.0"}},
		{name: "tagged projects", tags: true, want: map[string]string{"foo": "2.0.0", "bar": "1.1.0", "baz": "0.0.0"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.New(t)

			if tc.tags {
				repo.MustTag(t, "foo-1.0.0", repo.MustCommitFile(t, "feat!", "foo/init.txt"))
				repo.MustTag(t, "bar-1.0.0", repo.MustCommitFile(t, "chore!", "bar/init.txt"))
			}

			repo.MustCommitFile(t, "feat!", "foo/foo.txt")
			repo.MustCommitFile(t, "fix", "foo/xyz/foo.txt")
			repo.MustCommitFile(t, "feat", "bar/foo.txt")
			repo.MustCommitFile(t, "fix", "bar/baz/xyz/foo.txt")
			// Commits outside of any project, or under a directory merely prefixed by a project name
			repo.MustCommitFile(t, "feat!", "unknown/a.txt")
			repo.MustCommitFile(t, "feat!", "bazooka/a.txt")

			ctx := newTestContext(t)
			ctx.MonorepositoryCfg = []monorepo.Item{
				{Name: "foo", Path: "foo"},
				{Name: "bar", Path: "bar"},
				{Name: "baz", Path: "baz"},
			}

			output, err := New(ctx).Run(repo.MustClone(t).Repository)
			require.NoError(t, err)

			require.Len(t, output, len(ctx.MonorepositoryCfg))
			for i, project := range ctx.MonorepositoryCfg {
				assert.Equal(t, project.Name, output[i].Project.Name, "outputs should follow the projects order")
				assert.Equal(t, tc.want[project.Name], output[i].Semver.String(), "project %q", project.Name)
				assert.Equal(t, tc.want[project.Name] != "0.0.0", output[i].NewRelease, "project %q", project.Name)
			}
		})
	}
}

// TestParser_Run_StableAndPrereleaseBranches checks that stable branches are processed first and that branches
// sharing tags do not influence each other's computed versions.
func TestParser_Run_StableAndPrereleaseBranches(t *testing.T) {
	repo := gittest.New(t)

	repo.MustTag(t, "1.0.0", repo.MustCommit(t, "feat!"))
	repo.MustCommit(t, "feat")

	repo.MustCheckoutBranch(t, "rc")
	repo.MustCommit(t, "fix")
	repo.MustCheckout(t, "master")

	ctx := newTestContext(t)
	ctx.BranchesCfg = []branch.Item{{Name: "rc", Prerelease: true}, {Name: "master"}}

	output, err := New(ctx).Run(repo.MustClone(t).Repository)
	require.NoError(t, err)

	require.Len(t, output, 2)
	assert.Equal(t, "master", output[0].Branch, "stable branches should be processed first")
	assert.Equal(t, "1.1.0", output[0].Semver.String())
	assert.Equal(t, "rc", output[1].Branch)
	assert.Equal(t, "1.1.0-rc.1", output[1].Semver.String(), "prerelease should be computed from the shared 1.0.0 tag")
}

func TestParser_Run_InvalidBranch(t *testing.T) {
	repo := gittest.New(t)

	ctx := newTestContext(t)
	ctx.BranchesCfg = []branch.Item{{Name: "does_not_exist"}}

	_, err := New(ctx).Run(repo.Repository)
	assert.ErrorIs(t, err, plumbing.ErrReferenceNotFound)
}

func TestParser_CommitContainsProjectFiles(t *testing.T) {
	repo := gittest.New(t)
	p := New(nil)

	tests := []struct {
		file  string
		paths []string
		want  bool
	}{
		{file: "api/handler.go", paths: []string{"api"}, want: true},
		{file: "api/v2/handler.go", paths: []string{"api"}, want: true},
		// Directories merely sharing a prefix with the project path
		{file: "api-v2/handler.go", paths: []string{"api"}, want: false},
		{file: "api2/handler.go", paths: []string{"api"}, want: false},
		{file: "apiserver/handler.go", paths: []string{"api"}, want: false},
		// Exact match on a file path
		{file: "README.md", paths: []string{"README.md"}, want: true},
		// Nested project paths
		{file: "src/components/Button.tsx", paths: []string{"src/components"}, want: true},
		{file: "src/components/forms/Input.tsx", paths: []string{"src/components"}, want: true},
		{file: "src/componentsv2/Button.tsx", paths: []string{"src/components"}, want: false},
		// A path traversing a file cannot match
		{file: "lib/other.go", paths: []string{"README.md/lib"}, want: false},
		// Any of several paths matching is enough
		{file: "shared/util.go", paths: []string{"api", "shared"}, want: true},
		{file: "docs/index.md", paths: []string{"api", "shared"}, want: false},
		// Neither the repository root nor an empty configuration match everything
		{file: "root.go", paths: []string{"."}, want: false},
		{file: "any/file.go", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			commit, err := repo.CommitObject(repo.MustCommitFile(t, "fix", tc.file))
			require.NoError(t, err)

			got, err := p.commitContainsProjectFiles(repo.Storer, commit, monorepo.Item{Name: "p", Paths: tc.paths})
			require.NoError(t, err)

			assert.Equal(t, tc.want, got, "project paths %v", tc.paths)
		})
	}
}

func TestParser_CommitContainsProjectFiles_Deletion(t *testing.T) {
	repo := gittest.New(t)
	repo.MustCommitFile(t, "fix", "foo/foo.txt")

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	_, err = worktree.Remove("foo/foo.txt")
	require.NoError(t, err)

	hash, err := worktree.Commit("fix: remove foo", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.test", When: repo.When()},
	})
	require.NoError(t, err)

	commit, err := repo.CommitObject(hash)
	require.NoError(t, err)

	p := New(nil)

	got, err := p.commitContainsProjectFiles(repo.Storer, commit, monorepo.Item{Name: "foo", Path: "foo"})
	require.NoError(t, err)
	assert.True(t, got, "deleting a project file is a project change")

	got, err = p.commitContainsProjectFiles(repo.Storer, commit, monorepo.Item{Name: "bar", Path: "bar"})
	require.NoError(t, err)
	assert.False(t, got, "deleting another project's file is not a project change")
}

func TestParser_CommitContainsProjectFiles_RootCommit(t *testing.T) {
	repo := gittest.New(t)

	head, err := repo.Head()
	require.NoError(t, err)

	root, err := repo.CommitObject(head.Hash())
	require.NoError(t, err)
	require.Empty(t, root.ParentHashes, "the repository's first commit should have no parent")

	p := New(nil)

	got, err := p.commitContainsProjectFiles(repo.Storer, root, monorepo.Item{Name: "sample", Path: "sample.txt"})
	require.NoError(t, err)
	assert.True(t, got, "files added by the root commit are project changes")

	got, err = p.commitContainsProjectFiles(repo.Storer, root, monorepo.Item{Name: "foo", Path: "foo"})
	require.NoError(t, err)
	assert.False(t, got)
}

func TestParser_SortBranches(t *testing.T) {
	tests := []struct {
		name     string
		branches []branch.Item
		want     []string
	}{
		{name: "empty", branches: []branch.Item{}, want: []string{}},
		{
			name:     "stable first, order preserved within groups",
			branches: []branch.Item{{Name: "rc", Prerelease: true}, {Name: "main"}, {Name: "beta", Prerelease: true}, {Name: "develop"}},
			want:     []string{"main", "develop", "rc", "beta"},
		},
		{name: "all stable", branches: []branch.Item{{Name: "main"}, {Name: "develop"}}, want: []string{"main", "develop"}},
		{
			name:     "all prerelease",
			branches: []branch.Item{{Name: "rc", Prerelease: true}, {Name: "beta", Prerelease: true}},
			want:     []string{"rc", "beta"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original := slices.Clone(tc.branches)

			got := make([]string, 0, len(tc.branches))
			for _, b := range sortBranches(tc.branches) {
				got = append(got, b.Name)
			}

			assert.Equal(t, tc.want, got)
			assert.Equal(t, original, tc.branches, "input should not be modified")
		})
	}
}

func TestParser_GetBranchReference(t *testing.T) {
	repo := gittest.New(t)
	local := repo.MustCommit(t, "feat")
	remote := repo.MustCommit(t, "fix")

	// The local "master" and its remote-tracking branch point to different commits, and "rc" is only known as a
	// remote-tracking branch.
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("master"), local)))
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewRemoteReferenceName("origin", "master"), remote)))
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewRemoteReferenceName("origin", "rc"), remote)))

	p := New(newTestContext(t))

	ref, err := p.getBranchReference(repo.Repository, "master")
	require.NoError(t, err)
	assert.Equal(t, local, ref.Hash(), "the local branch should be preferred")

	ref, err = p.getBranchReference(repo.Repository, "rc")
	require.NoError(t, err)
	assert.Equal(t, remote, ref.Hash(), "a branch only known as a remote-tracking branch should be found")

	_, err = p.getBranchReference(repo.Repository, "missing")
	assert.ErrorIs(t, err, plumbing.ErrReferenceNotFound)
}

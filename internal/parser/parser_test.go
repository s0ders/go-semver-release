package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rs/zerolog"
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v7/internal/appcontext"
	"github.com/s0ders/go-semver-release/v7/internal/branch"
	"github.com/s0ders/go-semver-release/v7/internal/gittest"
	"github.com/s0ders/go-semver-release/v7/internal/monorepo"
	"github.com/s0ders/go-semver-release/v7/internal/rule"
	"github.com/s0ders/go-semver-release/v7/internal/semver"
)

func TestParser_CommitTypeRegex(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		commit     string
		commitType string
	}

	matrix := []test{
		{"feat: implemented foo", "feat"},
		{"fix(foo.js): fixed foo", "fix"},
		{"chore(api): fixed doc typos", "chore"},
		{"test(../tests/): implemented unit tests", "test"},
		{"ci(ci.yaml): added stages to pipeline", "ci"},
	}

	for _, item := range matrix {
		got := conventionalCommitRegex.FindStringSubmatch(item.commit)[1]

		assert.Equal(item.commitType, got, "commit type should be equal")
	}
}

func TestParser_BreakingChangeRegex(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		commit     string
		isBreaking bool
	}

	matrix := []test{
		{"feat: implemented foo", false},
		{"fix(foo.js)!: fixed foo", true},
		{"chore(docs): fixed doc typos BREAKING CHANGE: delete some APIs", true},
	}

	for _, item := range matrix {
		submatch := conventionalCommitRegex.FindStringSubmatch(item.commit)
		got := strings.Contains(submatch[3], "!") || strings.Contains(submatch[0], "BREAKING CHANGE")

		assert.Equal(item.isBreaking, got, "breaking change should be equal")
	}
}

func TestParser_FetchLatestSemverTag_NoTag(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building reachable commits", err)

	th := NewTestHelper(t)

	parser := New(th.Ctx)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet)
	checkErr(t, "fetching latest semver tag", err)

	assert.Nil(latest, "latest semver tag should be nil")
}

func TestParser_FetchLatestSemverTag_OneTag(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	head, err := testRepository.Head()
	checkErr(t, "fetching head", err)

	tagName := "1.0.0"

	err = testRepository.AddTag(tagName, head.Hash())
	checkErr(t, "creating tag", err)

	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building reachable commits", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet)
	checkErr(t, "fetching latest semver tag", err)

	assert.Equal(tagName, latest.Name, "latest semver tagName should be equal")
}

func TestParser_FetchLatestSemverTag_MultipleTags(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	head, err := testRepository.Head()
	if err != nil {
		t.Fatalf("fetching head: %s", err)
	}

	tags := []string{"2.0.0", "2.0.1", "3.0.0", "2.5.0", "0.0.2", "0.0.1", "0.1.0", "1.0.0"}

	for _, v := range tags {
		err = testRepository.AddTag(v, head.Hash())
		checkErr(t, "creating tag", err)
	}

	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building reachable commits", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet)
	checkErr(t, "fetching latest semver tag", err)

	want := "3.0.0"
	assert.Equal(want, latest.Name, "latest semver tag should be equal")
}

func TestParser_FetchLatestSemverTag_LightweightTags(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Add commits
	_, err = testRepository.AddCommit("fix")
	checkErr(t, "adding commit", err)

	hash1, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	_, err = testRepository.AddCommit("fix")
	checkErr(t, "adding commit", err)

	hash2, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	// Add lightweight tags (not annotated)
	err = testRepository.AddLightweightTag("v1.0.0", hash1)
	checkErr(t, "adding lightweight tag", err)

	err = testRepository.AddLightweightTag("v2.0.0", hash2)
	checkErr(t, "adding lightweight tag", err)

	ref, err := testRepository.Head()
	checkErr(t, "fetching head", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, ref)
	checkErr(t, "building reachable commits", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet)
	checkErr(t, "fetching latest semver tag", err)

	want := "v2.0.0"
	assert.Equal(want, latest.Name, "should find latest lightweight tag")
}

func TestParser_FetchLatestSemverTag_MixedTags(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Add commits
	hash1, err := testRepository.AddCommit("fix")
	checkErr(t, "adding commit", err)

	hash2, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	hash3, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	// Add mixed tags: v1.0.0 (lightweight), v2.0.0 (annotated), v3.0.0 (lightweight)
	err = testRepository.AddLightweightTag("v1.0.0", hash1)
	checkErr(t, "adding lightweight tag", err)

	err = testRepository.AddTag("v2.0.0", hash2)
	checkErr(t, "adding annotated tag", err)

	err = testRepository.AddLightweightTag("v3.0.0", hash3)
	checkErr(t, "adding lightweight tag", err)

	ref, err := testRepository.Head()
	checkErr(t, "fetching head", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, ref)
	checkErr(t, "building reachable commits", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet)
	checkErr(t, "fetching latest semver tag", err)

	want := "v3.0.0"
	assert.Equal(want, latest.Name, "should find latest tag regardless of type")
}

func TestParser_ComputeNewSemver_UntaggedRepository_NoRelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	want := "0.0.0"

	assert.Equal(want, output.Semver.String(), "version should be equal")
}

func TestParser_ComputeNewSemver_UntaggedRepository_PatchRelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	_, err = testRepository.AddCommit("fix")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	want := "0.0.1"
	assert.Equal(want, output.Semver.String(), "version should be equal")
}

func TestParser_ComputeNewSemver_UnknownReleaseType(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	_, err = testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	invalidRules := map[string]string{
		"feat": "unknown",
		"fix":  "patch",
	}

	th := NewTestHelper(t)
	th.Ctx.RulesCfg = invalidRules
	parser := New(th.Ctx)

	_, err = parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	assert.ErrorContains(err, "unknown release type")
}

func TestParser_ComputeNewSemver_UntaggedRepository_MinorRelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	_, err = testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	want := "0.1.0"
	assert.Equal(want, output.Semver.String(), "version should be equal")
}

func TestParser_ComputeNewSemver_UntaggedRepository_MajorRelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	_, err = testRepository.AddCommit("feat!")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver ", err)

	want := "1.0.0"

	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_TaggedRepository(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	firstCommitHash, err := testRepository.AddCommit("feat!") // 1.0.0
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0", firstCommitHash)
	checkErr(t, "adding tag", err)

	_, err = testRepository.AddCommit("feat") // 1.1.0
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("fix") // Still at "1.1.0" as "feat" causes a "minor" bump which supplants the "patch" bump of "fix"
	checkErr(t, "adding commit", err)
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver ", err)

	want := "1.1.0"

	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_UninitializedRepository(t *testing.T) {
	assert := assertion.New(t)

	tempPath, err := os.MkdirTemp("", "parser-*")
	checkErr(t, "creating temporary directory", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(tempPath)
	})

	repository, err := git.PlainInit(tempPath, false)
	checkErr(t, "initializing repository", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	_, err = parser.ComputeNewSemver(repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	assert.ErrorIs(err, plumbing.ErrReferenceNotFound)
}

func TestParser_ComputeNewSemver_BuildMetadata(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	_, err = testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	th.Ctx.BuildMetadata = "metadata"
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	want := semver.Version{
		Major:    0,
		Minor:    1,
		Patch:    0,
		Metadata: "metadata",
	}

	assert.Equal(want.String(), output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_Prerelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	_, err = testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	th.Ctx.BranchesCfg[0].Prerelease = true
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	want := semver.Version{
		Major:            0,
		Minor:            1,
		Patch:            0,
		PrereleaseLabel:  "master",
		PrereleaseNumber: 1,
	}

	assert.Equal(want.String(), output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_PrereleaseBump(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create initial prerelease tag
	firstCommit, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("0.1.0-master.1", firstCommit)
	checkErr(t, "adding tag", err)

	// Add another commit
	_, err = testRepository.AddCommit("fix")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	th.Ctx.BranchesCfg[0].Prerelease = true
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	// Should bump prerelease number: 0.1.0-master.1 -> 0.1.0-master.2
	want := "0.1.0-master.2"
	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_PrereleaseAfterStable(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create stable tag
	firstCommit, err := testRepository.AddCommit("feat!")
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0", firstCommit)
	checkErr(t, "adding tag", err)

	// Add new feature commit
	_, err = testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	// Keep branch name as "master" since that's the actual branch in the repo
	th.Ctx.BranchesCfg[0].Prerelease = true
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	// Should create new prerelease for next minor: 1.1.0-master.1
	want := "1.1.0-master.1"
	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_PrereleaseBreakingChange(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create stable tag
	firstCommit, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0", firstCommit)
	checkErr(t, "adding tag", err)

	// Add breaking change commit
	_, err = testRepository.AddCommit("feat!")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	// Keep branch name as "master" since that's the actual branch in the repo
	th.Ctx.BranchesCfg[0].Prerelease = true
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	// Should create new prerelease for next major: 2.0.0-master.1
	want := "2.0.0-master.1"
	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_PrereleasePromotion(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create a prerelease tag (simulating a merge from a prerelease branch)
	firstCommit, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0-rc.3", firstCommit)
	checkErr(t, "adding tag", err)

	th := NewTestHelper(t)
	// Stable branch (Prerelease = false)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	// Should promote prerelease to stable: 1.0.0-rc.3 -> 1.0.0
	want := "1.0.0"
	assert.Equal(want, output.Semver.String(), "version should be promoted to stable")
	assert.Equal(true, output.NewRelease, "should be marked as new release")
}

func TestParser_ComputeNewSemver_PrereleasePromotionWithNewCommits(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create a prerelease tag
	firstCommit, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0-rc.3", firstCommit)
	checkErr(t, "adding tag", err)

	// Add a new patch commit after the prerelease tag
	_, err = testRepository.AddCommit("fix")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	// Stable branch (Prerelease = false)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	// Should bump patch and clear prerelease: 1.0.0-rc.3 -> 1.0.1
	want := "1.0.1"
	assert.Equal(want, output.Semver.String(), "version should be bumped and promoted")
	assert.Equal(true, output.NewRelease, "should be marked as new release")
}

// FIXME: the "origin" name is not set when calling parser.checkoutBranch leaving remoteRef like "ref/remote/<empty>/<branch>
func TestParser_Run_NoMonorepoOutputLength(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	_, err = testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	clonedTestRepository, err := testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.Run(clonedTestRepository.Repository)
	checkErr(t, "computing new semver", err)

	want := semver.Version{
		Major: 0,
		Minor: 1,
		Patch: 0,
	}

	assert.Len(output, 1, "parser run output should contain one element")
	assert.Equal(want.String(), output[0].Semver.String(), "version should be equal")
}

func TestParser_ShortMessage(t *testing.T) {
	assert := assertion.New(t)

	msg := "This is a very long commit message that is over fifty character"
	short := shortenMessage(msg)

	expected := "This is a very long commit message that is over..."

	assert.Equal(expected, short, "short message should be equal")
}

func TestMonorepoParser_FetchLatestSemverTagPerProjects(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	head, err := testRepository.Head()
	checkErr(t, "fetching head", err)

	wantTag := "foo-1.0.0"

	err = testRepository.AddTag(wantTag, head.Hash())
	checkErr(t, fmt.Sprintf("creating tag %q", wantTag), err)

	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building reachable commits", err)

	th := NewTestHelper(t)
	th.Ctx.MonorepositoryCfg = []monorepo.Item{
		{Name: "foo", Path: "foo"},
		{Name: "bar", Path: "bar"},
	}
	parser := New(th.Ctx)

	gotTag, err := parser.FetchLatestSemverTag(testRepository.Repository, th.Ctx.MonorepositoryCfg[0], reachable.HashSet)
	checkErr(t, "fetching latest semver tag", err)

	assert.Equal(gotTag.Name, wantTag, "should have found tag")
}

func TestMonorepoParser_CommitContainsProjectFiles_True(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	hash, err := testRepository.AddCommitWithSpecificFile("fix", "./foo/foo.txt")
	checkErr(t, "adding commit", err)

	commit, err := testRepository.CommitObject(hash)
	checkErr(t, "getting commit", err)

	monorepoItem := monorepo.Item{
		Name: "foo",
		Path: "foo",
	}

	contains, err := commitContainsProjectFiles(commit, monorepoItem)
	checkErr(t, "checking project files", err)

	assert.True(contains, "commit contains project files")
}

func TestMonorepoParser_CommitContainsProjectFiles_False(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	hash, err := testRepository.AddCommitWithSpecificFile("fix", "./foo/foo.txt")
	checkErr(t, "adding commit", err)

	commit, err := testRepository.CommitObject(hash)
	checkErr(t, "getting commit", err)

	monorepoItem := monorepo.Item{
		Name: "bar",
		Path: "bar",
	}

	contains, err := commitContainsProjectFiles(commit, monorepoItem)
	checkErr(t, "checking project files", err)

	assert.False(contains, "commit does not contain project files")
}

func TestParser_Run_Monorepo(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Adding "foo" project commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./foo/foo.txt")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./foo/xyz/foo.txt")
	checkErr(t, "adding commit", err)

	// Adding "bar" project commits
	_, err = testRepository.AddCommitWithSpecificFile("feat", "./bar/foo.txt")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/foo.txt")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/bar.txt")
	checkErr(t, "adding commit", err)

	// Adding unrelated commits
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./unknown/a.txt")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./temp/abc/b.txt")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	th.Ctx.MonorepositoryCfg = []monorepo.Item{
		{Name: "foo", Path: "foo"},
		{Name: "bar", Path: "bar"},
	}
	parser := New(th.Ctx)

	clonedTestRepository, err := testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err := parser.Run(clonedTestRepository.Repository)
	checkErr(t, "computing projects new semver", err)

	assert.Len(output, 2, "parser run output should contain two elements")

	gotSemver := []string{output[0].Semver.String(), output[1].Semver.String()}

	assert.Contains(gotSemver, "1.0.0")
	assert.Contains(gotSemver, "0.1.0")
}

func TestParser_Run_MonorepoWithPreexistingTags(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Add previous "foo" tags
	fooCommit, err := testRepository.AddCommitWithSpecificFile("feat!", "./foo/foo.txt") // foo-1.0.0
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("foo-1.0.0", fooCommit)
	checkErr(t, "adding foo tag", err)

	// Add previous "bar" tags
	barCommit, err := testRepository.AddCommitWithSpecificFile("chore!", "./bar/foo.txt")
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("bar-1.0.0", barCommit)
	checkErr(t, "adding bar tag", err)

	// Adding "foo" project commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./foo/foo.txt") // foo-2.0.0
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./foo/xyz/foo.txt") // foo-2.0.0
	checkErr(t, "adding commit", err)

	// Adding "bar" project commits
	_, err = testRepository.AddCommitWithSpecificFile("feat", "./bar/foo.txt") // bar-1.1.0
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/foo.txt") // bar-1.1.0
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/bar.txt") // bar-1.1.0
	checkErr(t, "adding commit", err)

	// Adding unrelated commits
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./unknown/a.txt")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./temp/abc/b.txt")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	th.Ctx.MonorepositoryCfg = []monorepo.Item{
		{Name: "foo", Path: "foo"},
		{Name: "bar", Path: "bar"},
	}
	parser := New(th.Ctx)

	clonedTestRepository, err := testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err := parser.Run(clonedTestRepository.Repository)
	checkErr(t, "computing projects new semver", err)

	assert.Len(output, 2, "parser run output should contain two elements")

	gotSemver := []string{output[0].Semver.String(), output[1].Semver.String()}

	assert.Contains(gotSemver, "2.0.0")
	assert.Contains(gotSemver, "1.1.0")
}

func TestParser_Run_InvalidBranch(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	th := NewTestHelper(t)
	th.Ctx.BranchesCfg = []branch.Item{{Name: "does_not_exist"}}

	parser := New(th.Ctx)

	_, err = parser.Run(testRepository.Repository)
	assert.ErrorIs(err, plumbing.ErrReferenceNotFound, "parser run should have failed since branch does not exist")
}

func checkErr(t *testing.T, msg string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", msg, err.Error())
	}
}

/*
func BenchmarkParser_ComputeNewSemver(b *testing.B) {

		parser := New(logger, rules)
		testRepository, err := gittest.NewRepository()
		if err != nil {
			b.Fatalf("creating test repository: %s", err)
		}

		b.Cleanup(func() {
			os.RemoveAll(testRepository.Path)
		})

		commitTypes := []string{"feat", "fix", "chore"}

		for i := 1; i <= 10000; i++ {
			commitType := commitTypes[rand.Intn(len(commitTypes))]
			testRepository.AddCommit(commitType)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{})
		}
	}
*/

func TestParser_FetchLatestSemverTag_DivergentBranches(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create initial commits on master
	commit1, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.0.0", commit1)
	checkErr(t, "adding tag 1.0.0", err)

	commit2, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.1.0", commit2)
	checkErr(t, "adding tag 1.1.0", err)

	// Create release branch from 1.1.0
	err = testRepository.CheckoutBranch("release-1.x")
	checkErr(t, "creating release branch", err)

	// Add commits and tags on release branch
	releaseCommit, err := testRepository.AddCommit("fix")
	checkErr(t, "adding release commit", err)
	err = testRepository.AddTag("1.1.1", releaseCommit)
	checkErr(t, "adding tag 1.1.1", err)

	// Switch back to master and add more commits/tags
	worktree, err := testRepository.Worktree()
	checkErr(t, "getting worktree", err)
	err = worktree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")})
	checkErr(t, "checkout master", err)

	commit3, err := testRepository.AddCommit("feat!")
	checkErr(t, "adding breaking change commit", err)
	err = testRepository.AddTag("2.0.0", commit3)
	checkErr(t, "adding tag 2.0.0", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	// Test: from master, should see 2.0.0 as latest
	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	masterReachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building master reachable commits", err)

	latestFromMaster, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{}, masterReachable.HashSet)
	checkErr(t, "fetching latest tag from master", err)
	assert.Equal("2.0.0", latestFromMaster.Name, "master should see 2.0.0")

	// Test: from release-1.x, should see 1.1.1 as latest (NOT 2.0.0)
	releaseRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("release-1.x"), true)
	checkErr(t, "getting release reference", err)

	releaseReachable, err := BuildReachableCommits(testRepository.Repository, releaseRef)
	checkErr(t, "building release reachable commits", err)

	latestFromRelease, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{}, releaseReachable.HashSet)
	checkErr(t, "fetching latest tag from release", err)
	assert.Equal("1.1.1", latestFromRelease.Name, "release-1.x should see 1.1.1, not 2.0.0")
}

func TestParser_FetchLatestSemverTag_UnreachableTag(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create a commit and tag on master
	commit1, err := testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.0.0", commit1)
	checkErr(t, "adding tag 1.0.0", err)

	// Create a second branch from initial commit
	err = testRepository.CheckoutBranch("other")
	checkErr(t, "creating other branch", err)

	otherCommit, err := testRepository.AddCommit("feat")
	checkErr(t, "adding other commit", err)
	err = testRepository.AddTag("9.9.9", otherCommit)
	checkErr(t, "adding unreachable tag 9.9.9", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	// From master, should see 1.0.0 (not 9.9.9 from other branch)
	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building reachable commits", err)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet)
	checkErr(t, "fetching latest tag", err)
	assert.Equal("1.0.0", latest.Name, "should only see reachable tag 1.0.0, not 9.9.9")
}

func TestParser_FetchLatestSemverTag_MonorepoDivergentBranches(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Initial foo project commit and tag
	fooCommit, err := testRepository.AddCommitWithSpecificFile("feat!", "./foo/foo.txt")
	checkErr(t, "adding foo commit", err)
	err = testRepository.AddTag("foo-1.0.0", fooCommit)
	checkErr(t, "adding foo-1.0.0 tag", err)

	// Create release branch
	err = testRepository.CheckoutBranch("release-1.x")
	checkErr(t, "creating release branch", err)

	// Add patch on release
	releaseFooCommit, err := testRepository.AddCommitWithSpecificFile("fix", "./foo/bar.txt")
	checkErr(t, "adding release foo commit", err)
	err = testRepository.AddTag("foo-1.0.1", releaseFooCommit)
	checkErr(t, "adding foo-1.0.1 tag", err)

	// Back to master, add major version
	worktree, err := testRepository.Worktree()
	checkErr(t, "getting worktree", err)
	err = worktree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")})
	checkErr(t, "checkout master", err)

	masterFooCommit, err := testRepository.AddCommitWithSpecificFile("feat!", "./foo/baz.txt")
	checkErr(t, "adding master foo commit", err)
	err = testRepository.AddTag("foo-2.0.0", masterFooCommit)
	checkErr(t, "adding foo-2.0.0 tag", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)
	fooProject := monorepo.Item{Name: "foo", Path: "foo"}

	// From release-1.x, should see foo-1.0.1
	releaseRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("release-1.x"), true)
	checkErr(t, "getting release reference", err)

	releaseReachable, err := BuildReachableCommits(testRepository.Repository, releaseRef)
	checkErr(t, "building release reachable commits", err)

	latestFromRelease, err := parser.FetchLatestSemverTag(testRepository.Repository, fooProject, releaseReachable.HashSet)
	checkErr(t, "fetching latest tag from release", err)
	assert.Equal("foo-1.0.1", latestFromRelease.Name, "release-1.x should see foo-1.0.1")

	// From master, should see foo-2.0.0
	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	masterReachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building master reachable commits", err)

	latestFromMaster, err := parser.FetchLatestSemverTag(testRepository.Repository, fooProject, masterReachable.HashSet)
	checkErr(t, "fetching latest tag from master", err)
	assert.Equal("foo-2.0.0", latestFromMaster.Name, "master should see foo-2.0.0")
}

type TestHelper struct {
	Ctx *appcontext.AppContext
}

func NewTestHelper(t *testing.T) *TestHelper {
	ctx := &appcontext.AppContext{
		RemoteName:  "origin",
		BranchesCfg: []branch.Item{{Name: "master"}},
		Logger:      zerolog.New(io.Discard),
	}

	defaultBranchRules, err := json.Marshal(rule.Default)
	if err != nil {
		t.Fatalf("marshalling default branch rules: %s", err)
	}

	err = ctx.RulesCfg.Set(string(defaultBranchRules))
	if err != nil {
		t.Fatalf("setting default branch rules: %s", err)
	}

	return &TestHelper{
		Ctx: ctx,
	}
}

func TestParser_SortBranches(t *testing.T) {
	assert := assertion.New(t)

	// Mixed order: prerelease, stable, prerelease
	branches := []branch.Item{
		{Name: "rc", Prerelease: true},
		{Name: "main", Prerelease: false},
		{Name: "beta", Prerelease: true},
		{Name: "develop", Prerelease: false},
	}

	sorted := sortBranches(branches)

	// Stable branches should come first
	assert.Len(sorted, 4)
	assert.False(sorted[0].Prerelease, "first branch should be stable")
	assert.False(sorted[1].Prerelease, "second branch should be stable")
	assert.True(sorted[2].Prerelease, "third branch should be prerelease")
	assert.True(sorted[3].Prerelease, "fourth branch should be prerelease")

	// Original order within groups should be preserved (stable sort)
	assert.Equal("main", sorted[0].Name)
	assert.Equal("develop", sorted[1].Name)
}

func TestParser_SortBranches_AllStable(t *testing.T) {
	assert := assertion.New(t)

	branches := []branch.Item{
		{Name: "main", Prerelease: false},
		{Name: "develop", Prerelease: false},
	}

	sorted := sortBranches(branches)

	assert.Len(sorted, 2)
	assert.Equal("main", sorted[0].Name)
	assert.Equal("develop", sorted[1].Name)
}

func TestParser_SortBranches_AllPrerelease(t *testing.T) {
	assert := assertion.New(t)

	branches := []branch.Item{
		{Name: "rc", Prerelease: true},
		{Name: "beta", Prerelease: true},
	}

	sorted := sortBranches(branches)

	assert.Len(sorted, 2)
	assert.Equal("rc", sorted[0].Name)
	assert.Equal("beta", sorted[1].Name)
}

func TestParser_SortBranches_Empty(t *testing.T) {
	assert := assertion.New(t)

	branches := []branch.Item{}
	sorted := sortBranches(branches)

	assert.Len(sorted, 0)
}

func TestParser_ComputeNewSemver_SingleBumpPerRelease_MultipleFixes(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	firstCommitHash, err := testRepository.AddCommit("feat!") // 1.0.0
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0", firstCommitHash)
	checkErr(t, "adding tag", err)

	// Add multiple fix commits - should result in single patch bump
	_, err = testRepository.AddCommit("fix: first fix")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("fix: second fix")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("fix: third fix")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	// Should be 1.0.1, not 1.0.3 (single bump, not cumulative)
	assert.Equal("1.0.1", output.Semver.String())
	assert.True(output.NewRelease)
}

func TestParser_ComputeNewSemver_SingleBumpPerRelease_MixedCommits(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	firstCommitHash, err := testRepository.AddCommit("feat!") // 1.0.0
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0", firstCommitHash)
	checkErr(t, "adding tag", err)

	// Add mixed commits - feat should win over fix
	_, err = testRepository.AddCommit("fix: a fix")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("feat: a feature")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("fix: another fix")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	// Should be 1.1.0 (minor bump wins over patches)
	assert.Equal("1.1.0", output.Semver.String())
	assert.True(output.NewRelease)
}

func TestParser_ComputeNewSemver_SingleBumpPerRelease_BreakingChangeWins(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	firstCommitHash, err := testRepository.AddCommit("feat!") // 1.0.0
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0", firstCommitHash)
	checkErr(t, "adding tag", err)

	// Add mixed commits - breaking change should win
	_, err = testRepository.AddCommit("fix: a fix")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("feat: a feature")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("feat!: breaking change")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("fix: another fix")
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	// Should be 2.0.0 (major bump wins over minor and patch)
	assert.Equal("2.0.0", output.Semver.String())
	assert.True(output.NewRelease)
}

func TestParser_PathBelongsTo(t *testing.T) {
	assert := assertion.New(t)

	// Standard case: file in directory
	assert.True(pathBelongsTo("api/handler.go", "api"))
	assert.True(pathBelongsTo("api/v2/handler.go", "api"))

	// Should NOT match similar prefixes
	assert.False(pathBelongsTo("api-v2/handler.go", "api"))
	assert.False(pathBelongsTo("api2/handler.go", "api"))
	assert.False(pathBelongsTo("apiserver/handler.go", "api"))

	// Exact match
	assert.True(pathBelongsTo("api", "api"))

	// Nested paths
	assert.True(pathBelongsTo("src/components/Button.tsx", "src/components"))
	assert.True(pathBelongsTo("src/components/forms/Input.tsx", "src/components"))
	assert.False(pathBelongsTo("src/componentsv2/Button.tsx", "src/components"))

	// Empty dir path does not match files (must have path boundary)
	assert.False(pathBelongsTo("file.go", ""))
	assert.False(pathBelongsTo("nested/file.go", ""))

	// Empty file path
	assert.False(pathBelongsTo("", "api"))

	// Both empty
	assert.True(pathBelongsTo("", ""))
}

func TestParser_FetchLatestPrereleaseTag_MultipleNumbers(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create commits and tags with increasing prerelease numbers
	commit1, err := testRepository.AddCommit("feat: first")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.0.0-rc.1", commit1)
	checkErr(t, "adding tag", err)

	commit2, err := testRepository.AddCommit("fix: second")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.0.0-rc.2", commit2)
	checkErr(t, "adding tag", err)

	commit3, err := testRepository.AddCommit("fix: third")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.0.0-rc.3", commit3)
	checkErr(t, "adding tag", err)

	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building reachable commits", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	coreVersion := &semver.Version{Major: 1, Minor: 0, Patch: 0}
	latest, err := parser.FetchLatestPrereleaseTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet, coreVersion, "rc")
	checkErr(t, "fetching latest prerelease tag", err)

	// Should return the highest prerelease number
	assert.NotNil(latest)
	assert.Equal("1.0.0-rc.3", latest.String())
}

func TestParser_FetchLatestPrereleaseTag_DifferentLabels(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	// Create commits and tags with different prerelease labels
	commit1, err := testRepository.AddCommit("feat: first")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.0.0-beta.1", commit1)
	checkErr(t, "adding tag", err)

	commit2, err := testRepository.AddCommit("fix: second")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.0.0-rc.1", commit2)
	checkErr(t, "adding tag", err)

	commit3, err := testRepository.AddCommit("fix: third")
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("1.0.0-beta.2", commit3)
	checkErr(t, "adding tag", err)

	masterRef, err := testRepository.Reference(plumbing.NewBranchReferenceName("master"), true)
	checkErr(t, "getting master reference", err)

	reachable, err := BuildReachableCommits(testRepository.Repository, masterRef)
	checkErr(t, "building reachable commits", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	coreVersion := &semver.Version{Major: 1, Minor: 0, Patch: 0}

	// Fetch only "rc" label
	latestRc, err := parser.FetchLatestPrereleaseTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet, coreVersion, "rc")
	checkErr(t, "fetching latest rc tag", err)

	assert.NotNil(latestRc)
	assert.Equal("1.0.0-rc.1", latestRc.String())

	// Fetch only "beta" label
	latestBeta, err := parser.FetchLatestPrereleaseTag(testRepository.Repository, monorepo.Item{}, reachable.HashSet, coreVersion, "beta")
	checkErr(t, "fetching latest beta tag", err)

	assert.NotNil(latestBeta)
	assert.Equal("1.0.0-beta.2", latestBeta.String())
}

package parser

import (
	"context"
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

	"github.com/s0ders/go-semver-release/v6/internal/appcontext"
	"github.com/s0ders/go-semver-release/v6/internal/branch"
	"github.com/s0ders/go-semver-release/v6/internal/gittest"
	"github.com/s0ders/go-semver-release/v6/internal/monorepo"
	"github.com/s0ders/go-semver-release/v6/internal/rule"
	"github.com/s0ders/go-semver-release/v6/internal/semver"
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

	th := NewTestHelper(t)

	parser := New(th.Ctx)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{})
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

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{})
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

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	latest, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Item{})
	checkErr(t, "fetching latest semver tag", err)

	want := "3.0.0"
	assert.Equal(want, latest.Name, "latest semver tag should be equal")
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
	_, err = testRepository.AddCommit("fix") // 1.1.1
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver ", err)

	want := "1.1.1"

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

	prereleaseID := "master"

	th := NewTestHelper(t)
	th.Ctx.BranchesCfg[0].Prerelease = true
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Item{}, th.Ctx.BranchesCfg[0])
	checkErr(t, "computing new semver", err)

	want := semver.Version{
		Major:      0,
		Minor:      1,
		Patch:      0,
		Prerelease: prereleaseID,
	}

	assert.Equal(want.String(), output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
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

	output, err := parser.Run(context.Background(), clonedTestRepository.Repository)
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

	th := NewTestHelper(t)
	th.Ctx.MonorepositoryCfg = []monorepo.Item{
		{Name: "foo", Path: "foo"},
		{Name: "bar", Path: "bar"},
	}
	parser := New(th.Ctx)

	gotTag, err := parser.FetchLatestSemverTag(testRepository.Repository, th.Ctx.MonorepositoryCfg[0])
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

	output, err := parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing projects new semver", err)

	assert.Len(output, 2, "parser run output should contain two elements")

	gotSemver := []string{output[0].Semver.String(), output[1].Semver.String()}

	assert.Contains(gotSemver, "1.0.1")
	assert.Contains(gotSemver, "0.1.2")
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

	err = testRepository.AddTag("bar-1.0.0", barCommit) // bar-1.0.0
	checkErr(t, "adding bar tag", err)

	// Adding "foo" project commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./foo/foo.txt") // foo-2.0.0
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./foo/xyz/foo.txt") // foo-2.0.1
	checkErr(t, "adding commit", err)

	// Adding "bar" project commits
	_, err = testRepository.AddCommitWithSpecificFile("feat", "./bar/foo.txt") // bar-1.1.0
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/foo.txt") // bar-1.1.1
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/bar.txt") // bar-1.1.2
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

	output, err := parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing projects new semver", err)

	assert.Len(output, 2, "parser run output should contain two elements")

	gotSemver := []string{output[0].Semver.String(), output[1].Semver.String()}

	assert.Contains(gotSemver, "2.0.1")
	assert.Contains(gotSemver, "1.1.2")
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

	_, err = parser.Run(context.Background(), testRepository.Repository)
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

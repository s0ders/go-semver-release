package parser

import (
	"context"
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

	latestTagInfo, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{}, nil)
	checkErr(t, "fetching latest semver tag", err)

	assert.Nil(latestTagInfo.Semver, "latest semver struct should be nil")
	assert.Nil(latestTagInfo.Semver, "latest semver tag should be nil")
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

	latestTagInfo, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{}, nil)
	checkErr(t, "fetching latest semver tag", err)

	wantSemver := &semver.Version{Major: 1, Minor: 0, Patch: 0}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(tagName, latestTagInfo.Name, "latest semver tagName should be equal")
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

	tags := []string{
		"2.0.0",
		"2.0.1",
		"4.0.0-beta.2",
		"3.0.0",
		"2.5.0",
		"4.0.0-beta.1",
		"0.0.2",
		"0.0.1",
		"0.1.0",
		"5.0.0-alpha.2",
		"1.0.0",
		"5.0.0-alpha.3",
	}

	for _, v := range tags {
		err = testRepository.AddTag(v, head.Hash())
		checkErr(t, "creating tag", err)
	}

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	// test main branch
	latestTagInfo, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{}, nil)
	checkErr(t, "fetching latest semver tag", err)

	want := "3.0.0"
	wantSemver := &semver.Version{Major: 3, Minor: 0, Patch: 0}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(want, latestTagInfo.Name, "latest semver tag should be equal")

	// test beta branch
	latestTagInfo, err = parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{Name: "beta", Prerelease: true}, nil)
	checkErr(t, "fetching latest semver tag", err)

	want = "4.0.0-beta.2"
	wantSemver = &semver.Version{Major: 4, Minor: 0, Patch: 0, Prerelease: &semver.Prerelease{Name: "beta", Build: 2}}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(want, latestTagInfo.Name, "latest semver tag should be equal")

	// test alpha branch
	latestTagInfo, err = parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{Name: "alpha", Prerelease: true}, nil)
	checkErr(t, "fetching latest semver tag", err)

	want = "5.0.0-alpha.3"
	wantSemver = &semver.Version{Major: 5, Minor: 0, Patch: 0, Prerelease: &semver.Prerelease{Name: "alpha", Build: 3}}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(want, latestTagInfo.Name, "latest semver tag should be equal")
}

func TestParser_FetchLatestSemverTag_MultipleTags_NewPrereleaseBranch(t *testing.T) {
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

	tags := []string{
		"2.0.0",
		"2.0.1",
		"3.0.0",
		"2.5.0",
		"0.0.2",
		"0.0.1",
		"0.1.0",
		"1.0.0",
	}

	for _, v := range tags {
		err = testRepository.AddTag(v, head.Hash())
		checkErr(t, "creating tag", err)
	}

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	// test beta branch
	latestTagInfo, err := parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{Name: "beta", Prerelease: true}, nil)
	checkErr(t, "fetching latest semver tag", err)

	want := "3.0.0"
	wantSemver := &semver.Version{Major: 3, Minor: 0, Patch: 0}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(want, latestTagInfo.Name, "latest semver tag should be equal")

	// test new beta branch
	latestTagInfo, err = parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{Name: "beta", Prerelease: true}, nil)
	checkErr(t, "fetching latest semver tag", err)

	want = "3.0.0"
	wantSemver = &semver.Version{Major: 3, Minor: 0, Patch: 0}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(want, latestTagInfo.Name, "latest semver tag should be equal")

	// test beta branch with releases
	tags = []string{
		"4.0.0-beta.5",
		"4.0.0-beta.1",
		"4.0.0-beta.3",
	}

	for _, v := range tags {
		err = testRepository.AddTag(v, head.Hash())
		checkErr(t, "creating tag", err)
	}

	latestTagInfo, err = parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{Name: "beta", Prerelease: true}, nil)
	checkErr(t, "fetching latest semver tag", err)

	want = "4.0.0-beta.5"
	wantSemver = &semver.Version{Major: 4, Minor: 0, Patch: 0, Prerelease: &semver.Prerelease{Name: "beta", Build: 5}}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(want, latestTagInfo.Name, "latest semver tag should be equal")

	// test new alpha branch
	latestTagInfo, err = parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{Name: "alpha", Prerelease: true}, nil)
	checkErr(t, "fetching latest semver tag", err)

	want = "4.0.0-beta.5"
	wantSemver = &semver.Version{Major: 4, Minor: 0, Patch: 0, Prerelease: &semver.Prerelease{Name: "beta", Build: 5}}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(want, latestTagInfo.Name, "latest semver tag should be equal")

	// test alpha branch with releases
	tags = []string{
		"5.0.0-alpha.5",
		"5.0.0-alpha.1",
		"5.0.0-alpha.3",
	}

	for _, v := range tags {
		err = testRepository.AddTag(v, head.Hash())
		checkErr(t, "creating tag", err)
	}

	latestTagInfo, err = parser.FetchLatestSemverTag(testRepository.Repository, monorepo.Project{}, branch.Branch{Name: "alpha", Prerelease: true}, nil)
	checkErr(t, "fetching latest semver tag", err)

	want = "5.0.0-alpha.5"
	wantSemver = &semver.Version{Major: 5, Minor: 0, Patch: 0, Prerelease: &semver.Prerelease{Name: "alpha", Build: 5}}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(want, latestTagInfo.Name, "latest semver tag should be equal")
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

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
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

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
	checkErr(t, "computing new semver", err)

	want := "0.1.0"
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

	invalidRules := rule.Rules{
		Map: map[string]string{
			"feat": "unknown",
			"fix":  "patch",
		},
	}

	th := NewTestHelper(t)
	th.Ctx.Rules = invalidRules
	parser := New(th.Ctx)

	_, err = parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
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

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
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

	_, err = testRepository.AddCommit("feat!") // 0.1.0
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
	checkErr(t, "computing new semver ", err)

	want := "0.1.0"

	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_UntaggedRepository_Prerelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	_, err = testRepository.AddCommit("feat!") // 0.1.0-beata.1
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[1], nil)
	checkErr(t, "computing new semver ", err)

	want := "0.1.0-beta.1"

	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_TaggedRepository_PatchRelease(t *testing.T) {
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

	_, err = testRepository.AddCommit("fix") // 1.0.1
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
	checkErr(t, "computing new semver ", err)

	want := "1.0.1"

	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_TaggedRepository_PatchRelease_MinorRelease(t *testing.T) {
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
	_, err = testRepository.AddCommit("fix") // 1.1.0
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
	checkErr(t, "computing new semver ", err)

	want := "1.1.0"

	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_TaggedRepository_PatchRelease_MajorRelease(t *testing.T) {
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

	_, err = testRepository.AddCommit("feat!") // 2.0.0
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("fix") // 2.0.0
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
	checkErr(t, "computing new semver ", err)

	want := "2.0.0"

	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_TaggedRepository_PatchRelease_Prerelease(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	firstCommitHash, err := testRepository.AddCommit("feat!") // 1.0.0-beta.1
	checkErr(t, "adding commit", err)

	err = testRepository.AddTag("1.0.0", firstCommitHash)
	checkErr(t, "adding tag", err)

	_, err = testRepository.AddCommit("feat!") // 2.0.0-beta.1
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommit("fix") // 2.0.0-beta.1
	checkErr(t, "adding commit", err)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[1], nil)
	checkErr(t, "computing new semver ", err)

	want := "2.0.0-beta.1"

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

	_, err = parser.ComputeNewSemver(repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
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
	th.Ctx.BuildMetadataFlag = "metadata"
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[0], nil)
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
	parser := New(th.Ctx)

	output, err := parser.ComputeNewSemver(testRepository.Repository, monorepo.Project{}, th.Ctx.Branches[1], nil)
	checkErr(t, "computing new semver", err)

	want := semver.Version{
		Major:      0,
		Minor:      1,
		Patch:      0,
		Prerelease: &semver.Prerelease{Name: th.Ctx.Branches[1].Name, Build: 1},
	}

	assert.Equal(want.String(), output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

// FIXME: the "origin" name is not set when calling parser.checkoutBranch leaving remoteRef like "ref/remote/<empty>/<branch>
func TestParser_Run_NoMonorepo(t *testing.T) {
	assert := assertion.New(t)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	th.CreateAllBranches(t, testRepository)

	// only main channel has a release
	_, err = testRepository.AddCommit("feat")
	checkErr(t, "adding commit", err)

	clonedTestRepository, err := testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err := parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing new semver", err)

	assert.Len(output, 3, "parser run output should contain two element")

	assert.Equal("0.1.0", output[0].Semver.String(), "version should be equal")
	assert.True(output[0].NewRelease, "should be a new release")
	assert.Equal("0.0.0-beta.1", output[1].Semver.String(), "version should be equal")
	assert.False(output[1].NewRelease, "should be no new release")
	assert.Equal("0.0.0-alpha.1", output[2].Semver.String(), "version should be equal")
	assert.False(output[2].NewRelease, "should be no new release")

	// main and alpha has a release
	_, err = testRepository.AddCommitToBranch("feat", "alpha")
	checkErr(t, "adding commit", err)

	clonedTestRepository, err = testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err = parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing new semver", err)

	assert.Equal("0.1.0", output[0].Semver.String(), "version should be equal")
	assert.True(output[0].NewRelease, "should be a new release")
	assert.Equal("0.0.0-beta.1", output[1].Semver.String(), "version should be equal")
	assert.False(output[1].NewRelease, "should be no new release")
	assert.Equal("0.2.0-alpha.1", output[2].Semver.String(), "version should be equal")
	assert.True(output[2].NewRelease, "should be a new release")

	// main, beta and alpha have a release
	_, err = testRepository.AddCommitToBranch("feat!", "beta")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitToBranch("fix", "alpha")
	checkErr(t, "adding commit", err)

	clonedTestRepository, err = testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err = parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing new semver", err)

	assert.Equal("0.1.0", output[0].Semver.String(), "version should be equal")
	assert.True(output[0].NewRelease, "should be a new release")
	assert.Equal("1.0.0-beta.1", output[1].Semver.String(), "version should be equal")
	assert.True(output[1].NewRelease, "should be a new release")
	assert.Equal("1.0.1-alpha.1", output[2].Semver.String(), "version should be equal")
	assert.True(output[2].NewRelease, "should be a new release")
}

func TestParser_Run_NoMonorepoWithPreexistingTags(t *testing.T) {
	assert := assertion.New(t)

	th := NewTestHelper(t)
	parser := New(th.Ctx)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	th.CreateAllBranches(t, testRepository)

	// main has a release tag
	err = testRepository.AddTagToBranch("0.1.2", "main")
	checkErr(t, "adding foo tag", err)

	_, err = testRepository.AddCommitToBranch("feat", "main")
	checkErr(t, "adding commit", err)

	clonedTestRepository, err := testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err := parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing new semver", err)

	assert.Equal("0.2.0", output[0].Semver.String(), "version should be equal")
	assert.True(output[0].NewRelease, "should be a new release")
	assert.Equal("0.1.2", output[1].Semver.String(), "version should be equal")
	assert.False(output[1].NewRelease, "should be no new release")
	assert.Equal("0.1.2", output[2].Semver.String(), "version should be equal")
	assert.False(output[2].NewRelease, "should be no new release")

	// main and alpha have a release tag
	err = testRepository.AddTagToBranch("0.3.0-alpha.5", "alpha")
	checkErr(t, "adding foo tag", err)

	_, err = testRepository.AddCommitToBranch("fix", "alpha")
	checkErr(t, "adding commit", err)

	clonedTestRepository, err = testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err = parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing new semver", err)

	assert.Equal("0.2.0", output[0].Semver.String(), "version should be equal")
	assert.True(output[0].NewRelease, "should be a new release")
	assert.Equal("0.1.2", output[1].Semver.String(), "version should be equal")
	assert.False(output[1].NewRelease, "should be no new release")
	assert.Equal("0.3.0-alpha.6", output[2].Semver.String(), "version should be equal")
	assert.True(output[2].NewRelease, "should be a new release")

	// main, beta and alpha have a release tag
	err = testRepository.AddTagToBranch("0.4.0-beta.2", "beta")
	checkErr(t, "adding foo tag", err)

	_, err = testRepository.AddCommitToBranch("feat!", "beta")
	checkErr(t, "adding commit", err)

	clonedTestRepository, err = testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err = parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing new semver", err)

	assert.Equal("0.2.0", output[0].Semver.String(), "version should be equal")
	assert.True(output[0].NewRelease, "should be a new release")
	assert.Equal("1.0.0-beta.1", output[1].Semver.String(), "version should be equal")
	assert.True(output[1].NewRelease, "should be a new release")
	assert.Equal("0.4.1-alpha.1", output[2].Semver.String(), "version should be equal")
	assert.True(output[2].NewRelease, "should be a new release")
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
	th.Ctx.Projects = []monorepo.Project{
		{Name: "foo", Path: "foo"},
		{Name: "bar", Path: "bar"},
	}
	parser := New(th.Ctx)

	latestTagInfo, err := parser.FetchLatestSemverTag(testRepository.Repository, th.Ctx.Projects[0], branch.Branch{}, nil)
	checkErr(t, "fetching latest semver tag", err)

	wantSemver := &semver.Version{Major: 1, Minor: 0, Patch: 0}

	assert.Equal(wantSemver, latestTagInfo.Semver, "latest semver struct should be equal")
	assert.Equal(wantTag, latestTagInfo.Name, "should have found tag")
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

	contains, err := commitContainsProjectFiles(commit, "foo")
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

	contains, err := commitContainsProjectFiles(commit, "bar")
	checkErr(t, "checking project files", err)

	assert.False(contains, "commit does not contain project files")
}

func TestParser_Run_Monorepo(t *testing.T) {
	assert := assertion.New(t)

	th := NewTestHelper(t)
	th.Ctx.Projects = []monorepo.Project{
		{Name: "foo", Path: "foo"},
		{Name: "bar", Path: "bar"},
	}
	parser := New(th.Ctx)

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

	// Adding unrelated commits
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./unknown/a.txt")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./temp/abc/b.txt")
	checkErr(t, "adding commit", err)

	// branch beta from main
	err = testRepository.CheckoutBranch("beta", true)
	checkErr(t, "checkout beta branch", err)

	// Adding "bar" project commits
	_, err = testRepository.AddCommitWithSpecificFile("feat", "./bar/foo.txt")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/foo.txt")
	checkErr(t, "adding commit", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/bar.txt")
	checkErr(t, "adding commit", err)

	// branch alpha from beta
	err = testRepository.CheckoutBranch("alpha", true)
	checkErr(t, "checkout alpha branch", err)

	// Adding "foo" project commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./foo/foo.txt")
	checkErr(t, "adding commit", err)

	clonedTestRepository, err := testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err := parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing projects new semver", err)

	assert.Len(output, 6, "parser run output should contain four elements")

	// main foo
	assert.Equal("0.1.0", output[0].Semver.String())
	assert.True(output[0].NewRelease, "should be a new release")
	// main bar
	assert.Equal("0.0.0", output[1].Semver.String())
	assert.False(output[1].NewRelease, "should be no new release")
	// beta foo
	assert.Equal("0.1.0", output[2].Semver.String())
	assert.False(output[2].NewRelease, "should be no new release")
	// beta bar
	assert.Equal("0.1.0-beta.1", output[3].Semver.String())
	assert.True(output[3].NewRelease, "should be a new release")
	// alpha foo
	assert.Equal("1.0.0-alpha.1", output[4].Semver.String())
	assert.True(output[4].NewRelease, "should be a new release")
	// alpha bar
	assert.Equal("0.1.0-beta.1", output[5].Semver.String())
	assert.False(output[5].NewRelease, "should be no new release")
}

func TestParser_Run_MonorepoWithPreexistingTags(t *testing.T) {
	assert := assertion.New(t)

	th := NewTestHelper(t)
	th.Ctx.Projects = []monorepo.Project{
		{Name: "foo", Path: "foo"},
		{Name: "bar", Path: "bar"},
	}
	parser := New(th.Ctx)

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
	barCommit, err := testRepository.AddCommitWithSpecificFile("chore!", "./bar/foo.txt") // bar-1.0.0
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

	// branch beta from main
	err = testRepository.CheckoutBranch("beta", true)
	checkErr(t, "checkout beta branch", err)

	// Adding "foo" project beta commits
	fooCommit, err = testRepository.AddCommitWithSpecificFile("feat", "./foo/foo.txt") // foo-2.1.0-beta.1
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("foo-2.1.0-beta.1", fooCommit)
	checkErr(t, "adding foo tag", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./foo/xyz/foo.txt") // foo-2.1.0-beta.2
	checkErr(t, "adding commit", err)

	// Adding "bar" beta commits
	barCommit, err = testRepository.AddCommitWithSpecificFile("feat!", "./bar/foo.txt") // bar-2.0.0-beta.1
	checkErr(t, "adding commit", err)
	err = testRepository.AddTag("bar-2.0.0-beta.1", barCommit)
	checkErr(t, "adding foo tag", err)
	_, err = testRepository.AddCommitWithSpecificFile("fix", "./bar/baz/xyz/foo.txt") // bar-2.0.0-beta.2
	checkErr(t, "adding commit", err)

	// branch alpha from beta
	err = testRepository.CheckoutBranch("alpha", true)
	checkErr(t, "checkout alpha branch", err)

	// Adding "bar" alpha commits
	_, err = testRepository.AddCommitWithSpecificFile("feat!", "./bar/foo.txt") // bar-3.0.0-beta.1
	checkErr(t, "adding commit", err)

	clonedTestRepository, err := testRepository.Clone()
	checkErr(t, "cloning test repository", err)

	output, err := parser.Run(context.Background(), clonedTestRepository.Repository)
	checkErr(t, "computing projects new semver", err)

	assert.Len(output, 6, "parser run output should contain four elements")

	// main foo
	assert.Equal("2.0.0", output[0].Semver.String())
	assert.True(output[0].NewRelease, "should be a new release")
	// main bar
	assert.Equal("1.1.0", output[1].Semver.String())
	assert.True(output[1].NewRelease, "should be a new release")
	// beta foo
	assert.Equal("2.1.0-beta.2", output[2].Semver.String())
	assert.True(output[2].NewRelease, "should be a new release")
	// beta bar
	assert.Equal("2.0.0-beta.2", output[3].Semver.String())
	assert.True(output[3].NewRelease, "should be a new release")
	// alpha foo
	assert.Equal("2.1.0-beta.2", output[4].Semver.String())
	assert.False(output[4].NewRelease, "should be no new release")
	// alpha bar
	assert.Equal("3.0.0-alpha.1", output[5].Semver.String())
	assert.True(output[5].NewRelease, "should be a new release")
}

func TestParser_Run_InvalidBranch(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	th := NewTestHelper(t)
	th.Ctx.Branches = []branch.Branch{{Name: "does_not_exist"}}

	parser := New(th.Ctx)

	output, err := parser.Run(context.Background(), testRepository.Repository)
	checkErr(t, "running parser", err)
	assert.ErrorIs(output[0].Error, plumbing.ErrReferenceNotFound, "output should contain corresponding checkout error since branch does not exist")

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
		Rules:          rule.Default,
		RemoteNameFlag: "origin",
		Branches: []branch.Branch{
			{Name: "main"},
			{Name: "beta", Prerelease: true},
			{Name: "alpha", Prerelease: true},
		},
		Logger: zerolog.New(io.Discard),
	}

	return &TestHelper{
		Ctx: ctx,
	}
}

func (th *TestHelper) CreateAllBranches(t *testing.T, r *gittest.TestRepository) {
	for _, branch := range th.Ctx.Branches {
		if branch.Name == "main" {
			continue
		}
		err := r.CheckoutBranch(branch.Name, true)
		checkErr(t, fmt.Sprintf("adding %s branch", branch.Name), err)
	}
	err := r.CheckoutBranch("main", false)
	checkErr(t, "checkout main branch", err)
}

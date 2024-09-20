package parser

import (
	"fmt"
	"testing"

	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v4/internal/gittest"
	"github.com/s0ders/go-semver-release/v4/internal/monorepo"
)

var (
	projects = []monorepo.Project{{Name: "foo", Path: "foo"}, {Name: "bar", Path: "bar"}}
)

func TestMonorepoParser_FetchLatestSemverTagPerProjects(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	head, err := testRepository.Head()
	checkErr(t, "fetching head", err)

	wantTags := []string{"foo-1.0.0", "bar-0.0.2"}
	gotTags := make([]string, 0, len(wantTags))

	for _, tag := range wantTags {
		err = testRepository.AddTag(tag, head.Hash())
		checkErr(t, fmt.Sprintf("creating tag %q", tag), err)
	}

	parser := New(logger, tagger, rules, WithProjects(projects))

	latestTags, err := parser.FetchLatestSemverTagPerProjects(testRepository.Repository)
	checkErr(t, "fetching latest semver tag", err)

	for _, latestTag := range latestTags {
		gotTags = append(gotTags, latestTag.Name)
	}

	assert.Contains(gotTags, wantTags[0], "should have found tag")
	assert.Contains(gotTags, wantTags[1], "should have found tag")
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

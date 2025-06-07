package remote

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/s0ders/go-semver-release/v6/internal/tag"

	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v6/internal/gittest"
)

func TestRemote_Clone_HappyScenario(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating test repository")

	defer func() {
		err = testRepository.Remove()
		checkErr(t, err, "removing test repository")
	}()

	remote := New("origin", "password")

	clonedRepository, err := remote.Clone(testRepository.Path)
	checkErr(t, err, "cloning repository")

	assert.NotNil(clonedRepository)
	assert.NoError(err)
}

func TestRemote_Clone_NonExistingPath(t *testing.T) {
	assert := assertion.New(t)

	remote := New("origin", "password")
	clonedRepository, err := remote.Clone("https://example.com")

	assert.Nil(clonedRepository)
	assert.Error(err)
}

func TestRemote_PushTag(t *testing.T) {
	assert := assertion.New(t)

	tagName := "v1.0.0"

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating test repository")

	defer func() {
		err = testRepository.Remove()
		checkErr(t, err, "removing test repository")
	}()

	commitHash, err := testRepository.AddCommit("fix")
	checkErr(t, err, "adding commit to test repository")

	remote := New("origin", "password")

	clonedRepository, err := remote.Clone(testRepository.Path)
	checkErr(t, err, "cloning repository")

	_, err = clonedRepository.CreateTag(tagName, commitHash, &git.CreateTagOptions{
		Message: tagName,
		Tagger: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  time.Now(),
		},
	})
	checkErr(t, err, "creating tag on cloned repository")

	err = remote.PushTag(tagName)
	checkErr(t, err, "pushing tag to remote")

	assert.True(tag.Exists(testRepository.Repository, tagName))
}

func TestRemote_PushTag_UnavailableRemote(t *testing.T) {
	assert := assertion.New(t)

	tagName := "v1.0.0"

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating test repository")

	commitHash, err := testRepository.AddCommit("fix")
	checkErr(t, err, "adding commit to test repository")

	remote := New("origin", "password")

	clonedRepository, err := remote.Clone(testRepository.Path)
	checkErr(t, err, "cloning repository")

	_, err = clonedRepository.CreateTag(tagName, commitHash, &git.CreateTagOptions{
		Message: tagName,
		Tagger: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  time.Now(),
		},
	})
	checkErr(t, err, "creating tag on cloned repository")

	// Removing remote
	err = testRepository.Remove()
	checkErr(t, err, "removing test repository")

	err = remote.PushTag("v1.0.0")

	assert.Error(err)
}

func checkErr(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", msg, err)
	}
}

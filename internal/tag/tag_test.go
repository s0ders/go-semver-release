package tag

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/s0ders/go-semver-release/v2/internal/semver"
	"github.com/stretchr/testify/assert"
)

func TestTagger_TagExists(t *testing.T) {
	assert := assert.New(t)
	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	assert.NoError(err, "repository creation should have succeeded")

	defer func() {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "failed to remove repository")
	}()

	head, err := repository.Head()
	assert.NoError(err, "should have fetched HEAD")

	tags := []string{"1.0.0", "1.0.2"}

	for i, tag := range tags {
		_, err := repository.CreateTag(tag, head.Hash(), &git.CreateTagOptions{
			Message: tag,
			Tagger: &object.Signature{
				Name:  "Go Semver Release",
				Email: "ci@ci.ci",
				When:  time.Now().Add(time.Duration(i) * time.Hour),
			},
		})
		assert.NoError(err, "tag creation should have succeeded")
	}

	tagExists, err := TagExists(repository, tags[0])
	assert.NoError(err, "should have been able to check if tag exists")
	assert.Equal(tagExists, true, "tag should have been found")

	tagDoesNotExists, err := TagExists(repository, "0.0.1")
	assert.NoError(err, "should have been able to check if tag exists")
	assert.Equal(tagDoesNotExists, false, "tag should not have been found")
}

func TestTagger_AddTagToRepository(t *testing.T) {
	assert := assert.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	assert.NoError(err, "repository creation should have succeeded")

	defer func() {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "failed to remove repository")
	}()

	version, err := semver.New(1, 0, 0)
	assert.NoError(err, "semver creation should have succeeded")

	err = AddTagToRepository(repository, version, "")
	assert.NoError(err, "should have been able to add tag to repository")

	tagExists, err := TagExists(repository, version.String())
	assert.NoError(err, "should have been able to check if tag exists")

	assert.Equal(tagExists, true, "tag should have been found")
}

func TestTagger_AddExistingTagToRepository(t *testing.T) {
	assert := assert.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	assert.NoError(err, "repository creation should have succeeded")

	defer func() {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "failed to remove repository")
	}()

	version, err := semver.New(1, 0, 0)
	assert.NoError(err, "semver creation should have succeeded")

	err = AddTagToRepository(repository, version, "v")
	assert.NoError(err, "should not have been able to add tag to repository")

	err = AddTagToRepository(repository, version, "v")
	assert.Error(err, "should not have been able to add tag to repository")
}

func TestTagger_NewTagFromServer(t *testing.T) {
	assert := assert.New(t)

	var b [20]byte
	for i := range 20 {
		b[i] = byte(i)
	}

	hash := plumbing.Hash(b)

	version, err := semver.New(0, 0, 1)
	assert.NoError(err, "semver creation should have succeeded")

	gotTag := NewTagFromSemver(version, hash)

	wantTag := &object.Tag{
		Hash:   hash,
		Name:   version.String(),
		Tagger: GitSignature,
	}

	assert.Equal(*gotTag, *wantTag, "tag should match")
}

// TODO: replace by a mock ?
// createGitRepository creates an empty Git repository, adds a file to it then creates
// a commit with the given message.
func createGitRepository(firstCommitMessage string) (*git.Repository, string, error) {
	tempDirPath, err := os.MkdirTemp("", "tag-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	r, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize git repository: %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get worktree: %s", err)
	}

	tempFileName := "temp"
	tempFilePath := filepath.Join(tempDirPath, tempFileName)
	fileDescriptor, err := os.Create(tempFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp file: %s", err)
	}

	defer func() {
		_ = fileDescriptor.Close()
	}()

	err = os.WriteFile(tempFilePath, []byte("Hello world"), 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("failed to write to temp file: %s", err)
	}

	_, err = w.Add(tempFileName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to add temp file to worktree: %s", err)
	}

	commit, err := w.Commit(firstCommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create commit object %s", err)
	}

	if _, err = r.CommitObject(commit); err != nil {
		return nil, "", fmt.Errorf("failed to commit object %s", err)
	}

	return r, tempDirPath, nil
}
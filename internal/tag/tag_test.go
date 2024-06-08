package tag

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v2/internal/semver"
)

var (
	taggerName  = "Go Semver Release"
	taggerEmail = "go-semver@release.ci"
)

func TestTag_TagExists(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	head, err := repository.Head()
	checkErr(t, "fetching head", err)

	tags := []string{"1.0.0", "1.0.2"}

	for i, tag := range tags {
		_, err = repository.CreateTag(tag, head.Hash(), &git.CreateTagOptions{
			Message: tag,
			Tagger: &object.Signature{
				Name:  taggerName,
				Email: taggerEmail,
				When:  time.Now().Add(time.Duration(i) * time.Hour),
			},
		})
		checkErr(t, "creating tag", err)
	}

	tagExists, err := Exists(repository, tags[0])
	checkErr(t, "checking if tag exists", err)

	assert.Equal(tagExists, true, "tag should have been found")

	tagDoesNotExists, err := Exists(repository, "0.0.1")
	checkErr(t, "checking if tag exists", err)

	assert.Equal(tagDoesNotExists, false, "tag should not have been found")
}

func TestTag_AddTagToRepository(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	head, err := repository.Head()
	checkErr(t, "fetching head", err)

	version := &semver.Semver{Major: 1}

	tagger := NewTagger(taggerName, taggerEmail)

	err = tagger.TagRepository(repository, version, head.Hash())
	checkErr(t, "tagging repository", err)

	tagExists, err := Exists(repository, version.String())
	checkErr(t, "checking if tag exists", err)

	assert.Equal(tagExists, true, "tag should have been found")
}

func TestTag_AddExistingTagToRepository(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	head, err := repository.Head()
	checkErr(t, "fetching head", err)

	version := &semver.Semver{Major: 1}

	tagger := NewTagger(taggerName, taggerEmail)

	err = tagger.TagRepository(repository, version, head.Hash())
	checkErr(t, "tagging repository", err)

	err = tagger.TagRepository(repository, version, head.Hash())
	assert.Error(err, "should not have been able to add tag to repository")
}

func TestTag_NewTagFromServer(t *testing.T) {
	assert := assertion.New(t)

	var b [20]byte
	for i := range 20 {
		b[i] = byte(i)
	}

	hash := plumbing.Hash(b)

	version := &semver.Semver{Patch: 1}

	tagger := NewTagger(taggerName, taggerEmail)

	gotTag := tagger.TagFromSemver(version, hash)

	wantTag := &object.Tag{
		Hash:   hash,
		Name:   version.String(),
		Tagger: tagger.GitSignature,
	}

	assert.Equal(*gotTag, *wantTag)
}

func TestTag_AddToRepositoryWithNoHead(t *testing.T) {
	assert := assertion.New(t)

	tempDirPath, err := os.MkdirTemp("", "tag-*")
	checkErr(t, "creating temporary directory", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(tempDirPath)
	})

	repository, err := git.PlainInit(tempDirPath, false)
	checkErr(t, "initializing repository", err)

	tagger := NewTagger(taggerName, taggerEmail)

	err = tagger.TagRepository(repository, &semver.Semver{}, plumbing.Hash{})
	assert.Error(err, "should have failed trying to fetch uninitialized repository head")
}

func TestTag_SignKey(t *testing.T) {
	assert := assertion.New(t)

	config := &packet.Config{
		Algorithm: packet.PubKeyAlgoRSA,
	}

	entity, err := openpgp.NewEntity("John Doe", "", "john.doe@example.com", config)
	checkErr(t, "creating openpgp entity", err)

	repository, repositoryPath, err := createGitRepository("fix: ...")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	head, err := repository.Head()
	checkErr(t, "fetching head", err)

	version := &semver.Semver{Major: 1}

	tagger := NewTagger(taggerName, taggerEmail, WithSignKey(entity))

	err = tagger.TagRepository(repository, version, head.Hash())
	checkErr(t, "tagging repository", err)

	reference, err := repository.Reference(plumbing.NewTagReferenceName(version.String()), true)
	checkErr(t, "fetching tag reference", err)

	actualTag, err := repository.TagObject(reference.Hash())
	checkErr(t, "fetching tag from reference", err)

	assert.NotEqual("", actualTag.PGPSignature, "PGP signature should not be empty")
}

func createGitRepository(commitMsg string) (*git.Repository, string, error) {
	dirPath, err := os.MkdirTemp("", "tag-*")
	if err != nil {
		return nil, "", fmt.Errorf("creating temporary directory: %s", err)
	}

	repository, err := git.PlainInit(dirPath, false)
	if err != nil {
		return nil, "", fmt.Errorf("initializing git repository: %s", err)
	}

	_, err = addCommit(repository, commitMsg)
	if err != nil {
		return nil, "", fmt.Errorf("adding commit: %s", err)
	}

	return repository, dirPath, nil
}

func addCommit(repo *git.Repository, message string) (plumbing.Hash, error) {
	w, err := repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("fetching worktree: %s", err)
	}

	fsRoot := w.Filesystem.Root()

	fileName := fmt.Sprintf("file_%d.txt", time.Now().UnixNano())
	filePath := filepath.Join(fsRoot, fileName)

	err = os.WriteFile(filePath, []byte(message), 0644)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	_, err = w.Add(fileName)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("adding file to worktree: %s", err)
	}
	commitHash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  time.Now(),
		},
	})
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("committing file: %s", err)
	}

	return commitHash, nil
}

func checkErr(t *testing.T, msg string, err error) {
	if err != nil {
		t.Fatalf("%s: %s", msg, err)
	}
}

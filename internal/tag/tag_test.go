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
	taggerName  = "Go SemVer Release"
	taggerEmail = "go-semver@release.ci"
)

func TestTag_TagExists(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	head, err := repository.Head()
	if err != nil {
		t.Fatalf("fetching head: %s", err)
	}

	tags := []string{"1.0.0", "1.0.2"}

	for i, tag := range tags {
		_, err = repository.CreateTag(tag, head.Hash(), &git.CreateTagOptions{
			Message: tag,
			Tagger: &object.Signature{
				Name:  "Go SemVer Release",
				Email: "go-semver@release.ci",
				When:  time.Now().Add(time.Duration(i) * time.Hour),
			},
		})
		if err != nil {
			t.Fatalf("creating tag: %s", err)
		}
	}

	tagExists, err := Exists(repository, tags[0])
	if err != nil {
		t.Fatalf("checking if tag exists: %s", err)
	}

	assert.Equal(tagExists, true, "tag should have been found")

	tagDoesNotExists, err := Exists(repository, "0.0.1")
	if err != nil {
		t.Fatalf("checking if tag exists: %s", err)
	}
	assert.Equal(tagDoesNotExists, false, "tag should not have been found")
}

func TestTag_AddTagToRepository(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	version := &semver.Semver{Major: 1}

	tagger := NewTagger(taggerName, taggerEmail)

	err = tagger.TagRepository(repository, version)
	if err != nil {
		t.Fatalf("adding tag: %s", err)
	}

	tagExists, err := Exists(repository, version.String())
	if err != nil {
		t.Fatalf("checking if tag exists: %s", err)
	}

	assert.Equal(tagExists, true, "tag should have been found")
}

func TestTag_AddExistingTagToRepository(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	version := &semver.Semver{Major: 1}

	tagger := NewTagger(taggerName, taggerEmail)

	err = tagger.TagRepository(repository, version)
	if err != nil {
		t.Fatalf("adding tag to repository: %s", err)
	}

	err = tagger.TagRepository(repository, version)
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

	assert.Equal(*gotTag, *wantTag, "tag should match")
}

func TestTag_AddToRepositoryWithNoHead(t *testing.T) {
	assert := assertion.New(t)

	tempDirPath, err := os.MkdirTemp("", "tag-*")
	if err != nil {
		t.Fatalf("creating temporary directory: %v", err)
	}

	defer func() {
		err = os.RemoveAll(tempDirPath)
		if err != nil {
			t.Fatalf("removing temporary directory: %v", err)
		}
	}()

	repository, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		t.Fatalf("initializing git repository: %v", err)
	}

	tagger := NewTagger(taggerName, taggerEmail)

	err = tagger.TagRepository(repository, nil)
	assert.Error(err, "should have failed trying to fetch uninitialized repository head")
}

func TestTag_SignKey(t *testing.T) {
	assert := assertion.New(t)

	config := &packet.Config{
		Algorithm: packet.PubKeyAlgoRSA,
	}

	entity, err := openpgp.NewEntity("John Doe", "", "john.doe@example.com", config)
	if err != nil {
		t.Fatalf("generating openpgp entity: %s", err)
	}

	repository, repositoryPath, err := createGitRepository("fix: ...")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	version := &semver.Semver{Major: 1}

	tagger := NewTagger(taggerName, taggerEmail, WithSignKey(entity))

	err = tagger.TagRepository(repository, version)
	if err != nil {
		t.Fatalf("adding tag to repository: %s", err)
	}

	reference, err := repository.Reference(plumbing.NewTagReferenceName(version.String()), true)
	if err != nil {
		t.Fatalf("fetching tag ref: %s", err)
	}

	actualTag, err := repository.TagObject(reference.Hash())
	if err != nil {
		t.Fatalf("fetching tag object from ref: %s", err)
	}

	assert.NotEqual("", actualTag.PGPSignature, "PGP signature should not be empty")
}

func createGitRepository(commitMsg string) (*git.Repository, string, error) {
	dirPath, err := os.MkdirTemp("", "parser-*")

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
			Name:  "Go SemVer Release",
			Email: "go-semver@release.ci",
			When:  time.Now(),
		},
	})
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("commiting file: %s", err)
	}

	return commitHash, nil
}

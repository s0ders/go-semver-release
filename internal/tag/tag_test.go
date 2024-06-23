package tag

import (
	"os"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v3/internal/gittest"
	"github.com/s0ders/go-semver-release/v3/internal/semver"
)

var (
	taggerName  = "Go Semver Release"
	taggerEmail = "go-semver@release.ci"
)

func TestTag_TagExists(t *testing.T) {
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

	tagExists, err := Exists(testRepository.Repository, tagName)
	checkErr(t, "checking if tag exists", err)
	assert.Equal(tagExists, true, "tag should have been found")

	tagDoesNotExists, err := Exists(testRepository.Repository, "0.0.1")
	checkErr(t, "checking if tag exists", err)
	assert.Equal(tagDoesNotExists, false, "tag should not have been found")
}

func TestTag_AddTagToRepository(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	head, err := testRepository.Head()
	checkErr(t, "fetching head", err)

	version := &semver.Semver{Major: 1}
	prefix := "v"

	tagger := NewTagger(taggerName, taggerEmail, WithTagPrefix(prefix))

	err = tagger.TagRepository(testRepository.Repository, version, head.Hash())
	checkErr(t, "tagging repository", err)

	wantTag := prefix + version.String()

	tagExists, err := Exists(testRepository.Repository, wantTag)
	checkErr(t, "checking if tag exists", err)

	assert.Equal(tagExists, true, "tag should have been found")
}

func TestTag_AddExistingTagToRepository(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	head, err := testRepository.Head()
	checkErr(t, "fetching head", err)

	version := &semver.Semver{Major: 1}

	tagger := NewTagger(taggerName, taggerEmail)

	err = tagger.TagRepository(testRepository.Repository, version, head.Hash())
	checkErr(t, "tagging repository", err)

	err = tagger.TagRepository(testRepository.Repository, version, head.Hash())
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

	assert.Equal(*wantTag, *gotTag)
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

	testRepository, err := gittest.NewRepository()
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = testRepository.Remove()
	})

	head, err := testRepository.Head()
	checkErr(t, "fetching head", err)

	version := &semver.Semver{Major: 1}

	tagger := NewTagger(taggerName, taggerEmail, WithSignKey(entity))

	err = tagger.TagRepository(testRepository.Repository, version, head.Hash())
	checkErr(t, "tagging repository", err)

	reference, err := testRepository.Reference(plumbing.NewTagReferenceName(version.String()), true)
	checkErr(t, "fetching tag reference", err)

	actualTag, err := testRepository.TagObject(reference.Hash())
	checkErr(t, "fetching tag from reference", err)

	assert.NotEqual("", actualTag.PGPSignature, "PGP signature should not be empty")
}

func checkErr(t *testing.T, msg string, err error) {
	if err != nil {
		t.Fatalf("%s: %s", msg, err)
	}
}

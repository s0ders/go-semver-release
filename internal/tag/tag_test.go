package tag

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/s0ders/go-semver-release/v8/internal/gittest"
	"github.com/s0ders/go-semver-release/v8/internal/semver"
)

const (
	taggerName  = "Go Version Release"
	taggerEmail = "go-semver@release.ci"
)

func TestTag_Exists(t *testing.T) {
	repo := gittest.New(t)
	repo.MustTag(t, "1.0.0", repo.MustCommit(t, "fix"))

	exists, err := Exists(repo.Repository, "1.0.0")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = Exists(repo.Repository, "0.0.1")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestTag_Format(t *testing.T) {
	version := &semver.Version{Major: 1, Minor: 2, Patch: 3, PrereleaseLabel: "rc", PrereleaseNumber: 1}

	tests := []struct {
		name    string
		prefix  string
		project string
		want    string
	}{
		{name: "no prefix", want: "1.2.3-rc.1"},
		{name: "prefix", prefix: "v", want: "v1.2.3-rc.1"},
		{name: "project", project: "foo", want: "foo-1.2.3-rc.1"},
		{name: "project and prefix", prefix: "v", project: "foo", want: "foo-v1.2.3-rc.1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tagger := NewTagger(taggerName, taggerEmail, WithTagPrefix(tc.prefix))
			tagger.SetProjectName(tc.project)

			assert.Equal(t, tc.want, tagger.Format(version))
		})
	}
}

func TestTag_TagFromSemver(t *testing.T) {
	hash := plumbing.NewHash("000102030405060708090a0b0c0d0e0f10111213")
	tagger := NewTagger(taggerName, taggerEmail)

	got := tagger.TagFromSemver(&semver.Version{Patch: 1}, hash)

	assert.Equal(t, object.Tag{Hash: hash, Name: "0.0.1", Tagger: tagger.GitSignature}, *got)
}

func TestTag_TagRepository(t *testing.T) {
	version := &semver.Version{Major: 1, Minor: 2, Patch: 3}

	tests := []struct {
		name        string
		lightweight bool
		project     string
		wantTag     string
	}{
		{name: "annotated", wantTag: "v1.2.3"},
		{name: "annotated with project", project: "foo", wantTag: "foo-v1.2.3"},
		{name: "lightweight", lightweight: true, wantTag: "v1.2.3"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := gittest.New(t)
			head := repo.MustCommit(t, "fix")

			tagger := NewTagger(taggerName, taggerEmail, WithTagPrefix("v"), WithLightweight(tc.lightweight))
			tagger.SetProjectName(tc.project)

			require.NoError(t, tagger.TagRepository(repo.Repository, version, head))

			ref, err := repo.Tag(tc.wantTag)
			require.NoError(t, err)

			tagObject, err := repo.TagObject(ref.Hash())
			if tc.lightweight {
				assert.ErrorIs(t, err, plumbing.ErrObjectNotFound, "a lightweight tag has no tag object")
				assert.Equal(t, head, ref.Hash(), "a lightweight tag references the commit directly")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, head, tagObject.Target)
			assert.Equal(t, tc.wantTag, strings.TrimSpace(tagObject.Message))
			assert.Equal(t, taggerName, tagObject.Tagger.Name)
			assert.Equal(t, taggerEmail, tagObject.Tagger.Email)
		})
	}
}

func TestTag_TagRepository_Signed(t *testing.T) {
	entity, err := openpgp.NewEntity("John Doe", "", "john.doe@example.com", &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA})
	require.NoError(t, err)

	repo := gittest.New(t)
	head := repo.MustCommit(t, "fix")

	tagger := NewTagger(taggerName, taggerEmail, WithSignKey(entity))
	require.NoError(t, tagger.TagRepository(repo.Repository, &semver.Version{Major: 1}, head))

	ref, err := repo.Tag("1.0.0")
	require.NoError(t, err)

	tagObject, err := repo.TagObject(ref.Hash())
	require.NoError(t, err)

	var publicKey bytes.Buffer
	armorWriter, err := armor.Encode(&publicKey, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, entity.Serialize(armorWriter))
	require.NoError(t, armorWriter.Close())

	signer, err := tagObject.Verify(publicKey.String())
	require.NoError(t, err, "tag signature should be valid")
	assert.Equal(t, entity.PrimaryKey.Fingerprint, signer.PrimaryKey.Fingerprint)
}

func TestTag_TagRepository_Errors(t *testing.T) {
	t.Run("existing tag", func(t *testing.T) {
		repo := gittest.New(t)
		head := repo.MustCommit(t, "fix")
		repo.MustTag(t, "1.0.0", head)

		err := NewTagger(taggerName, taggerEmail).TagRepository(repo.Repository, &semver.Version{Major: 1}, head)
		assert.ErrorIs(t, err, ErrTagAlreadyExists)
	})

	t.Run("unknown commit", func(t *testing.T) {
		repository, err := git.PlainInit(t.TempDir(), false)
		require.NoError(t, err)

		err = NewTagger(taggerName, taggerEmail).TagRepository(repository, &semver.Version{}, plumbing.ZeroHash)
		assert.ErrorContains(t, err, "creating annotated tag on repository")
	})

	t.Run("nil semver", func(t *testing.T) {
		repo := gittest.New(t)

		err := NewTagger(taggerName, taggerEmail).TagRepository(repo.Repository, nil, repo.MustCommit(t, "fix"))
		assert.ErrorContains(t, err, "semver is nil")
	})
}

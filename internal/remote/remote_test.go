package remote

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/s0ders/go-semver-release/v8/internal/gittest"
)

// mustClone clones the given repository through a new Remote and removes the clone when the test ends.
func mustClone(t *testing.T, url string) (*Remote, *git.Repository) {
	t.Helper()

	r := New("origin", "")

	repository, err := r.Clone(url)
	require.NoError(t, err)

	worktree, err := repository.Worktree()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(worktree.Filesystem.Root()) })

	return r, repository
}

func TestRemote_Clone(t *testing.T) {
	source := gittest.New(t)
	head := source.MustCommit(t, "fix")

	_, repository := mustClone(t, source.Path)

	cloneHead, err := repository.Reference(plumbing.NewRemoteReferenceName("origin", "master"), true)
	require.NoError(t, err)
	assert.Equal(t, head, cloneHead.Hash(), "remote-tracking branch should point to the source HEAD")

	worktree, err := repository.Worktree()
	require.NoError(t, err)

	entries, err := os.ReadDir(worktree.Filesystem.Root())
	require.NoError(t, err)
	for _, entry := range entries {
		assert.Equal(t, ".git", entry.Name(), "the working tree should not be checked out")
	}
}

func TestRemote_Clone_NonExistingRepository(t *testing.T) {
	repository, err := New("origin", "").Clone(filepath.Join(t.TempDir(), "does-not-exist"))

	assert.Nil(t, repository)
	assert.ErrorContains(t, err, "cloning repository")
}

func TestRemote_PushTags(t *testing.T) {
	tests := []struct {
		name string
		tags []string
	}{
		{name: "no tag"},
		{name: "single tag", tags: []string{"v1.0.0"}},
		{name: "several tags", tags: []string{"v1.0.0", "foo-v1.0.0", "bar-v1.0.0"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := gittest.New(t)
			head := source.MustCommit(t, "fix")

			r, repository := mustClone(t, source.Path)

			for _, tagName := range tc.tags {
				_, err := repository.CreateTag(tagName, head, nil)
				require.NoError(t, err)
			}

			require.NoError(t, r.PushTags(tc.tags...))

			for _, tagName := range tc.tags {
				assert.True(t, source.HasTag(t, tagName), "tag %q should have been pushed", tagName)
			}
		})
	}
}

func TestRemote_PushTags_UnavailableRemote(t *testing.T) {
	source, err := gittest.NewRepository()
	require.NoError(t, err)
	head := source.MustCommit(t, "fix")

	r, repository := mustClone(t, source.Path)

	_, err = repository.CreateTag("v1.0.0", head, nil)
	require.NoError(t, err)

	require.NoError(t, source.Remove())

	assert.ErrorContains(t, r.PushTags("v1.0.0"), `pushing tags ["v1.0.0"]`)
}

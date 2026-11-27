package gittest

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	repo := New(t)

	head, err := repo.Head()
	require.NoError(t, err)
	assert.Equal(t, plumbing.NewBranchReferenceName("master"), head.Name())

	commit, err := repo.CommitObject(head.Hash())
	require.NoError(t, err)
	assert.Equal(t, "First commit", commit.Message)
	assert.Empty(t, commit.ParentHashes, "the first commit should be the root commit")

	_, err = commit.File(sampleFile)
	assert.NoError(t, err, "the first commit should contain the sample file")
}

func TestRepository_MustCommit(t *testing.T) {
	repo := New(t)

	first := repo.MustCommit(t, "feat")
	second := repo.MustCommit(t, "fix(api)!")

	commit, err := repo.CommitObject(second)
	require.NoError(t, err)

	assert.Equal(t, "fix(api)!: this a test commit", commit.Message)
	assert.Equal(t, []plumbing.Hash{first}, commit.ParentHashes)

	firstCommit, err := repo.CommitObject(first)
	require.NoError(t, err)
	assert.True(t, commit.Committer.When.After(firstCommit.Committer.When), "commits should have increasing timestamps")
}

func TestRepository_MustCommits(t *testing.T) {
	repo := New(t)

	last := repo.MustCommits(t, "feat", "fix", "chore")

	commit, err := repo.CommitObject(last)
	require.NoError(t, err)
	assert.Equal(t, "chore: this a test commit", commit.Message)

	assert.Equal(t, plumbing.ZeroHash, repo.MustCommits(t), "no commit should return the zero hash")
}

func TestRepository_MustCommitFile(t *testing.T) {
	repo := New(t)

	commit, err := repo.CommitObject(repo.MustCommitFile(t, "feat", "./subdir/file.txt"))
	require.NoError(t, err)

	_, err = commit.File("subdir/file.txt")
	assert.NoError(t, err, "the commit should contain the file at its normalized path")
}

func TestRepository_Tags(t *testing.T) {
	repo := New(t)
	head := repo.MustCommit(t, "feat")

	repo.MustTag(t, "v1.0.0", head)
	repo.MustLightweightTag(t, "v2.0.0", head)

	annotated, err := repo.Tag("v1.0.0")
	require.NoError(t, err)
	tagObject, err := repo.TagObject(annotated.Hash())
	require.NoError(t, err)
	assert.Equal(t, head, tagObject.Target)

	lightweight, err := repo.Tag("v2.0.0")
	require.NoError(t, err)
	assert.Equal(t, head, lightweight.Hash())

	assert.True(t, repo.HasTag(t, "v1.0.0"))
	assert.False(t, repo.HasTag(t, "v3.0.0"))
}

func TestRepository_Branches(t *testing.T) {
	repo := New(t)
	base := repo.MustCommit(t, "feat")

	repo.MustCheckoutBranch(t, "feature")
	feature := repo.MustCommit(t, "fix")

	repo.MustCheckout(t, "master")
	master := repo.MustCommit(t, "fix")

	assert.Equal(t, master, repo.MustBranchRef(t, "master").Hash())
	assert.Equal(t, feature, repo.MustBranchRef(t, "feature").Hash())

	commit, err := repo.CommitObject(master)
	require.NoError(t, err)
	assert.Equal(t, []plumbing.Hash{base}, commit.ParentHashes, "master should not contain the feature commit")
}

func TestRepository_MustClone(t *testing.T) {
	repo := New(t)
	head := repo.MustCommit(t, "feat")

	clone := repo.MustClone(t)

	remoteRef, err := clone.Reference(plumbing.NewRemoteReferenceName("origin", "master"), true)
	require.NoError(t, err)
	assert.Equal(t, head, remoteRef.Hash())
}

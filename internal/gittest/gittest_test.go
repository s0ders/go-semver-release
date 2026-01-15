package gittest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepository(t *testing.T) {
	repo, err := NewRepository()
	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.NotEmpty(t, repo.Path)

	t.Cleanup(func() {
		_ = repo.Remove()
	})

	// Check repository exists
	_, err = os.Stat(repo.Path)
	assert.NoError(t, err)
}

func TestRepository_AddCommit(t *testing.T) {
	repo, err := NewRepository()
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Remove()
	})

	hash, err := repo.AddCommit("feat")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestRepository_AddTag(t *testing.T) {
	repo, err := NewRepository()
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Remove()
	})

	head, err := repo.Head()
	assert.NoError(t, err)

	err = repo.AddTag("v1.0.0", head.Hash())
	assert.NoError(t, err)
}

func TestRepository_AddLightweightTag(t *testing.T) {
	repo, err := NewRepository()
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Remove()
	})

	head, err := repo.Head()
	assert.NoError(t, err)

	err = repo.AddLightweightTag("v2.0.0", head.Hash())
	assert.NoError(t, err)
}

func TestRepository_CheckoutBranch(t *testing.T) {
	repo, err := NewRepository()
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Remove()
	})

	err = repo.CheckoutBranch("feature")
	assert.NoError(t, err)
}

func TestRepository_AddCommitWithSpecificFile(t *testing.T) {
	repo, err := NewRepository()
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Remove()
	})

	hash, err := repo.AddCommitWithSpecificFile("feat", "subdir/file.txt")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestRepository_When(t *testing.T) {
	repo, err := NewRepository()
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = repo.Remove()
	})

	// First call
	when1 := repo.When()
	// Second call should be later
	when2 := repo.When()

	assert.True(t, when2.After(when1))
}

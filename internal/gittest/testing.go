package gittest

import (
	"errors"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// New creates a new TestRepository that is removed when the test ends. The test fails immediately if the repository
// cannot be created.
func New(tb testing.TB) *TestRepository {
	tb.Helper()

	r, err := NewRepository()
	if err != nil {
		tb.Fatalf("creating test repository: %s", err)
	}

	tb.Cleanup(func() { _ = r.Remove() })

	return r
}

// MustClone clones the repository, see Clone. The clone is removed when the test ends.
func (r *TestRepository) MustClone(tb testing.TB) *TestRepository {
	tb.Helper()

	clone, err := r.Clone()
	if err != nil {
		tb.Fatalf("cloning test repository: %s", err)
	}

	tb.Cleanup(func() { _ = clone.Remove() })

	return clone
}

// MustCommit adds a commit with the given message prefix, see AddCommit.
func (r *TestRepository) MustCommit(tb testing.TB, commitType string) plumbing.Hash {
	tb.Helper()

	hash, err := r.AddCommit(commitType)
	if err != nil {
		tb.Fatalf("adding commit %q: %s", commitType, err)
	}

	return hash
}

// MustCommits adds one commit per given message prefix and returns the hash of the last one.
func (r *TestRepository) MustCommits(tb testing.TB, commitTypes ...string) plumbing.Hash {
	tb.Helper()

	var last plumbing.Hash
	for _, commitType := range commitTypes {
		last = r.MustCommit(tb, commitType)
	}

	return last
}

// MustCommitFile adds a commit writing the given file, see AddCommitWithSpecificFile.
func (r *TestRepository) MustCommitFile(tb testing.TB, commitType, filePath string) plumbing.Hash {
	tb.Helper()

	hash, err := r.AddCommitWithSpecificFile(commitType, filePath)
	if err != nil {
		tb.Fatalf("adding commit %q on %q: %s", commitType, filePath, err)
	}

	return hash
}

// MustTag adds an annotated tag, see AddTag.
func (r *TestRepository) MustTag(tb testing.TB, tagName string, hash plumbing.Hash) {
	tb.Helper()

	if err := r.AddTag(tagName, hash); err != nil {
		tb.Fatalf("adding tag %q: %s", tagName, err)
	}
}

// MustLightweightTag adds a lightweight tag, see AddLightweightTag.
func (r *TestRepository) MustLightweightTag(tb testing.TB, tagName string, hash plumbing.Hash) {
	tb.Helper()

	if err := r.AddLightweightTag(tagName, hash); err != nil {
		tb.Fatalf("adding lightweight tag %q: %s", tagName, err)
	}
}

// MustCheckoutBranch creates a branch from HEAD and checks it out, see CheckoutBranch.
func (r *TestRepository) MustCheckoutBranch(tb testing.TB, name string) {
	tb.Helper()

	if err := r.CheckoutBranch(name); err != nil {
		tb.Fatalf("checking out branch %q: %s", name, err)
	}
}

// MustCheckout checks out an existing local branch.
func (r *TestRepository) MustCheckout(tb testing.TB, name string) {
	tb.Helper()

	if err := r.Checkout(name); err != nil {
		tb.Fatalf("checking out branch %q: %s", name, err)
	}
}

// MustBranchRef returns the reference of the given local branch.
func (r *TestRepository) MustBranchRef(tb testing.TB, name string) *plumbing.Reference {
	tb.Helper()

	ref, err := r.Reference(plumbing.NewBranchReferenceName(name), true)
	if err != nil {
		tb.Fatalf("getting branch %q reference: %s", name, err)
	}

	return ref
}

// HasTag reports whether the repository has a tag with the given name.
func (r *TestRepository) HasTag(tb testing.TB, tagName string) bool {
	tb.Helper()

	_, err := r.Tag(tagName)
	if err == nil {
		return true
	}

	if !errors.Is(err, git.ErrTagNotFound) {
		tb.Fatalf("looking up tag %q: %s", tagName, err)
	}

	return false
}

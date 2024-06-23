// Package gittest provides basic types and functions for testing operations related to Git repositories.
package gittest

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const sampleFile = "sample.txt"

var referenceTime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

type TestRepository struct {
	*git.Repository
	Path    string
	Counter int
}

// NewRepository creates a new TestRepository.
func NewRepository() (testRepository *TestRepository, err error) {
	testRepository = &TestRepository{}

	path, err := os.MkdirTemp("", "gittest-*")
	if err != nil {
		return testRepository, fmt.Errorf("creating temporary directory: %w", err)
	}

	testRepository.Path = path

	repository, err := git.PlainInit(path, false)
	if err != nil {
		return testRepository, fmt.Errorf("initializing repository: %s", err)
	}

	testRepository.Repository = repository

	tempFilePath := filepath.Join(path, sampleFile)

	commitFile, err := os.Create(tempFilePath)
	if err != nil {
		return testRepository, fmt.Errorf("creating first commit file: %s", err)
	}

	defer func() {
		err = commitFile.Close()
	}()

	_, err = commitFile.WriteString("...")
	if err != nil {
		return testRepository, err
	}

	worktree, err := repository.Worktree()
	if err != nil {
		return testRepository, fmt.Errorf("fetching worktree: %w", err)
	}

	_, err = worktree.Add(sampleFile)
	if err != nil {
		return testRepository, fmt.Errorf("adding commit file to worktree: %w", err)
	}

	_, err = worktree.Commit("First commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  referenceTime,
		},
	})
	if err != nil {
		return testRepository, fmt.Errorf("creating commit: %w", err)
	}

	return testRepository, err
}

// AddCommit adds a new commit with a given conventional commit type to the underlying Git repository.
func (r *TestRepository) AddCommit(commitType string) (plumbing.Hash, error) {
	var commitHash plumbing.Hash

	worktree, err := r.Worktree()
	if err != nil {
		return commitHash, fmt.Errorf("fetching worktree: %w", err)
	}

	commitFilePath := filepath.Join(r.Path, sampleFile)

	err = os.WriteFile(commitFilePath, []byte(strconv.Itoa(rand.IntN(10000))), 0o644)
	if err != nil {
		return commitHash, fmt.Errorf("writing commit file: %w", err)
	}

	_, err = worktree.Add(sampleFile)
	if err != nil {
		return commitHash, fmt.Errorf("adding commit file to worktree: %w", err)
	}

	commitMessage := fmt.Sprintf("%s: this a test commit", commitType)

	commitOpts := &git.CommitOptions{
		Committer: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  r.When(),
		},
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  r.When(),
		},
	}

	commitHash, err = worktree.Commit(commitMessage, commitOpts)
	if err != nil {
		return commitHash, fmt.Errorf("creating commit: %w", err)
	}

	return commitHash, nil
}

// AddTag adds a new tag to the underlying Git repository with a given name and pointing to a given hash.
func (r *TestRepository) AddTag(tagName string, hash plumbing.Hash) error {
	tagOpts := &git.CreateTagOptions{
		Message: tagName,
		Tagger: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  r.When(),
		},
	}

	_, err := r.CreateTag(tagName, hash, tagOpts)

	return err
}

// Remove removes the underlying Git repository.
func (r *TestRepository) Remove() error {
	return os.RemoveAll(r.Path)
}

// CheckoutBranch creates a new branch with the given name and checkout to it.
func (r *TestRepository) CheckoutBranch(name string) error {
	head, err := r.Head()
	if err != nil {
		return err
	}

	refName := "refs/heads/" + name
	ref := plumbing.NewHashReference(plumbing.ReferenceName(refName), head.Hash())

	err = r.Storer.SetReference(ref)
	if err != nil {
		return err
	}

	return nil
}

// When returns a time.Time starting at 2000/01/01 00:00:00 and increasing of 10 second every new call.
func (r *TestRepository) When() time.Time {
	r.Counter++
	return referenceTime.Add(time.Duration(r.Counter*10) * time.Second)
}

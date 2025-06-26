// Package gittest provides basic types and functions for testing operations related to Git repositories.
package gittest

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

const sampleFile = "sample.txt"

var referenceTime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

type TestRepository struct {
	*git.Repository
	RemoteServer *http.Server
	RemoteURL    string
	Path         string
	Counter      uint
}

// NewRepository creates a new TestRepository.
func NewRepository() (*TestRepository, error) {
	testRepository := &TestRepository{}

	path, err := os.MkdirTemp("", "gittest-*")
	if err != nil {
		return testRepository, fmt.Errorf("creating temporary directory: %w", err)
	}

	testRepository.Path = path

	repository, err := git.PlainInit(path, false)
	if err != nil {
		return testRepository, fmt.Errorf("initializing repository: %s", err)
	}

	err = repository.Storer.SetReference(
		plumbing.NewSymbolicReference(
			plumbing.HEAD,
			plumbing.NewBranchReferenceName("main"),
		),
	)
	if err != nil {
		return testRepository, fmt.Errorf("create main branch: %s", err)
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

// Clone clones the current TestRepository to memory and returns the clone of that repository. This
// method is useful when testing on repository that are expected to have a configured remote.
func (r *TestRepository) Clone() (*TestRepository, error) {
	testRepository := &TestRepository{}

	var err error
	testRepository.Path = "."
	testRepository.Repository, err = git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:      r.Path,
		Progress: io.Discard,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning repository: %w", err)
	}

	return testRepository, nil
}

// AddCommitToBranch adds a new commit to specified branch
func (r *TestRepository) AddCommitToBranch(commitType string, branch string) (*object.Commit, error) {
	err := r.CheckoutBranch(branch, false)
	if err != nil {
		return nil, fmt.Errorf("checkout branch: %w", err)
	}
	return r.AddCommit(commitType)
}

// AddCommit adds a new commit with a given conventional commit type to the underlying Git repository.
func (r *TestRepository) AddCommit(commitType string) (*object.Commit, error) {
	worktree, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("fetching worktree: %w", err)
	}

	newFile, err := worktree.Filesystem.Create(sampleFile)
	if err != nil {
		return nil, fmt.Errorf("creating commit file: %w", err)
	}

	_, err = newFile.Write([]byte(strconv.Itoa(rand.IntN(10000))))
	if err != nil {
		return nil, fmt.Errorf("writing commit file: %w", err)
	}

	_, err = worktree.Add(newFile.Name())
	if err != nil {
		return nil, fmt.Errorf("adding commit file to worktree: %w", err)
	}

	commitMessage := fmt.Sprintf("%s: this a test commit", commitType)

	when := r.When()

	commitOpts := &git.CommitOptions{
		Committer: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  when,
		},
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  when,
		},
	}

	commitHash, err := worktree.Commit(commitMessage, commitOpts)
	if err != nil {
		return nil, fmt.Errorf("creating commit: %w", err)
	}

	c, err := r.CommitObject(commitHash)
	if err != nil {
		return nil, fmt.Errorf("fetching commit: %w", err)
	}

	return c, nil
}

func (r *TestRepository) AddCommitWithSpecificFile(commitType, filePath string) (*object.Commit, error) {
	worktree, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("fetching worktree: %w", err)
	}

	commitFilePath := filepath.Clean(filePath)
	dirs := filepath.Dir(commitFilePath)

	err = worktree.Filesystem.MkdirAll(dirs, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("creating parent directory: %w", err)
	}

	newFile, err := worktree.Filesystem.Create(commitFilePath)
	if err != nil {
		return nil, fmt.Errorf("creating commit file: %w", err)
	}

	_, err = newFile.Write([]byte(strconv.Itoa(rand.IntN(10000))))
	if err != nil {
		return nil, fmt.Errorf("writing commit file: %w", err)
	}

	_, err = worktree.Add(filePath)
	if err != nil {
		return nil, fmt.Errorf("adding commit file to worktree: %w", err)
	}

	commitMessage := fmt.Sprintf("%s: this a test commit", commitType)

	when := r.When()

	commitOpts := &git.CommitOptions{
		Committer: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  when,
		},
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  when,
		},
	}

	commitHash, err := worktree.Commit(commitMessage, commitOpts)
	if err != nil {
		return nil, fmt.Errorf("creating commit: %w", err)
	}

	c, err := r.CommitObject(commitHash)
	if err != nil {
		return nil, fmt.Errorf("fetching commit: %w", err)
	}

	return c, nil
}

// Adds tag to a specific branch
func (r *TestRepository) AddTagToBranch(tagName string, branch string) error {
	ref, err := r.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		return fmt.Errorf("fetching branch: %w", err)
	}
	return r.AddTag(tagName, ref.Hash())
}

// AddTag adds a new tag to the underlying Git repository with a given name and pointing to a given hash.
func (r *TestRepository) AddTag(tagName string, hash plumbing.Hash) error {
	c, err := r.CommitObject(hash)
	if err != nil {
		return fmt.Errorf("getting commit: %w", err)
	}

	tagOpts := &git.CreateTagOptions{
		Message: tagName,
		Tagger: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver@release.ci",
			When:  c.Committer.When,
		},
	}

	_, err = r.CreateTag(tagName, hash, tagOpts)

	return err
}

// Remove removes the underlying Git repository.
func (r *TestRepository) Remove() error {
	return os.RemoveAll(r.Path)
}

// CheckoutBranch checkouts a branch or creates a new branch with the given name and checkout to it.
func (r *TestRepository) CheckoutBranch(name string, create bool) error {
	branch := plumbing.NewBranchReferenceName(name)

	if create {
		head, err := r.Head()
		if err != nil {
			return err
		}

		ref := plumbing.NewHashReference(branch, head.Hash())
		err = r.Storer.SetReference(ref)
		if err != nil {
			return err
		}
	}

	worktree, err := r.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branch,
		Force:  true,
	})
	if err != nil {
		return err
	}

	return nil
}

// CheckoutOrCreateBranch checkouts branch or creates a new one if the branch does not exist.
func (r *TestRepository) CheckoutOrCreateBranch(name string) error {
	_, err := r.Reference(plumbing.NewBranchReferenceName(name), true)
	if err == nil {
		return r.CheckoutBranch(name, false)
	} else {
		return r.CheckoutBranch(name, true)
	}
}

// When returns a time.Time starting at 2000/01/01 00:00:00 and increasing of 10 second every new call.
func (r *TestRepository) When() time.Time {
	r.Counter++
	return referenceTime.Add(time.Duration(r.Counter*10) * time.Second)
}

func (r *TestRepository) LatestCommit() (*object.Commit, error) {
	head, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("fetching head: %w", err)
	}
	c, err := r.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("fetching commit: %w", err)
	}
	return c, nil
}

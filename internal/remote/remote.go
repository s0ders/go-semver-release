// Package remote provides basic functions to work with Git remotes.
package remote

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

type Remote struct {
	auth       *http.BasicAuth
	repository *git.Repository
	storage    *filesystem.Storage
	name       string
	// dir is the temporary directory of the clone, removed by Close. It is empty for a repository opened in place.
	dir string
	// local reports whether the repository was opened in place, in which case tags created in it need no push.
	local bool
}

func New(name string, token string) *Remote {
	return &Remote{
		name: name,
		auth: &http.BasicAuth{
			Username: "go-semver-release",
			Password: token,
		},
	}
}

// Open returns the repository to version. A local repository is opened in place, which is equivalent to cloning it
// and pushing the created tags back to it, without copying all of its objects. Any other repository is cloned, see
// Clone.
func (r *Remote) Open(url string) (*git.Repository, error) {
	if info, err := os.Stat(url); err == nil && info.IsDir() {
		return r.openLocal(url)
	}

	return r.Clone(url)
}

// openLocal opens the local repository found at the given path, keeping its packfiles open for the duration of the
// run. Repositories whose Git directory is not a plain ".git" directory or a bare repository (e.g., a worktree or a
// submodule, whose ".git" is a file) are opened with go-git's defaults.
func (r *Remote) openLocal(path string) (*git.Repository, error) {
	var dotGit, worktree string

	switch {
	case isDir(filepath.Join(path, git.GitDirName)):
		dotGit, worktree = filepath.Join(path, git.GitDirName), path
	case isDir(filepath.Join(path, "objects")) && isFile(filepath.Join(path, "HEAD")):
		dotGit = path
	default:
		repository, err := git.PlainOpen(path)
		if err != nil {
			return nil, fmt.Errorf("opening repository: %w", err)
		}

		r.repository, r.local = repository, true

		return repository, nil
	}

	r.storage = filesystem.NewStorageWithOptions(osfs.New(dotGit), cache.NewObjectLRUDefault(), filesystem.Options{KeepDescriptors: true})

	var worktreeFS billy.Filesystem
	if worktree != "" {
		worktreeFS = osfs.New(worktree)
	}

	repository, err := git.Open(r.storage, worktreeFS)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("opening repository: %w", err), r.Close())
	}

	r.repository, r.local = repository, true

	return repository, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Clone clones a given remote repository to a temporary directory, which is removed by Close.
//
// The working tree is not checked out: versions are computed from the commit graph and tags are created directly in
// the object database, so writing every file of the repository to disk would be wasted work.
func (r *Remote) Clone(url string) (*git.Repository, error) {
	tempDir, err := os.MkdirTemp("", "go-semver-release-*")
	if err != nil {
		return nil, fmt.Errorf("creating temporary directory: %w", err)
	}

	r.dir = tempDir

	// The clone is private to this process: packfiles are kept open instead of being reopened for every object read,
	// and their list is cached. This divides the cost of walking the commit history.
	r.storage = filesystem.NewStorageWithOptions(
		osfs.New(filepath.Join(tempDir, git.GitDirName)),
		cache.NewObjectLRUDefault(),
		filesystem.Options{KeepDescriptors: true, ExclusiveAccess: true},
	)

	r.repository, err = git.Clone(r.storage, osfs.New(tempDir), &git.CloneOptions{
		RemoteName: r.name,
		Auth:       r.auth,
		URL:        url,
		Progress:   io.Discard,
		NoCheckout: true,
	})
	if err != nil {
		return nil, errors.Join(fmt.Errorf("cloning repository: %w", err), r.Close())
	}

	return r.repository, nil
}

// Close releases the repository's open files and, for a clone, removes it. The cloned repository, which may hold
// private sources, would otherwise be left on disk, e.g., on shared or self-hosted CI runners. A repository opened in
// place is never removed.
func (r *Remote) Close() error {
	var errs []error

	if r.storage != nil {
		if err := r.storage.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing repository: %w", err))
		}
	}

	if r.dir != "" {
		if err := os.RemoveAll(r.dir); err != nil {
			errs = append(errs, fmt.Errorf("removing local clone: %w", err))
		}
	}

	r.dir, r.storage, r.repository, r.local = "", nil, nil, false

	return errors.Join(errs...)
}

// PushTags pushes the given tags to the previously cloned repository's remote in a single push, avoiding one
// network round trip (and reference negotiation) per tag. Tags of a repository opened in place were created in the
// repository itself and need no push.
func (r *Remote) PushTags(tagNames ...string) error {
	if len(tagNames) == 0 || r.local {
		return nil
	}

	refSpecs := make([]config.RefSpec, 0, len(tagNames))
	for _, tagName := range tagNames {
		refSpecs = append(refSpecs, config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", tagName, tagName)))
	}

	po := &git.PushOptions{
		RemoteName: r.name,
		RefSpecs:   refSpecs,
		Auth:       r.auth,
		Progress:   io.Discard,
	}

	err := r.repository.Push(po)
	if err != nil {
		return fmt.Errorf("pushing tags %q: %w", tagNames, err)
	}

	return nil
}

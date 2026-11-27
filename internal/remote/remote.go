// Package remote provides basic functions to work with Git remotes.
package remote

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Remote struct {
	auth       *http.BasicAuth
	repository *git.Repository
	name       string
	dir        string
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

	r.repository, err = git.PlainClone(tempDir, false, &git.CloneOptions{
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

// Close removes the local clone, if any. The cloned repository, which may hold private sources, would otherwise be
// left on disk, e.g., on shared or self-hosted CI runners.
func (r *Remote) Close() error {
	if r.dir == "" {
		return nil
	}

	if err := os.RemoveAll(r.dir); err != nil {
		return fmt.Errorf("removing local clone: %w", err)
	}

	r.dir = ""
	r.repository = nil

	return nil
}

// PushTags pushes the given tags to the previously cloned repository's remote in a single push, avoiding one
// network round trip (and reference negotiation) per tag.
func (r *Remote) PushTags(tagNames ...string) error {
	if len(tagNames) == 0 {
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

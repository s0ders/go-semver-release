// Package remote provides basic functions to work with Git remotes.
package remote

import (
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

// Clone clones a given remote repository to a temporary directory.
//
// The working tree is not checked out: versions are computed from the commit graph and tags are created directly in
// the object database, so writing every file of the repository to disk would be wasted work.
func (r *Remote) Clone(url string) (*git.Repository, error) {
	tempDir, err := os.MkdirTemp("", "*")
	if err != nil {
		return nil, fmt.Errorf("creating temporary directory: %w", err)
	}

	r.repository, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		RemoteName: r.name,
		Auth:       r.auth,
		URL:        url,
		Progress:   io.Discard,
		NoCheckout: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning repository: %w", err)
	}

	return r.repository, nil
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

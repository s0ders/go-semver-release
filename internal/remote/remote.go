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
	name       string
	auth       *http.BasicAuth
	repository *git.Repository
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
	})
	if err != nil {
		return nil, fmt.Errorf("cloning repository: %w", err)
	}

	return r.repository, nil
}

// PushTag pushes a given tag to the previously cloned repository's remote.
func (r *Remote) PushTag(tagName string) error {
	po := &git.PushOptions{
		RemoteName: r.name,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", tagName, tagName))},
		Auth:       r.auth,
		Progress:   io.Discard,
	}

	err := r.repository.Push(po)
	if err != nil {
		return fmt.Errorf("pushing tag %q: %w", tagName, err)
	}

	return nil
}

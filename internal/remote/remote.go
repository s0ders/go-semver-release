// Package remote provides basic functions to work with Git remotes.
package remote

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"os"
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

	repository, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		RemoteName: r.name,
		Auth:       r.auth,
		URL:        url,
		Progress:   os.Stdout,
	})

	r.repository = repository

	return repository, nil
}

// PushTag pushes a given tag to the previously cloned repository's remote.
func (r *Remote) PushTag(tagName string) error {
	po := &git.PushOptions{
		RemoteName: r.name,
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", tagName, tagName))},
		Auth:       r.auth,
	}

	if err := r.repository.Push(po); err != nil {
		return fmt.Errorf("pushing tag %q: %w", tagName, err)
	}

	return nil
}

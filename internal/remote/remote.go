package remote

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"os"
)

type Remote struct {
	auth       *http.BasicAuth
	repository *git.Repository
}

func New(token string) *Remote {
	return &Remote{
		auth: &http.BasicAuth{
			Username: "go-semver-release",
			Password: token,
		},
	}
}

func (r *Remote) Clone(url string) (*git.Repository, error) {
	tempDir, err := os.MkdirTemp("", "*")
	if err != nil {
		return nil, fmt.Errorf("creating temporary directory: %w", err)
	}

	repository, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		Auth:     r.auth,
		URL:      url,
		Progress: os.Stdout,
	})

	r.repository = repository

	return repository, nil
}

func (r *Remote) PushTag(tagName string) error {
	po := &git.PushOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
		Auth:       r.auth,
	}

	if err := r.repository.Push(po); err != nil {
		return fmt.Errorf("pushing tag %q: %w", tagName, err)
	}

	return nil
}

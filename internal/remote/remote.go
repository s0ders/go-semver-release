package remote

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/s0ders/go-semver-release/v2/internal/semver"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Remote struct {
	logger     *slog.Logger
	auth       *http.BasicAuth
	remoteName string
}

type Repository interface {
	Push(o *git.PushOptions) error
}

func New(logger *slog.Logger, token string, remoteName string) Remote {
	return Remote{
		logger: logger,
		auth: &http.BasicAuth{
			Username: "go-semver-release",
			Password: token,
		},
		remoteName: remoteName,
	}
}

func (r Remote) Clone(url, branch string) (*git.Repository, string, error) {
	path, err := os.MkdirTemp("", "go-semver-release-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	cloneOption := &git.CloneOptions{
		Auth:     r.auth,
		URL:      url,
		Progress: os.Stdout,
	}

	if branch != "" {
		cloneOption.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))
	}

	repository, err := git.PlainClone(path, false, cloneOption)
	if err != nil {
		return nil, "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return repository, path, nil
}

func (r Remote) PushTagToRemote(repository Repository, semver *semver.Semver) error {
	po := &git.PushOptions{
		Auth:       r.auth,
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
		RemoteName: r.remoteName,
	}

	if err := repository.Push(po); err != nil {
		return fmt.Errorf("failed to push tag to remote: %w", err)
	}

	r.logger.Debug("pushed tag on repository", "tag", semver)

	return nil
}

package cloner

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Cloner struct {
	l *slog.Logger
}

func New(logger *slog.Logger) Cloner {
	return Cloner{
		l: logger,
	}
}

func (c Cloner) Clone(url, branch, token string) (*git.Repository, string, error) {
	auth := &http.BasicAuth{
		Username: "go-semver-release",
		Password: token,
	}

	path, err := os.MkdirTemp("", "go-semver-release-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	cloneOption := &git.CloneOptions{
		Auth:     auth,
		URL:      url,
		Progress: nil,
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

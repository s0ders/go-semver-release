package cloner

import (
	"fmt"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Cloner struct {
	l *log.Logger
}

func NewCloner() Cloner {
	logger := log.New(os.Stdout, "cloner", log.Default().Flags())

	return Cloner{
		l: logger,
	}
}

func (c Cloner) Clone(url, branch, token string) (*git.Repository, string) {
	auth := &http.BasicAuth{
		Username: "go-semver-release",
		Password: token,
	}

	path, err := os.MkdirTemp("", "go-semver-release-*")
	if err != nil {
		c.l.Fatalf("failed to create temporary directory to clone repository in: %s", err)
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
		c.l.Fatalf("failed to clone repository: %s", err)
	}

	return repository, path
}

package tagger

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/s0ders/go-semver-release/v2/internal/semver"
)

var GitSignature = object.Signature{
	Name:  "Go Semver Release",
	Email: "ci@ci.ci",
	When:  time.Now(),
}

type Tagger struct {
	logger    *slog.Logger
	tagPrefix string
	verbose   bool
}

func New(logger *slog.Logger, prefix string, verbose bool) *Tagger {
	return &Tagger{
		logger:    logger,
		tagPrefix: prefix,
		verbose:   verbose,
	}
}

func NewTagFromSemver(semver semver.Semver, hash plumbing.Hash) *object.Tag {
	tag := &object.Tag{
		Hash:   hash,
		Name:   semver.String(),
		Tagger: GitSignature,
	}

	return tag
}

func (t *Tagger) TagExists(r *git.Repository, tagName string) (bool, error) {
	tagExists := false
	tags, err := r.Tags()
	if err != nil {
		return false, fmt.Errorf("failed to fetch tags: %w", err)
	}

	err = tags.ForEach(func(tag *plumbing.Reference) error {
		tagRef := fmt.Sprintf("refs/tags/%s", tagName)
		if tag.Name().String() == tagRef {
			tagExists = true
			return nil
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	return tagExists, nil
}

// AddTagToRepository create a new annotated tag on the repository
// with a name corresponding to the semver passed as a parameter.
func (t *Tagger) AddTagToRepository(r *git.Repository, semver *semver.Semver) (*git.Repository, error) {
	h, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch head: %w", err)
	}

	tag := fmt.Sprintf("%s%s", t.tagPrefix, semver.NormalVersion())

	tagExists, err := t.TagExists(r, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to check if tag exists: %w", err)
	}

	if tagExists {
		return nil, fmt.Errorf("tag already exists")
	}

	_, err = r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Message: semver.NormalVersion(),
		Tagger:  &GitSignature,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tag on repository: %w", err)
	}

	if t.verbose {
		t.logger.Info("created new tag on repository", "tag", semver.String())
	}

	return r, nil
}

func (t *Tagger) PushTagToRemote(r *git.Repository, token string, semver *semver.Semver) error {
	repo, err := t.AddTagToRepository(r, semver)
	if err != nil {
		return fmt.Errorf("failed to add tag on repository: %w", err)
	}

	po := &git.PushOptions{
		Auth: &http.BasicAuth{
			Username: "go-semver-release",
			Password: token,
		},
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
		RemoteName: "origin",
	}

	if err := repo.Push(po); err != nil {
		return fmt.Errorf("failed to push tag to remote: %w", err)
	}

	if t.verbose {
		t.logger.Info("pushed tag on repository", "tag", semver)
	}

	return nil
}

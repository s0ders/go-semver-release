package tagger

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/s0ders/go-semver-release/semver"
)

var TaggerGitSignature = object.Signature{
	Name:  "Go Semver Release",
	Email: "ci@ci.ci",
	When:  time.Now(),
}

type Tagger struct {
	logger    *log.Logger
}

func NewTagger(logger *log.Logger) *Tagger {
	return &Tagger{
		logger:    logger,
	}
}

func NewTag(semver semver.Semver, hash plumbing.Hash) *object.Tag {

	tag := &object.Tag{
		Hash:   hash,
		Name:   semver.String(),
		Tagger: TaggerGitSignature,
	}

	return tag
}

// AddTagToRepository create a new annotated tag on the repository
// with a name corresponding to the semver passed as a parameter.
func (t *Tagger) AddTagToRepository(r *git.Repository, semver *semver.Semver) (*git.Repository, error) {
	h, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("AddTagToRepository: failed to fetch head: %w", err)
	}

	tag := semver.String()

	_, err = r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Message: semver.String(),
		Tagger:  &TaggerGitSignature,
	})

	if err != nil {
		return nil, fmt.Errorf("AddTagToRepository: failed to create tag on repository: %w", err)
	}

	t.logger.Printf("created new tag %s on repository", semver.String())

	return r, nil
}

func (t *Tagger) PushTagToRemote(r *git.Repository, auth transport.AuthMethod) error {
	po := &git.PushOptions{
		Auth:       auth,
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
		RemoteName: "origin",
	}

	if err := r.Push(po); err != nil {
		return fmt.Errorf("PushTagToRemote: failed to push: %w", err)
	}

	return nil
}

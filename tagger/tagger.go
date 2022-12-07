package tagger

import (
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

type Tagger struct {
	logger *log.Logger
}

func NewTagger(l *log.Logger) *Tagger {
	return &Tagger{l}
}

func NewTag(semver semver.Semver, hash plumbing.Hash) (*object.Tag, error) {

	tag := &object.Tag{
		Hash: hash,
		Name: semver.String(),
		Tagger: object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	}

	return tag, nil
}

// AddTagToRepository create a new annotated tag on the repository
// with a name corresponding to the semver passed as a parameter.
func (t *Tagger) AddTagToRepository(r *git.Repository, semver *semver.Semver) (*git.Repository, error) {
	h, err := r.Head()
	if err != nil {
		return nil, err
	}

	_, err = r.CreateTag(semver.String(), h.Hash(), &git.CreateTagOptions{
		Message: semver.String(),
		Tagger: &object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	})

	if err != nil {
		return nil, err
	}

	t.logger.Printf("Created new tag %s on repository", semver.String())

	return r, nil
}

func (t *Tagger) PushTagToRemote(r *git.Repository, auth transport.AuthMethod) error {
	po := &git.PushOptions{
		Auth:       auth,
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
		RemoteName: "origin",
	}

	return r.Push(po)
}

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
	
	"github.com/s0ders/go-semver-release/internal/semver"
)

var TaggerGitSignature = object.Signature{
	Name:  "Go Semver Release",
	Email: "ci@ci.ci",
	When:  time.Now(),
}

type Tagger struct {
	logger    *log.Logger
	tagPrefix string
}

func NewTagger(logger *log.Logger, tagPrefix string) *Tagger {
	return &Tagger{
		logger:    logger,
		tagPrefix: tagPrefix,
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

func (t *Tagger) TagExists(r *git.Repository, tagName string) (bool, error) {
	tagExists := false
	tags, err := r.Tags()

	if err != nil {
		return false, fmt.Errorf("TagExists: failed to fetch tags: %w", err)
	}

	tags.ForEach(func(tag *plumbing.Reference) error {
		tagRef := fmt.Sprintf("refs/tags/%s", tagName)
		if tag.Name().String() == tagRef {
			tagExists = true
			return nil
		}
		return nil
	})

	return tagExists, nil
}

// AddTagToRepository create a new annotated tag on the repository
// with a name corresponding to the semver passed as a parameter.
func (t *Tagger) AddTagToRepository(r *git.Repository, semver *semver.Semver) (*git.Repository, error) {
	h, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("AddTagToRepository: failed to fetch head: %w", err)
	}

	tag := fmt.Sprintf("%s%s", t.tagPrefix, semver.NormalVersion())

	tagExists, err := t.TagExists(r, tag)
	if err != nil {
		return nil, fmt.Errorf("AddTagToRepository: failed to check if tag exists: %w", err)
	}

	if tagExists {
		return nil, fmt.Errorf("AddTagToRepository: tag already exists")
	}

	_, err = r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Message: semver.NormalVersion(),
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

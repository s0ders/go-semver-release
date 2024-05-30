// Package tag provides function to work with Git tags.
package tag

import (
	"errors"
	"fmt"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/s0ders/go-semver-release/v2/internal/semver"
)

var ErrTagAlreadyExists = errors.New("tag already exists")

type OptionFunc func(t *Tagger)

func WithTagPrefix(prefix string) OptionFunc {
	return func(t *Tagger) {
		t.TagPrefix = prefix
	}
}

func WithSignKey(key *openpgp.Entity) OptionFunc {
	return func(t *Tagger) {
		t.SignKey = key
	}
}

type Tagger struct {
	TagPrefix    string
	GitSignature object.Signature
	SignKey      *openpgp.Entity
}

func NewTagger(name, email string, options ...OptionFunc) *Tagger {
	tagger := &Tagger{
		GitSignature: object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
	}

	for _, option := range options {
		option(tagger)
	}

	return tagger
}

// TagFromSemver creates a new Git annotated tag from a semantic version number.
func (t *Tagger) TagFromSemver(semver *semver.Semver, hash plumbing.Hash) *object.Tag {
	tag := &object.Tag{
		Hash:   hash,
		Name:   semver.String(),
		Tagger: t.GitSignature,
	}

	return tag
}

// Exists check if a given tag name exists on a given Git repository.
func Exists(repository *git.Repository, tagName string) (bool, error) {
	reference, err := repository.Reference(plumbing.NewTagReferenceName(tagName), true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return false, nil
		}
		return false, err
	}

	exists := reference != nil

	return exists, nil
}

// AddTagToRepository create a new annotated tag on the repository with a name corresponding to the semver passed as a
// parameter.
func (t *Tagger) AddTagToRepository(repository *git.Repository, semver *semver.Semver) error {
	head, err := repository.Head()
	if err != nil {
		return fmt.Errorf("fetching head: %w", err)
	}

	tagOpts := &git.CreateTagOptions{
		Message: t.TagPrefix + semver.String(),
		SignKey: t.SignKey,
		Tagger:  &t.GitSignature,
	}

	if exists, err := Exists(repository, tagOpts.Message); err != nil {
		return fmt.Errorf("checking if tag exists: %w", err)
	} else if exists {
		return ErrTagAlreadyExists
	}

	if _, err = repository.CreateTag(tagOpts.Message, head.Hash(), tagOpts); err != nil {
		return fmt.Errorf("creating tag on repository: %w", err)
	}

	return nil
}

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

	"github.com/s0ders/go-semver-release/v8/internal/semver"
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

func WithLightweight(lightweight bool) OptionFunc {
	return func(t *Tagger) {
		t.Lightweight = lightweight
	}
}

type Tagger struct {
	TagPrefix    string
	ProjectName  string
	GitSignature object.Signature
	SignKey      *openpgp.Entity
	Lightweight  bool
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

func (t *Tagger) SetProjectName(name string) {
	t.ProjectName = name
}

// TagFromSemver creates a new Git annotated tag from a semantic version number.
func (t *Tagger) TagFromSemver(semver *semver.Version, hash plumbing.Hash) *object.Tag {
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

// TagRepository creates a new tag on the repository with a name corresponding to the semver passed as a parameter.
// By default, it creates an annotated tag. If Lightweight is set to true, it creates a lightweight tag instead.
func (t *Tagger) TagRepository(repository *git.Repository, semver *semver.Version, commitHash plumbing.Hash) error {
	if semver == nil {
		return fmt.Errorf("semver is nil")
	}

	tagName := t.Format(semver)

	if exists, err := Exists(repository, tagName); err != nil {
		return fmt.Errorf("checking if tag exists: %w", err)
	} else if exists {
		return ErrTagAlreadyExists
	}

	if t.Lightweight {
		// Create lightweight tag (no options)
		if _, err := repository.CreateTag(tagName, commitHash, nil); err != nil {
			return fmt.Errorf("creating lightweight tag on repository: %w", err)
		}
	} else {
		// Create annotated tag
		tagOpts := &git.CreateTagOptions{
			Message: tagName,
			SignKey: t.SignKey,
			Tagger:  &t.GitSignature,
		}

		if _, err := repository.CreateTag(tagName, commitHash, tagOpts); err != nil {
			return fmt.Errorf("creating annotated tag on repository: %w", err)
		}
	}

	return nil
}

func (t *Tagger) Format(semver *semver.Version) string {
	tag := t.TagPrefix + semver.String()

	if t.ProjectName != "" {
		tag = t.ProjectName + "-" + tag
	}

	return tag
}

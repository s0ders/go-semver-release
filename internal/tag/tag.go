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
	"github.com/spf13/viper"

	"github.com/s0ders/go-semver-release/v2/internal/semver"
)

var ErrTagAlreadyExists = errors.New("tag already exists")

var GitSignature = object.Signature{
	Name:  viper.GetString("git-name"),
	Email: viper.GetString("git-email"),
	When:  time.Now(),
}

type OptionFunc func(options *git.CreateTagOptions)

func WithSignKey(key *openpgp.Entity) OptionFunc {
	return func(c *git.CreateTagOptions) {
		c.SignKey = key
	}
}

// NewFromSemver creates a new Git annotated tag from a semantic version number.
func NewFromSemver(semver *semver.Semver, hash plumbing.Hash) *object.Tag {
	tag := &object.Tag{
		Hash:   hash,
		Name:   semver.String(),
		Tagger: GitSignature,
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

// AddToRepository create a new annotated tag on the repository with a name corresponding to the semver passed as a
// parameter.
func AddToRepository(repository *git.Repository, semver *semver.Semver, options ...OptionFunc) error {
	head, err := repository.Head()
	if err != nil {
		return fmt.Errorf("failed to fetch head: %w", err)
	}

	tagOpts := &git.CreateTagOptions{
		Message: viper.GetString("tag-prefix") + semver.String(),
		Tagger:  &GitSignature,
	}

	for _, option := range options {
		option(tagOpts)
	}

	if exists, err := Exists(repository, tagOpts.Message); err != nil {
		return fmt.Errorf("failed to check if tag exists: %w", err)
	} else if exists {
		return ErrTagAlreadyExists
	}

	if _, err = repository.CreateTag(tagOpts.Message, head.Hash(), tagOpts); err != nil {
		return fmt.Errorf("failed to create tag on repository: %w", err)
	}

	return nil
}

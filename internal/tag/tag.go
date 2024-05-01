// Package tag provides function to work with Git tags.
package tag

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/s0ders/go-semver-release/v2/internal/semver"
)

var GitSignature = object.Signature{
	Name:  "Go Semver Release",
	Email: "ci@ci.ci",
	When:  time.Now(),
}

type Options struct {
	Prefix string
}

// NewFromSemver creates a new Git annotated tag from a semantic
// version number.
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

// AddToRepository create a new annotated tag on the repository
// with a name corresponding to the semver passed as a parameter.
func AddToRepository(repository *git.Repository, semver *semver.Semver, opts *Options) error {
	head, err := repository.Head()
	if err != nil {
		return fmt.Errorf("failed to fetch head: %w", err)
	}

	var prefix string

	if opts != nil {
		prefix = opts.Prefix
	}

	tag := fmt.Sprintf("%s%s", prefix, semver.String())

	tagExists, err := Exists(repository, tag)
	if err != nil {
		return fmt.Errorf("failed to check if tag exists: %w", err)
	}

	if tagExists {
		return fmt.Errorf("tag already exists")
	}

	createTagOptions := git.CreateTagOptions{
		Message: semver.String(),
		Tagger:  &GitSignature,
	}

	_, err = repository.CreateTag(tag, head.Hash(), &createTagOptions)
	if err != nil {
		return fmt.Errorf("failed to create tag on repository: %w", err)
	}

	return nil
}

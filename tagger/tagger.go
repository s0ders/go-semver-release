package tagger

import (
	"log"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/s0ders/go-semver-release/semver"
)

type Tagger struct {
	l *log.Logger
}

func NewTagger(l *log.Logger) *Tagger {
	return &Tagger{l}
}

// CreateAndPushNewTag create a new annotated tag on the repository
// with a name corresponding to the semver passed as a parameter.
func (t *Tagger) CreateAndPushNewTag(r *git.Repository, semver *semver.Semver) *git.Repository {
	h, err := r.Head()
	failOnError(err)

	_, err = r.CreateTag(semver.String(), h.Hash(), &git.CreateTagOptions{
		Message: semver.String(),
		Tagger: &object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	})

	failOnError(err)
	t.l.Printf("Created new tag %s on repository", semver.String())

	return r
}

func failOnError(e error) {
	if e != nil {
		log.Fatalf("Error: %s\n", e)
	}
}

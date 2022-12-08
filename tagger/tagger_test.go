package tagger

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestTagExists(t *testing.T) {

	tempDirPath, err := os.MkdirTemp("", "tagger-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %s", err)
	}

	defer os.RemoveAll(tempDirPath)

	r, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		t.Fatalf("failed to initialize git repository: %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %s", err)
	}

	tempFileName := "temp"
	tempFilePath := filepath.Join(tempDirPath, tempFileName)
	_, err = os.Create(tempFilePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %s", err)
	}

	err = os.WriteFile(tempFilePath, []byte("..."), 0644)
	if err != nil {
		t.Fatalf("failed to write to temp file: %s", err)
	}

	_, err = w.Add(tempFileName)
	if err != nil {
		t.Fatalf("failed to add temp file to worktree: %s", err)
	}

	commit, err := w.Commit("firt commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})

	_, err = r.CommitObject(commit)
	if err != nil {
		t.Fatalf("failed to commit object %s", err)
	}

	h, err := r.Head()
	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	tags := []string{"1.0.0", "1.0.2"}

	for i, tag := range tags {
		r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
			Message: tag,
			Tagger: &object.Signature{
				Name:  "Go Semver Release",
				Email: "ci@ci.ci",
				When:  time.Now().Add(time.Duration(i) * time.Hour),
			},
		})
	}

	tagger := NewTagger(log.Default(), "")

	tagExistsTrue, err := tagger.TagExists(r, tags[0])
	if err != nil {
		t.Fatalf("failed to check if tag exists: %s", err)
	}

	if want := true; tagExistsTrue != want {
		t.Fatalf("got: %t want: %t", tagExistsTrue, want)
	}

	tagExistsFalse, err := tagger.TagExists(r, "0.0.1")
	if err != nil {
		t.Fatalf("failed to check if tag exists: %s", err)
	}

	if want := false; tagExistsFalse != want {
		t.Fatalf("got: %t want: %t", tagExistsFalse, want)
	}
}

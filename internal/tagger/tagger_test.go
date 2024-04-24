package tagger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/s0ders/go-semver-release/internal/semver"
)

func TestTagExists(t *testing.T) {
	r, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	h, err := r.Head()
	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	tags := []string{"1.0.0", "1.0.2"}

	for i, tag := range tags {
		_, err := r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
			Message: tag,
			Tagger: &object.Signature{
				Name:  "Go Semver Release",
				Email: "ci@ci.ci",
				When:  time.Now().Add(time.Duration(i) * time.Hour),
			},
		})
		if err != nil {
			t.Fatalf("failed to create tag: %s", err)
		}
	}

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	tagger := New(logger, "")

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

func TestAddTagToRepository(t *testing.T) {
	r, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	semver, err := semver.New(1, 0, 0, "")
	if err != nil {
		t.Fatalf("failed to create semver: %s", err)
	}

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	tagger := New(logger, "")
	taggedRepository, err := tagger.AddTagToRepository(r, semver)
	if err != nil {
		t.Fatalf("failed to tag repository: %s", err)
	}

	tagExists, err := tagger.TagExists(taggedRepository, semver.String())
	if err != nil {
		t.Fatalf("failed to check if tag exists: %s", err)
	}

	if want := true; tagExists != want {
		t.Fatalf("want: %t got: %t", want, tagExists)
	}
}

func createGitRepository(firstCommitMessage string) (*git.Repository, string, error) {
	tempDirPath, err := os.MkdirTemp("", "tagger-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	r, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize git repository: %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get worktree: %s", err)
	}

	tempFileName := "temp"
	tempFilePath := filepath.Join(tempDirPath, tempFileName)
	_, err = os.Create(tempFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp file: %s", err)
	}

	err = os.WriteFile(tempFilePath, []byte("Hello world"), 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("failed to write to temp file: %s", err)
	}

	_, err = w.Add(tempFileName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to add temp file to worktree: %s", err)
	}

	commit, err := w.Commit(firstCommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create commit object %s", err)
	}

	if _, err = r.CommitObject(commit); err != nil {
		return nil, "", fmt.Errorf("failed to commit object %s", err)
	}

	return r, tempDirPath, nil
}

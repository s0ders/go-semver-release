// go-semver-release package aims to be a simple
// program for CI/CD runner that applies the semver
// spec. and conventional commit spec. to a Git repository
// so that version number are automatically and reliably
// handled by Git annotated tags.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/s0ders/go-semver-release/commitanalyzer"
	"github.com/s0ders/go-semver-release/tagger"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

// TODO: Properly handle errors
// TODO: Add a --verbose flag to enable verbose output
// TODO: Add color to console text for verbose mod
func main() {
	logger := log.New(os.Stdout, fmt.Sprintf("%-20s ", "[go-semver-release]"), log.Default().Flags())
	gitUrl := flag.String("url", "", "The Git repository to work on")
	releaseRulesPath := flag.String("rules", "", "Path to a JSON file containing the rules for releasing new version based on commit types")
	accessTokenFlag := flag.String("token", "", "A personnal access token to log in to the Git repository in order to push tags")
	dryrunFlag := flag.Bool("dry-run", false, "Enable dry-run which will only compute the next semantic version number for a repository and not push any tag")

	flag.Parse()

	if *gitUrl == "" {
		logger.Fatal("--url cannot be empty\n")
	}

	auth := &http.BasicAuth{
		Username: "go-semver-release",
		Password: *accessTokenFlag,
	}

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		Auth:     auth,
		URL:      *gitUrl,
		Progress: os.Stdout,
	})

	if err != nil {
		logger.Fatalf("Failed to clone repository: %s", err)
	}

	commitAnalyzer, err := commitanalyzer.NewCommitAnalyzer(log.New(os.Stdout, fmt.Sprintf("%-20s ", "[commit-analyzer]"), log.LstdFlags), releaseRulesPath)
	if err != nil {
		logger.Fatalf("Failed to create commit analyzer: %s", err)
	}

	// Fetch all semantic versioning tags (i.e. vX.Y.Z) from the repository
	latestSemverTag, err := commitAnalyzer.FetchLatestSemverTag(r)
	if err != nil {
		logger.Fatalf("Failed to fetch latest semver tag: %s", err)
	}

	logOptions := &git.LogOptions{}

	if latestSemverTag.Name != "v0.0.0" {
		logOptions.Since = &latestSemverTag.Tagger.When
	}

	commitHistory, err := r.Log(logOptions)
	if err != nil {
		logger.Fatalf("Failed to fetch commit history: %s", err)
	}

	// Compute the next semantic versioning number
	semver, noNewVersion := commitAnalyzer.ComputeNewSemverNumber(commitHistory, latestSemverTag)

	switch {
	case noNewVersion:
		logger.Printf("No new version, still on %s", semver)
		os.Exit(0)
	case *dryrunFlag:
		logger.Printf("Dry-run enabled, next version will be %s", semver)
		os.Exit(0)
	}

	t := tagger.NewTagger(log.New(os.Stdout, fmt.Sprintf("%-20s ", "[tagger]"), log.Default().Flags()))
	r, err = t.AddTagToRepository(r, semver)

	if err != nil {
		logger.Fatalf("Failed to create new tag: %s", err)
	}

	// Push tag to remote
	if err = t.PushTagToRemote(r, auth); err != nil {
		logger.Fatalf("Failed to push tag: %s", err)
	}
	logger.Printf("Pushed tag %s on repository", semver)
	os.Exit(0)
}

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

// TODO: Add a --verbose flag to enable verbose output
// TODO: Add color to console text for verbose mod
func main() {
	logger := log.New(os.Stdout, fmt.Sprintf("%-20s ", "[go-semver-release]"), log.Default().Flags())

	gitUrl 				:= flag.String("url", "", "The Git repository to version")
	releaseRulesPath 	:= flag.String("rules", "", "Path to a JSON file containing the rules for releasing new semantic versions based on commit types")
	accessToken 		:= flag.String("token", "", "A personnal access token to log in to the Git repository in order to push tags")
	dryrun	 			:= flag.Bool("dry-run", false, "Enable dry-run which only computes the next semantic version for a repository, no tags are pushed")

	flag.Parse()

	if *gitUrl == "" {
		logger.Fatal("--url cannot be empty\n")
	}

	auth := &http.BasicAuth{
		Username: "go-semver-release",
		Password: *accessToken,
	}

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		Auth:     auth,
		URL:      *gitUrl,
		Progress: nil,
	})

	if err != nil {
		logger.Fatalf("failed to clone repository: %s", err)
	}

	commitAnalyzer, err := commitanalyzer.NewCommitAnalyzer(log.New(os.Stdout, fmt.Sprintf("%-20s ", "[commit-analyzer]"), log.LstdFlags), releaseRulesPath)
	if err != nil {
		logger.Fatalf("failed to create commit analyzer: %s", err)
	}

	// Fetch all semantic versioning tags (i.e. vX.Y.Z) from the repository
	latestSemverTag, err := commitAnalyzer.FetchLatestSemverTag(r)
	if err != nil {
		logger.Fatalf("failed to fetch latest semver tag: %s", err)
	}

	logOptions := &git.LogOptions{}

	// TODO: fix this mess, handle case where a repo has an existing 0.0.0 tag
	if latestSemverTag.Name != fmt.Sprintf("0.0.0") {
		logOptions.Since = &latestSemverTag.Tagger.When
	}

	commitHistory, err := r.Log(logOptions)
	if err != nil {
		logger.Fatalf("failed to fetch commit history: %s", err)
	}

	// Compute the next semantic versioning number
	semver, noNewVersion, err := commitAnalyzer.ComputeNewSemverNumber(commitHistory, latestSemverTag)
	if err != nil {
		fmt.Printf("failed to compute SemVer: %s", err)
	}

	switch {
	case noNewVersion:
		logger.Printf("no new version, still on %s", semver)
		os.Exit(0)
	case *dryrun:
		logger.Printf("dry-run enabled, next version will be %s", semver)
		os.Exit(0)
	}

	t := tagger.NewTagger(log.New(os.Stdout, fmt.Sprintf("%-20s ", "[tagger]"), log.Default().Flags()))
	r, err = t.AddTagToRepository(r, semver)

	if err != nil {
		logger.Fatalf("failed to create new tag: %s", err)
	}

	// Push tag to remote
	if err = t.PushTagToRemote(r, auth); err != nil {
		logger.Fatalf("Failed to push tag: %s", err)
	}
	logger.Printf("pushed tag %s on repository", semver)
	os.Exit(0)
}

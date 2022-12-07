// go-semver-release package aims to be a simple
// program for CI/CD runner that applies the semver
// spec. and conventional commit spec. to a Git repository
// so that version number are automatically and reliably
// handled by Git annotated tags.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/s0ders/go-semver-release/commitanalyzer"
	"github.com/s0ders/go-semver-release/tagger"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

// TODO: Properly handle errors
// TODO: Add a --verbose flag to enable verbose output
// TODO: Add a --dry-run flag
func main() {
	logger := log.New(os.Stdout, "[main] ", log.Default().Flags())
	gitUrl := flag.String("url", "", "The Git repository to work on")
	releaseRulesPath := flag.String("rules", "", "Path to a JSON file containing the rules for releasing new version based on commit types")
	accessTokenFlag := flag.String("token", "", "A personnal access token to push tag to the Git repository")
	flag.Parse()

	if *gitUrl == "" {
		logger.Fatal("--url cannot be empty\n")
	}

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: "go-semver-release",
			Password: *accessTokenFlag,
		},
		URL:      *gitUrl,
		Progress: os.Stdout,
	})

	if err != nil {
		logger.Fatalf("Failed to clone repository: %s", err)
	}

	tags, err := r.TagObjects()
	
	if err != nil {
		logger.Fatalf("Failed to fetch tags: %s", err)
	}

	commitAnalyzer := commitanalyzer.NewCommitAnalyzer(log.New(os.Stdout, "[commit-analyzer] ", log.Default().Flags()), releaseRulesPath)

	// Fetch all semantic versioning tags (i.e. vX.Y.Z) from the repository
	latestSemverTag := commitAnalyzer.FetchLatestSemverTag(tags)

	commitHistory, err := r.Log(&git.LogOptions{Since: &latestSemverTag.Tagger.When})
	if err != nil {
		logger.Fatalf("Failed to fetch commit history: %s", err)
	}

	// Compute the next semantic versioning number
	semver, noNewVersion := commitAnalyzer.ComputeNewSemverNumber(commitHistory, latestSemverTag)
	
	if noNewVersion {
		logger.Printf("No new version, still on %s", semver)
		os.Exit(0)
	}

	t := tagger.NewTagger(log.New(os.Stdout, "[tagger] ", log.Default().Flags()))
	r = t.CreateAndPushNewTag(r, semver)

	po := &git.PushOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/tags/*:refs/tags/*")},
		Auth:       &http.BasicAuth{
			Username: "go-semver-release",
			Password: *accessTokenFlag,
		},
	}
	
	// Push tag to remote
	err = r.Push(po)
	if err != nil {
		logger.Fatalf("Failed to push tag: %s", err)
	}
	logger.Printf("Pushed tag %s on repository", semver)
	os.Exit(0)
}

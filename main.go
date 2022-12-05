// go-semver-release package aims to be a simple
// program for CI/CD runner that applies the semver
// spec. and conventional commit spec. to a Git repository
// so that version // number are automatically and reliably
// handled by Git annotated tags.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/s0ders/go-semver-release/commitanalyzer"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

// TODO: Read GitHub/GitLab access token from environment or flag
// TODO: Properly handle errors
// TODO: Add a --rules flag that takes a JSON file defining release rule
// TODO: Add a --verbose flag to enable verbose output
func main() {
	gitUrl := flag.String("url", "", "The Git repository to work on")
	releaseRules := flag.String("rules", "", "Path to a JSON file containing the rules for releasing new version based on commit types.")
	flag.Parse()

	if *gitUrl == "" {
		log.Fatalf("--url cannot be empty\n")
	}

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: *gitUrl,
	})
	failOnError(err)

	// Fetch all tags from Git repository
	tags, err := r.TagObjects()
	failOnError(err)

	commitAnalyzer := commitanalyzer.NewCommitAnalyzer(log.New(os.Stdout, "[commit-analyzer] ", log.Lshortfile))

	// Fetch latest semver tag HERE
	latestSemverTag := commitAnalyzer.FetchLatestSemverTag(tags)

	commitHistory, err := r.Log(&git.LogOptions{Since: &latestSemverTag.Tagger.When})
	failOnError(err)
	
	semver := commitAnalyzer.ComputeNewSemverNumber(commitHistory, latestSemverTag, releaseRules)

	fmt.Println("Semver: ", semver)
}

func failOnError(e error) {
	if e != nil {
		log.Fatalf("Error: %s\n", e)
	}
}

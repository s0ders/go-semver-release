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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// TODO: Properly handle errors
// TODO: Add a --verbose flag to enable verbose output
func main() {
	gitUrl := flag.String("url", "", "The Git repository to work on")
	releaseRulesPath := flag.String("rules", "", "Path to a JSON file containing the rules for releasing new version based on commit types")
	accessTokenFlag := flag.String("access-token", "", "A personnal access token to push tag to the Git repository")
	flag.Parse()

	if *gitUrl == "" {
		log.Fatalf("--url cannot be empty\n")
	}

	if *accessTokenFlag == "" {
		log.Fatalf("--access-token cannot be nul")
	}

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: "go-semver-release",
			Password: *accessTokenFlag,
		},
		URL: *gitUrl,
		Progress: os.Stdout,
	})

	failOnError(err)

	tags, err := r.TagObjects()
	failOnError(err)

	commitAnalyzer := commitanalyzer.NewCommitAnalyzer(log.New(os.Stdout, "[commit-analyzer] ", log.Default().Flags()), releaseRulesPath)
	latestSemverTag := commitAnalyzer.FetchLatestSemverTag(tags)

	commitHistory, err := r.Log(&git.LogOptions{Since: &latestSemverTag.Tagger.When})
	failOnError(err)
	
	semver := commitAnalyzer.ComputeNewSemverNumber(commitHistory, latestSemverTag)

	fmt.Println("Semver: ", semver)
}

func failOnError(e error) {
	if e != nil {
		log.Fatalf("Error: %s\n", e)
	}
}

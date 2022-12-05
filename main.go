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

	"github.com/s0ders/go-semver-release/commitanalyzer"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

// TODO: take an input file "release rule"
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

	// Fetch latest semver tag HERE
	latestSemverTag := commitanalyzer.FetchLatestSemverTag(tags)

	commitHistory, err := r.Log(&git.LogOptions{Since: &latestSemverTag.Tagger.When})
	failOnError(err)
	
	semver := commitanalyzer.ComputeNewSemverNumber(commitHistory, latestSemverTag, releaseRules)

	fmt.Println("Semver: ", semver)
}

func failOnError(e error) {
	if e != nil {
		log.Fatalf("Error: %s\n", e)
	}
}

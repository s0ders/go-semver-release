package main

import (
	"flag"
	"log"

	"github.com/s0ders/go-semver-release/commitanalyzer"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

func main() {
	gitUrl := flag.String("url", "", "The Git repository to work on")
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
	
	commitanalyzer.ComputeNewSemverNumber(commitHistory)
}

func failOnError(e error) {
	if e != nil {
		log.Fatalf("Error: %s\n", e)
	}
}

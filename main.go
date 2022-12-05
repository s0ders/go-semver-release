package main

import (
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"log"
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

	if err != nil {
		log.Fatalf("Could not clone repository: %s\n", err)
	}

	ref, err := r.Head()

	commitHistory, err := r.Log(&git.LogOptions{From: ref.Hash()})

	commitHistory.ForEach(func(commit *object.Commit) error {
		fmt.Println(commit.Message)
		return nil
	})
}

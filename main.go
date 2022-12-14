// go-semver-release package aims to be a simple
// program for CI/CD runner that applies the semver
// spec. and conventional commit spec. to a Git repository
// so that version number are automatically and reliably
// handled by Git annotated tags.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/s0ders/go-semver-release/commitanalyzer"
	"github.com/s0ders/go-semver-release/tagger"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var (
	releaseRulesPath    string
	gitUrl              string
	accessToken         string
	tagPrefix           string
	releaseBranch       string
	dryrun              bool
	defaultReleaseRules = `{
		"releaseRules": [
			{"type": "feat", "release": "minor"},
			{"type": "perf", "release": "minor"},
			{"type": "fix", "release": "patch"}
		]
	}`
)

func main() {
	logger := log.New(os.Stdout, fmt.Sprintf("%-20s ", "[go-semver-release]"), log.Default().Flags())

	flag.StringVar(&releaseRulesPath, "rules", "", "Path to a JSON file containing the rules for releasing new semantic versions based on commit types")
	flag.StringVar(&gitUrl, "url", "", "The Git repository to version")
	flag.StringVar(&accessToken, "token", "", "A personnal access token to log in to the Git repository in order to push tags")
	flag.StringVar(&tagPrefix, "tag-prefix", "", "A prefix to append to the semantic version number used to name tag (e.g. 'v') and used to match existing tags on remote")
	flag.StringVar(&releaseBranch, "branch", "", "The branch to check commit history from (e.g. \"main\", \"master\", \"release\"), will default to the main branch if empty")
	flag.BoolVar(&dryrun, "dry-run", false, "Enable dry-run which only computes the next semantic version for a repository, no tags are pushed")
	flag.Parse()

	if gitUrl == "" {
		logger.Fatal("--url cannot be empty\n")
	}

	auth := &http.BasicAuth{
		Username: "go-semver-release",
		Password: accessToken,
	}

	gitDirectoryPath, err := os.MkdirTemp("", "go-semver-release-*")
	defer os.RemoveAll(gitDirectoryPath)
	if err != nil {
		logger.Fatalf("failed to temporary directory to clone repository: %s", err)
	}

	cloneOption := &git.CloneOptions{
		Auth:     auth,
		URL:      gitUrl,
		Progress: nil,
	}

	if releaseBranch != "" {
		cloneOption.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", releaseBranch))
	}

	r, err := git.PlainClone(gitDirectoryPath, false, cloneOption)

	if err != nil {
		logger.Fatalf("failed to clone repository: %s", err)
	}

	var releaseRulesReader io.Reader
	if releaseRulesPath == "" {
		releaseRulesReader = strings.NewReader(defaultReleaseRules)
	} else {
		releaseRulesReader, err = os.Open(releaseRulesPath)
		if err != nil {
			logger.Fatalf("failed to open release rules from path: %s", err)
		}
	}

	commitAnalyzer, err := commitanalyzer.NewCommitAnalyzer(log.New(os.Stdout, fmt.Sprintf("%-20s ", "[commit-analyzer]"), log.LstdFlags), releaseRulesReader)
	if err != nil {
		logger.Fatalf("failed to create commit analyzer: %s", err)
	}

	latestSemverTag, err := commitAnalyzer.FetchLatestSemverTag(r)
	if err != nil {
		logger.Fatalf("failed to fetch latest semver tag: %s", err)
	}

	semver, newRelease, err := commitAnalyzer.ComputeNewSemverNumber(r, latestSemverTag)
	if err != nil {
		fmt.Printf("failed to compute SemVer: %s", err)
	}

	if !newRelease {
		logger.Printf("no new release, still on %s", semver)
		os.Exit(0)
	}

	if dryrun {
		logger.Printf("dry-run enabled, next version will be %s", semver)
		os.Exit(0)
	}

	t := tagger.NewTagger(log.New(os.Stdout, fmt.Sprintf("%-20s ", "[tagger]"), log.Default().Flags()), tagPrefix)

	r, err = t.AddTagToRepository(r, semver)

	if err != nil {
		logger.Fatalf("failed to create new tag: %s", err)
	}

	if err = t.PushTagToRemote(r, auth); err != nil {
		logger.Fatalf("Failed to push tag: %s", err)
	}

	logger.Printf("pushed tag %s on repository", semver)
	os.Exit(0)
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/s0ders/go-semver-release/internal/cloner"
	"github.com/s0ders/go-semver-release/internal/commitanalyzer"
	"github.com/s0ders/go-semver-release/internal/output"
	"github.com/s0ders/go-semver-release/internal/releaserules"
	"github.com/s0ders/go-semver-release/internal/tagger"
)

var (
	rulesPath     string
	gitUrl        string
	token         string
	prefix        string
	releaseBranch string
	dryrunFlag    string
)

func main() {
	logger := log.New(os.Stdout, fmt.Sprintf("%-20s ", "[go-semver-release]"), log.Default().Flags())

	flag.StringVar(&rulesPath, "rules", "", "Path to a JSON file containing the rules for releasing new semantic versions based on commit types")
	flag.StringVar(&gitUrl, "url", "", "The Git repository to version")
	flag.StringVar(&token, "token", "", "A personnal access token to log in to the Git repository in order to push tags")
	flag.StringVar(&prefix, "tag-prefix", "", "A prefix to append to the semantic version number used to name tag (e.g. 'v') and used to match existing tags on remote")
	flag.StringVar(&releaseBranch, "branch", "", "The branch to check commit history from (e.g. \"main\", \"master\", \"release\"), will default to the branch pointed by HEAD if empty")
	flag.StringVar(&dryrunFlag, "dry-run", "false", "Enable dry-run which only computes the next semantic version for a repository, no tags are pushed")
	flag.Parse()

	if gitUrl == "" {
		logger.Fatal("--url cannot be empty\n")
	}

	dryrun, err := strconv.ParseBool(dryrunFlag)
	if err != nil {
		logger.Fatalf("failed to parse --dry-run value")
	}

	repository, path := cloner.NewCloner().Clone(gitUrl, releaseBranch, token)
	defer os.RemoveAll(path)

	rules, err := releaserules.NewReleaseRuleReader().Read(rulesPath).Parse()
	if err != nil {
		logger.Fatalf("failed to parse rules: %s", err)
	}

	semver, release, err := commitanalyzer.NewCommitAnalyzer(rules).ComputeNewSemverNumber(repository)
	if err != nil {
		logger.Fatalf("failed to compute semver: %s", err)
	}

	output.NewOutput().Generate(prefix, semver, release)

	if !release {
		logger.Printf("no new release, still on %s", semver)
		os.Exit(0)
	}

	if dryrun {
		logger.Printf("dry-run enabled, next version will be %s", semver)
		os.Exit(0)
	}

	if err = tagger.NewTagger(prefix).PushTagToRemote(repository, token, semver); err != nil {
		logger.Fatalf("Failed to push tag: %s", err)
	}

	os.Exit(0)
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/s0ders/go-semver-release/internal/cloner"
	"github.com/s0ders/go-semver-release/internal/commitanalyzer"
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

	r := cloner.NewCloner().Clone(gitUrl, releaseBranch, token)

	rules, err := releaserules.NewReleaseRuleReader().Read(rulesPath)
	if err != nil {
		logger.Fatalf("failed to parse release rules: %s", err)
	}

	commitAnalyzer, err := commitanalyzer.NewCommitAnalyzer(rules)
	if err != nil {
		logger.Fatalf("failed to create commit analyzer: %s", err)
	}

	semver, newRelease, err := commitAnalyzer.ComputeNewSemverNumber(r)
	if err != nil {
		logger.Fatalf("failed to compute semver: %s", err)
	}

	outputFile := os.Getenv("GITHUB_OUTPUT")
	output := fmt.Sprintf("\nSEMVER=%s%s\nNEW_RELEASE=%t", prefix, semver.NormalVersion(), newRelease)
	if err = os.WriteFile(outputFile, []byte(output), os.ModeAppend); err != nil {
		logger.Fatalf("failed to generate output: %s", err)
	}

	if !newRelease {
		logger.Printf("no new release, still on %s", semver)
		os.Exit(0)
	}

	if dryrun {
		logger.Printf("dry-run enabled, next version will be %s", semver)
		os.Exit(0)
	}

	if err = tagger.NewTagger(prefix).PushTagToRemote(r, token, semver); err != nil {
		logger.Fatalf("Failed to push tag: %s", err)
	}

	os.Exit(0)
}

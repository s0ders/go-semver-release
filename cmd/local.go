package cmd

import (
	"log/slog"

	"github.com/go-git/go-git/v5"
	"github.com/s0ders/go-semver-release/v2/internal/ci"
	"github.com/s0ders/go-semver-release/v2/internal/parser"
	"github.com/s0ders/go-semver-release/v2/internal/rules"
	"github.com/s0ders/go-semver-release/v2/internal/tagger"
	"github.com/spf13/cobra"
)

var (
	rulesPath     string
	tagPrefix     string
	releaseBranch string
	dryRun        bool
	verbose       bool
	jsonOutput    bool
)

func init() {
	localCmd.Flags().StringVarP(&rulesPath, "rules-path", "r", "", "Path to the JSON or YAML file containing the release rules")
	localCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "t", "v", "Prefix added to the version tag name")
	localCmd.Flags().StringVarP(&releaseBranch, "release-branch", "b", "main", "Branch to fetch commits from")
	localCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next semver, do not push any tag")
	localCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose ci")
	localCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "JSON formatted output")

	rootCmd.AddCommand(localCmd)
}

var localCmd = &cobra.Command{
	Use:   "local <REPOSITORY_PATH>",
	Short: "Version a local repository",
	Long:  "Version a local repository by adding an annotated tag named after the right semver allowing you to push it back to your remote without sharing any secret token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var logHandler slog.Handler
		var logLevel slog.Level

		if verbose {
			logLevel = slog.LevelDebug
		} else {
			logLevel = slog.LevelInfo
		}

		logOpts := &slog.HandlerOptions{
			Level: logLevel,
		}

		if jsonOutput {
			logHandler = slog.NewJSONHandler(cmd.OutOrStdout(), logOpts)
		} else {
			logHandler = slog.NewTextHandler(cmd.OutOrStdout(), logOpts)
		}

		logger := slog.New(logHandler)

		repository, err := git.PlainOpen(args[0])
		if err != nil {
			return err
		}

		rulesReader, err := rules.New(logger).Read(rulesPath)
		if err != nil {
			return err
		}

		rules, err := rulesReader.Parse()
		if err != nil {
			return err
		}

		semver, release, err := parser.New(logger, rules).ComputeNewSemver(repository)
		if err != nil {
			return err
		}

		err = ci.New(logger).GenerateGitHub(tagPrefix, semver, release)
		if err != nil {
			return err
		}

		switch {
		case !release:
			logger.Info("no new release", "current-version", semver.String(), "new-release", false)
			return nil
		case release && dryRun:
			logger.Info("new release found, dry-run is enabled", "next-version", semver.String(), "new-release", true)
			return nil
		default:
			logger.Info("new release found", "new-version", semver.String(), "new-release", true)
			err = tagger.New(logger, tagPrefix).AddTagToRepository(repository, semver)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

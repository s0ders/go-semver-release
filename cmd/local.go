package cmd

import (
	"log/slog"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/s0ders/go-semver-release/internal/ci"
	"github.com/s0ders/go-semver-release/internal/parser"
	"github.com/s0ders/go-semver-release/internal/rules"
	"github.com/s0ders/go-semver-release/internal/tagger"
	"github.com/spf13/cobra"
)

func init() {
	localCmd.Flags().StringVarP(&rulesPath, "rules-path", "r", "", "Path to the JSON or YAML file containing the release rules")
	localCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "p", "v", "Prefix added to the version tag name")
	localCmd.Flags().StringVarP(&releaseBranch, "release-branch", "b", "main", "Branch to fetch commits from")
	localCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next semver, do not push any tag")
	localCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose ci")
	localCmd.Flags().BoolVarP(&json, "json", "j", false, "JSON formatted output")

	rootCmd.AddCommand(localCmd)
}

var localCmd = &cobra.Command{
	Use:   "local <REPOSITORY_PATH>",
	Short: "Version a local repository",
	Long:  "Version a local repository by adding an annotated tag named after the right semver allowing you to push it back to your remote without sharing any secret token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var logHandler slog.Handler

		if json {
			logHandler = slog.NewJSONHandler(os.Stdout, nil)
		} else {
			logHandler = slog.NewTextHandler(os.Stdout, nil)
		}

		logger := slog.New(logHandler)

		repo, err := git.PlainOpen(args[0])
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

		semver, release, err := parser.New(logger, rules, verbose).ComputeNewSemver(repo)
		if err != nil {
			return err
		}

		err = ci.New(logger).GenerateGitHub(tagPrefix, semver, release)
		if err != nil {
			return err
		}

		switch {
		case !release:
			logger.Info("no new release", "current-version", semver.NormalVersion())
			return nil
		case release && dryRun:
			logger.Info("new release found, dry-run is enabled", "next-version", semver)
			return nil
		default:
			_, err = tagger.New(logger, tagPrefix, verbose).AddTagToRepository(repo, semver)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

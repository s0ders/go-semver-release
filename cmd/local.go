package cmd

import (
	"github.com/go-git/go-git/v5"
	"github.com/s0ders/go-semver-release/internal/commitanalyzer"
	"github.com/s0ders/go-semver-release/internal/output"
	"github.com/s0ders/go-semver-release/internal/releaserules"
	"github.com/s0ders/go-semver-release/internal/tagger"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

func init() {
	localCmd.Flags().StringVarP(&rulesPath, "rules-path", "r", "", "Path to the JSON containing the release rules")
	localCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "p", "v", "Prefix added to the version tag name")
	localCmd.Flags().StringVarP(&releaseBranch, "release-branch", "b", "main", "Branch to fetch commits from")
	localCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next semver, do not push any tag")
	localCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(localCmd)
}

var localCmd = &cobra.Command{
	Use:   "local <REPOSITORY_PATH>",
	Short: "Version a local repository",
	Long:  "Version a local repository by adding an annotated tag named after the right semver allowing you to push it back to your remote without sharing any secret token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		repo, err := git.PlainOpen(args[0])
		if err != nil {
			return err
		}

		rulesReader, err := releaserules.New(logger).Read(rulesPath)
		if err != nil {
			return err
		}

		rules, err := rulesReader.Parse()
		if err != nil {
			return err
		}

		semver, release, err := commitanalyzer.New(logger, rules, verbose).ComputeNewSemver(repo)
		if err != nil {
			return err
		}

		err = output.NewOutput(logger).Generate(tagPrefix, semver, release)
		if err != nil {
			return err
		}

		if !release {
			logger.Info("no new release", "current-version", semver)
		}

		if release && dryRun {
			logger.Info("new release found, dry-run is enabled", "next-version", semver)
		}

		_, err = tagger.NewTagger(tagPrefix).AddTagToRepository(repo, semver)
		if err != nil {
			return err
		}

		return nil
	},
}

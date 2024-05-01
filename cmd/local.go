package cmd

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/s0ders/go-semver-release/v2/internal/ci"
	"github.com/s0ders/go-semver-release/v2/internal/parser"
	"github.com/s0ders/go-semver-release/v2/internal/rules"
	"github.com/s0ders/go-semver-release/v2/internal/tag"
)

var (
	rulesPath     string
	tagPrefix     string
	releaseBranch string
	dryRun        bool
)

func init() {
	localCmd.Flags().StringVarP(&rulesPath, "rules-path", "r", "", "Path to the JSON or YAML file containing the release rules")
	localCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "t", "v", "Prefix added to the version tag name")
	localCmd.Flags().StringVarP(&releaseBranch, "release-branch", "b", "main", "Branch to fetch commits from")
	localCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next semver, do not push any tag")

	rootCmd.AddCommand(localCmd)
}

var localCmd = &cobra.Command{
	Use:   "local <REPOSITORY_PATH>",
	Short: "Version a local repository",
	Long:  "Version a local repository by adding an annotated tag named after the right semver allowing you to push it back to your remote without sharing any secret token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var rulesOpts rules.Options

		logger := zerolog.New(cmd.OutOrStdout())

		if verbose {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}

		repository, err := git.PlainOpen(args[0])
		if err != nil {
			return err
		}

		if rulesPath != "" {
			file, err := os.Open(rulesPath)
			if err != nil {
				return err
			}

			rulesOpts.Reader = file

			defer func() {
				err = file.Close()
				return
			}()
		}

		rules, err := rules.Init(&rulesOpts)
		if err != nil {
			return err
		}

		semver, release, err := parser.New(logger, rules).ComputeNewSemver(repository)
		if err != nil {
			return err
		}

		err = ci.GenerateGitHubOutput(tagPrefix, semver, release)
		if err != nil {
			return err
		}

		switch {
		case !release:
			logger.Info().Str("current-version", semver.String()).Bool("new-release", false).Msg("no new release")
			return nil
		case release && dryRun:
			logger.Info().Str("next-version", semver.String()).Bool("new-release", true).Msg("new release found, dry-run is enabled")
			return nil
		default:
			logger.Info().Str("new-version", semver.String()).Bool("new-release", true).Msg("new release found")

			err = tag.AddTagToRepository(repository, semver, tagPrefix)
			if err != nil {
				return err
			}

			logger.Debug().Str("tag", tagPrefix+semver.String()).Msg("new tag added to repository")
		}

		return nil
	},
}

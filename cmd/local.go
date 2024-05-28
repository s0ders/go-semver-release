package cmd

import (
	"bytes"
	"github.com/spf13/viper"
	"io"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/s0ders/go-semver-release/v2/internal/ci"
	"github.com/s0ders/go-semver-release/v2/internal/gpg"
	"github.com/s0ders/go-semver-release/v2/internal/parser"
	"github.com/s0ders/go-semver-release/v2/internal/rule"
	"github.com/s0ders/go-semver-release/v2/internal/tag"
)

var (
	rulesPath        string
	tagPrefix        string
	releaseBranch    string
	armoredKeyPath   string
	buildMetadata    string
	prereleaseSuffix string
	dryRun           bool
	prerelease       bool
)

func init() {
	localCmd.Flags().StringVarP(&rulesPath, "rule-path", "r", "", "Path to the JSON or YAML file containing the release rule")
	localCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "t", "", "Prefix added to the version tag name")
	localCmd.Flags().StringVarP(&releaseBranch, "release-branch", "b", "main", "Branch to fetch commits from")
	localCmd.Flags().StringVar(&armoredKeyPath, "gpg-key-path", "", "Path to an armored GPG key used to sign produced tags")
	localCmd.Flags().StringVar(&buildMetadata, "build-metadata", "", "Build metadata (e.g. build number) that will be appended to the semantic version")
	localCmd.Flags().StringVar(&prereleaseSuffix, "prerelease-suffix", "rc", "Suffix appended to the tag if in prerelease mode")
	localCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next semver, do not push any tag")
	localCmd.Flags().BoolVar(&prerelease, "prerelease", false, "Whether or not the semantic version is a prerelease")

	viper.BindPFlag("tag-prefix", localCmd.Flags().Lookup("tag-prefix"))
	viper.BindPFlag("prerelease-suffix", localCmd.Flags().Lookup("prerelease-suffix"))

	rootCmd.AddCommand(localCmd)
}

var localCmd = &cobra.Command{
	Use:   "local <REPOSITORY_PATH>",
	Short: "Version a local Git repository",
	Long:  "Tag a Git repository with the new semantic version number if a new release is found",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var rulesReader io.Reader
		var entity *openpgp.Entity

		logger := zerolog.New(cmd.OutOrStdout())

		if verbose {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}

		if armoredKeyPath != "" {
			armoredKeyFile, err := os.ReadFile(armoredKeyPath)
			if err != nil {
				return err
			}

			entity, err = gpg.FromArmored(bytes.NewReader(armoredKeyFile))
			if err != nil {
				return err
			}
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

			rulesReader = file

			defer func() {
				err = file.Close()
			}()
		}

		rules, err := rule.Init(rule.WithReader(rulesReader))
		if err != nil {
			return err
		}

		parser := parser.New(logger, rules, parser.WithReleaseBranch(releaseBranch), parser.WithBuildMetadata(buildMetadata), parser.WithPrereleaseMode(prerelease))
		semver, release, err := parser.ComputeNewSemver(repository)
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

			err = tag.AddToRepository(repository, semver, tag.WithSignKey(entity))
			if err != nil {
				return err
			}

			logger.Debug().Str("tag", tagPrefix+semver.String()).Msg("new tag added to repository")
		}

		return
	},
}

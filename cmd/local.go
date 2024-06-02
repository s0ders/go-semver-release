package cmd

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
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
	rulesPath           string
	tagPrefix           string
	releaseBranch       string
	armoredKeyPath      string
	buildMetadata       string
	prereleasIdentifier string
	dryRun              bool
	prereleaseMode      bool
)

func init() {
	localCmd.Flags().StringVarP(&rulesPath, "rules", "r", "", "JSON release rules used for versioning")
	localCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "t", "", "Prefix added to the version tag name")
	localCmd.Flags().StringVarP(&releaseBranch, "release-branch", "b", "master", "Branch to fetch commits from")
	localCmd.Flags().StringVar(&armoredKeyPath, "gpg-key-path", "", "Path to an armored GPG key used to sign produced tags")
	localCmd.Flags().StringVar(&buildMetadata, "build-metadata", "", "Build metadata (e.g. build number) that will be appended to the SemVer")
	localCmd.Flags().StringVar(&prereleasIdentifier, "prereleaseMode-identifier", "rc", "Identifier used for the prereleaseMode part of the SemVer (e.g. rc, alpha, beta)")
	localCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next SemVer, do not push any tag")
	localCmd.Flags().BoolVar(&prereleaseMode, "prereleaseMode", false, "Whether or not the SemVer is a prerelease")

	cobra.CheckErr(viper.BindPFlag("tag-prefix", localCmd.Flags().Lookup("tag-prefix")))
	cobra.CheckErr(viper.BindPFlag("rules", localCmd.Flags().Lookup("rules")))
	cobra.CheckErr(viper.BindPFlag("prereleaseMode-identifier", localCmd.Flags().Lookup("prereleaseMode-identifier")))

	rootCmd.AddCommand(localCmd)
}

var localCmd = &cobra.Command{
	Use:   "local <REPOSITORY_PATH>",
	Short: "Version a local Git repository",
	Long:  "Tag a Git repository with the new semantic version number if a new release is found",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var rules rule.Rules
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
				return fmt.Errorf("reading armored key: %w", err)
			}

			entity, err = gpg.FromArmored(bytes.NewReader(armoredKeyFile))
			if err != nil {
				return fmt.Errorf("loading armored key: %w", err)
			}
		}

		repository, err := git.PlainOpen(args[0])
		if err != nil {
			return err
		}

		if viper.IsSet("rules") {
			err = viper.UnmarshalKey("rules", &rules.Unmarshalled)
			if err != nil {
				return fmt.Errorf("unmarshalling rules: %w", err)
			}

			if err = rules.Validate(); err != nil {
				return fmt.Errorf("validating rules: %w", err)
			}
		} else {
			rules = rule.Default
		}

		tagger := tag.NewTagger(gitName, gitEmail, tag.WithTagPrefix(tagPrefix), tag.WithSignKey(entity))

		parser := parser.New(logger, tagger, rules,
			parser.WithReleaseBranch(releaseBranch),
			parser.WithBuildMetadata(buildMetadata),
			parser.WithPrereleaseMode(prereleaseMode),
			parser.WithPrereleaseIdentifier(prereleasIdentifier))

		semver, release, err := parser.ComputeNewSemver(repository)
		if err != nil {
			return fmt.Errorf("computing new semver: %w", err)
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

			err = tagger.TagRepository(repository, semver)
			if err != nil {
				return fmt.Errorf("tagging repository: %w", err)
			}

			logger.Debug().Str("tag", tagPrefix+semver.String()).Msg("new tag added to repository")
		}

		return
	},
}

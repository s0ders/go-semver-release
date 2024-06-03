package cmd

import (
	"bytes"
	"fmt"
	"github.com/s0ders/go-semver-release/v2/internal/branch"
	"github.com/spf13/viper"
	"os"
	"sync"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/s0ders/go-semver-release/v2/internal/ci"
	"github.com/s0ders/go-semver-release/v2/internal/gpg"
	"github.com/s0ders/go-semver-release/v2/internal/parser"
	"github.com/s0ders/go-semver-release/v2/internal/rule"
	"github.com/s0ders/go-semver-release/v2/internal/tag"
)

var (
	tagPrefix      string
	armoredKeyPath string
	buildMetadata  string
	dryRun         bool
)

func init() {
	localCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "t", "", "Prefix added to the version tag name")
	localCmd.Flags().StringVar(&armoredKeyPath, "gpg-key-path", "", "Path to an armored GPG key used to sign produced tags")
	localCmd.Flags().StringVar(&buildMetadata, "build-metadata", "", "Build metadata (e.g. build number) that will be appended to the SemVer")
	localCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next SemVer, do not push any tag")

	cobra.CheckErr(viper.BindPFlag("tag-prefix", localCmd.Flags().Lookup("tag-prefix")))
	cobra.CheckErr(viper.BindPFlag("rules", localCmd.Flags().Lookup("rules")))

	rootCmd.AddCommand(localCmd)
}

var localCmd = &cobra.Command{
	Use:   "local <REPOSITORY_PATH>",
	Short: "Version a local Git repository",
	Long:  "Tag a Git repository with the new semantic version number if a new release is found",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var (
			rules    rule.Rules
			branches []branch.Branch
			entity   *openpgp.Entity
		)

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

		if !viper.IsSet("rules") {
			rules = rule.Default
		} else {
			var rulesMarshalled map[string][]string

			err = viper.UnmarshalKey("rules", &rulesMarshalled)
			if err != nil {
				return fmt.Errorf("unmarshalling rules: %w", err)
			}

			rules, err = rule.Unmarshall(rulesMarshalled)
			if err != nil {
				return err
			}
		}

		if !viper.IsSet("branches") {
			return fmt.Errorf("missing branches key in configuration")
		}

		var branchesMarshalled []map[string]string

		err = viper.UnmarshalKey("branches", &branchesMarshalled)
		if err != nil {
			return fmt.Errorf("unmarshalling branches: %w", err)
		}

		branches, err = branch.Unmarshall(branchesMarshalled)
		if err != nil {
			return fmt.Errorf("unmarshalling branches: %w", err)
		}

		tagger := tag.NewTagger(gitName, gitEmail, tag.WithTagPrefix(tagPrefix), tag.WithSignKey(entity))

		err = viper.UnmarshalKey("branches", &branches)
		if err != nil {
			return fmt.Errorf("unmarshalling branches: %w", err)
		}

		group, _ := errgroup.WithContext(cmd.Context())
		var mu sync.RWMutex

		// Launch a parser per branch to analyze
		for _, branch := range branches {
			group.Go(func() error {
				parser := parser.New(logger, tagger, rules,
					parser.WithReleaseBranch(branch.Pattern),
					parser.WithPrereleaseMode(branch.Prerelease),
					parser.WithPrereleaseIdentifier(branch.PrereleaseIdentifier),
					parser.WithBuildMetadata(buildMetadata),
				)

				mu.RLock()
				semver, release, err := parser.ComputeNewSemver(repository)
				mu.RUnlock()
				if err != nil {
					return fmt.Errorf("computing new semver: %w", err)
				}

				// TODO: handle multi branch and parallelism
				mu.Lock()
				err = ci.GenerateGitHubOutput(branch.Pattern, tagPrefix, semver, release)
				mu.Unlock()
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

					mu.Lock()
					err = tagger.TagRepository(repository, semver)
					mu.Unlock()
					if err != nil {
						return fmt.Errorf("tagging repository: %w", err)
					}

					logger.Debug().Str("tag", tagPrefix+semver.String()).Msg("new tag added to repository")
				}

				return nil
			})
		}

		if err := group.Wait(); err != nil {
			return err
		}

		return
	},
}

package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/s0ders/go-semver-release/v3/internal/branch"
	"github.com/s0ders/go-semver-release/v3/internal/ci"
	"github.com/s0ders/go-semver-release/v3/internal/gpg"
	"github.com/s0ders/go-semver-release/v3/internal/parser"
	"github.com/s0ders/go-semver-release/v3/internal/rule"
	"github.com/s0ders/go-semver-release/v3/internal/tag"
)

var (
	armoredKeyPath string
	buildMetadata  string
	dryRun         bool
)

func init() {
	localCmd.Flags().StringVar(&armoredKeyPath, "gpg-key-path", "", "Path to an armored GPG key used to sign produced tags")
	localCmd.Flags().StringVar(&buildMetadata, "build-metadata", "", "Build metadata (e.g. build number) that will be appended to the SemVer")
	localCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next SemVer, do not push any tag")

	rootCmd.AddCommand(localCmd)
}

var localCmd = &cobra.Command{
	Use:   "local <REPOSITORY_PATH>",
	Short: "Version a local Git repository",
	Long:  "Tag a Git repository with the new semantic version number if a new release is found",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
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

		rules, err := configureRules()
		if err != nil {
			return fmt.Errorf("loading rules configuration: %w", err)
		}

		branches, err := configureBranches()
		if err != nil {
			return fmt.Errorf("loading branches configuration: %w", err)
		}

		tagger := tag.NewTagger(gitName, gitEmail, tag.WithTagPrefix(tagPrefix), tag.WithSignKey(entity))

		// Launch a parser per branch to analyze
		for _, branch := range branches {
			// TODO: optimize parser creation, create one and update branch
			parser := parser.New(logger, tagger, rules,
				parser.WithReleaseBranch(branch.Name),
				parser.WithPrereleaseMode(branch.Prerelease),
				parser.WithPrereleaseIdentifier(branch.Name),
				parser.WithBuildMetadata(buildMetadata),
			)

			computeSemverOutput, err := parser.ComputeNewSemver(repository)
			if err != nil {
				return fmt.Errorf("computing new semver: %w", err)
			}

			semver := computeSemverOutput.Semver
			release := computeSemverOutput.NewRelease
			commitHash := computeSemverOutput.CommitHash

			err = ci.GenerateGitHubOutput(semver, branch.Name, ci.WithNewRelease(release), ci.WithTagPrefix(tagPrefix))
			if err != nil {
				return fmt.Errorf("generating github output: %w", err)
			}

			logEvent := logger.Info()
			logEvent.Bool("new-release", release)
			logEvent.Str("version", semver.String())
			logEvent.Str("branch", branch.Name)

			switch {
			case !release:
				logEvent.Msg("no new release")
				return nil
			case release && dryRun:
				logEvent.Msg("dry-run enabled, next release found")
				return nil
			default:
				logEvent.Msg("new release found")

				err = tagger.TagRepository(repository, semver, commitHash)
				if err != nil {
					return fmt.Errorf("tagging repository: %w", err)
				}

				logger.Debug().Str("tag", tagPrefix+semver.String()).Msg("new tag added to repository")
			}

		}

		return nil
	},
}

func configureRules() (rule.Rules, error) {
	if !viperInstance.IsSet("rules") {
		return rule.Default, nil
	}

	var (
		rulesMarshalled map[string][]string
		rules           rule.Rules
	)

	err := viperInstance.UnmarshalKey("rules", &rulesMarshalled)
	if err != nil {
		return rules, fmt.Errorf("unmarshalling rules key: %w", err)
	}

	rules, err = rule.Unmarshall(rulesMarshalled)
	if err != nil {
		return rules, fmt.Errorf("parsing rules: %w", err)
	}

	return rules, nil
}

func configureBranches() ([]branch.Branch, error) {
	if !viperInstance.IsSet("branches") {
		return nil, fmt.Errorf("missing branches key in configuration")
	}

	var (
		branchesMarshalled []map[string]string
		branches           []branch.Branch
	)

	err := viperInstance.UnmarshalKey("branches", &branchesMarshalled)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling branches: %w", err)
	}

	branches, err = branch.Unmarshall(branchesMarshalled)
	if err != nil {
		return nil, fmt.Errorf("parsing branches: %w", err)
	}

	return branches, nil
}

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"

	"github.com/s0ders/go-semver-release/v5/internal/branch"
	"github.com/s0ders/go-semver-release/v5/internal/ci"
	"github.com/s0ders/go-semver-release/v5/internal/gpg"
	"github.com/s0ders/go-semver-release/v5/internal/monorepo"
	"github.com/s0ders/go-semver-release/v5/internal/parser"
	"github.com/s0ders/go-semver-release/v5/internal/remote"
	"github.com/s0ders/go-semver-release/v5/internal/rule"
	"github.com/s0ders/go-semver-release/v5/internal/tag"
)

var (
	buildMetadata string
	dryRun        bool
)

func NewReleaseCmd(ctx *AppContext) *cobra.Command {
	releaseCmd := &cobra.Command{
		Use:   "release <REPOSITORY_PATH_OR_URL>",
		Short: "Version a local Git repository",
		Long:  "Tag a Git repository with the new semantic version number if a new release is found",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var (
				repository *git.Repository
				origin     *remote.Remote
			)

			if remoteMode {
				origin = remote.New(remoteName, accessToken)
				repository, err = origin.Clone(args[0])
				if err != nil {
					return fmt.Errorf("cloning Git repository: %w", err)
				}
			} else {
				repository, err = git.PlainOpen(args[0])
				if err != nil {
					return fmt.Errorf("opening local Git repository: %w", err)
				}
			}

			entity, err := configureGPGKey(ctx)
			if err != nil {
				return fmt.Errorf("configuring GPG key: %w", err)
			}

			rules, err := configureRules(ctx)
			if err != nil {
				return fmt.Errorf("loading rules configuration: %w", err)
			}

			branches, err := configureBranches(ctx)
			if err != nil {
				return fmt.Errorf("loading branches configuration: %w", err)
			}

			projects, err := configureProjects(ctx)
			if err != nil {
				return fmt.Errorf("loading projects configuration: %w", err)
			}

			tagger := tag.NewTagger(gitName, gitEmail, tag.WithTagPrefix(tagPrefix), tag.WithSignKey(entity))
			semverParser := parser.New(ctx.Logger, tagger, rules, parser.WithBuildMetadata(buildMetadata), parser.WithProjects(projects))

			for _, branch := range branches {
				semverParser.SetBranch(branch.Name)
				semverParser.SetPrerelease(branch.Prerelease)
				semverParser.SetPrereleaseIdentifier(branch.Name)

				outputs, err := semverParser.Run(context.Background(), repository)
				if err != nil {
					return fmt.Errorf("computing new semver: %w", err)
				}

				for _, output := range outputs {
					semver := output.Semver
					release := output.NewRelease
					commitHash := output.CommitHash
					project := output.Project.Name

					err = ci.GenerateGitHubOutput(semver, branch.Name, ci.WithNewRelease(release), ci.WithTagPrefix(tagPrefix), ci.WithProject(project))
					if err != nil {
						return fmt.Errorf("generating github output: %w", err)
					}

					logEvent := ctx.Logger.Info()
					logEvent.Bool("new-release", release)
					logEvent.Str("version", semver.String())
					logEvent.Str("branch", branch.Name)

					if project != "" {
						logEvent.Str("project", project)

						tagger.SetProjectName(project)
					}

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

						ctx.Logger.Debug().Str("tag", tagger.Format(semver)).Msg("new tag added to repository")

						if remoteMode {
							err = origin.PushTag(tagger.Format(semver))
							if err != nil {
								return fmt.Errorf("pushing tag to remote: %w", err)
							}
						}
					}
				}
			}

			return nil
		},
	}

	releaseCmd.Flags().StringVar(&buildMetadata, "build-metadata", "", "Build metadata (e.g. build number) that will be appended to the SemVer")
	releaseCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next SemVer, do not push any tag")

	return releaseCmd
}

func configureRules(ctx *AppContext) (rule.Rules, error) {
	if !ctx.Viper.IsSet(RulesConfiguration) {
		return rule.Default, nil
	}

	var (
		rulesMarshalled   map[string][]string
		unmarshalledRules rule.Rules
	)

	err := ctx.Viper.UnmarshalKey(RulesConfiguration, &rulesMarshalled)
	if err != nil {
		return unmarshalledRules, fmt.Errorf("unmarshalling %s key: %w", RulesConfiguration, err)
	}

	unmarshalledRules, err = rule.Unmarshall(rulesMarshalled)
	if err != nil {
		return unmarshalledRules, fmt.Errorf("parsing rules: %w", err)
	}

	return unmarshalledRules, nil
}

func configureBranches(ctx *AppContext) ([]branch.Branch, error) {
	if !ctx.Viper.IsSet(BranchesConfiguration) {
		return nil, fmt.Errorf("missing %s key in configuration", BranchesConfiguration)
	}

	var (
		branchesMarshalled   []map[string]any
		unmarshalledBranches []branch.Branch
	)

	err := ctx.Viper.UnmarshalKey(BranchesConfiguration, &branchesMarshalled)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling %s key: %w", BranchesConfiguration, err)
	}

	unmarshalledBranches, err = branch.Unmarshall(branchesMarshalled)
	if err != nil {
		return nil, fmt.Errorf("parsing branches: %w", err)
	}

	return unmarshalledBranches, nil
}

func configureProjects(ctx *AppContext) ([]monorepo.Project, error) {
	if !ctx.Viper.IsSet(MonorepoConfiguration) {
		return nil, nil
	}

	var (
		projectsMarshalled []map[string]string
		projects           []monorepo.Project
	)

	err := ctx.Viper.UnmarshalKey(MonorepoConfiguration, &projectsMarshalled)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling %s key: %w", MonorepoConfiguration, err)
	}

	projects, err = monorepo.Unmarshall(projectsMarshalled)
	if err != nil {
		return nil, fmt.Errorf("parsing projects: %w", err)
	}

	return projects, nil
}

func configureGPGKey(ctx *AppContext) (*openpgp.Entity, error) {
	if armoredKeyPath == "" {
		return nil, nil
	}

	ctx.Logger.Debug().Str("path", armoredKeyPath).Msg("using the following armored key for signing")

	armoredKeyFile, err := os.ReadFile(armoredKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading armored key: %w", err)
	}

	entity, err := gpg.FromArmored(bytes.NewReader(armoredKeyFile))
	if err != nil {
		return nil, fmt.Errorf("loading armored key: %w", err)
	}

	return entity, nil
}

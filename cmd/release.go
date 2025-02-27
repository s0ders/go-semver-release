package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"

	"github.com/s0ders/go-semver-release/v6/internal/appcontext"
	"github.com/s0ders/go-semver-release/v6/internal/branch"
	"github.com/s0ders/go-semver-release/v6/internal/ci"
	"github.com/s0ders/go-semver-release/v6/internal/gpg"
	"github.com/s0ders/go-semver-release/v6/internal/monorepo"
	"github.com/s0ders/go-semver-release/v6/internal/parser"
	"github.com/s0ders/go-semver-release/v6/internal/remote"
	"github.com/s0ders/go-semver-release/v6/internal/rule"
	"github.com/s0ders/go-semver-release/v6/internal/tag"
)

func NewReleaseCmd(ctx *appcontext.AppContext) *cobra.Command {
	releaseCmd := &cobra.Command{
		Use:   "release <REPOSITORY_PATH_OR_URL>",
		Short: "Version a Git repository according the the given configuration",
		Long:  "Tag a Git repository with the new semantic version number if a new release is found on the given release branches and projects if executed in a monorepo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var (
				repository *git.Repository
				origin     *remote.Remote
			)

			entity, err := configureGPGKey(ctx)
			if err != nil {
				return fmt.Errorf("configuring GPG key: %w", err)
			}

			ctx.Rules, err = configureRules(ctx)
			if err != nil {
				return fmt.Errorf("loading rules configuration: %w", err)
			}

			ctx.Branches, err = configureBranches(ctx)
			if err != nil {
				return fmt.Errorf("loading branches configuration: %w", err)
			}

			ctx.Projects, err = configureProjects(ctx)
			if err != nil {
				return fmt.Errorf("loading projects configuration: %w", err)
			}

			origin = remote.New(ctx.RemoteNameFlag, ctx.AccessTokenFlag)

			repository, err = origin.Clone(args[0])
			if err != nil {
				return fmt.Errorf("cloning Git repository: %w", err)
			}

			outputs, err := parser.New(ctx).Run(context.Background(), repository)
			if err != nil {
				return fmt.Errorf("computing new semver: %w", err)
			}

			tagger := tag.NewTagger(ctx.GitNameFlag, ctx.GitEmailFlag, tag.WithTagPrefix(ctx.TagPrefixFlag), tag.WithSignKey(entity))

			for _, output := range outputs {
				semver := output.Semver
				release := output.NewRelease
				commitHash := output.CommitHash
				project := output.Project.Name

				err = ci.GenerateGitHubOutput(semver, output.Branch, ci.WithNewRelease(release), ci.WithTagPrefix(ctx.TagPrefixFlag), ci.WithProject(project))
				if err != nil {
					return fmt.Errorf("generating github output: %w", err)
				}

				logEvent := ctx.Logger.Info()
				logEvent.Bool("new-release", release)
				logEvent.Str("version", semver.String())
				logEvent.Str("branch", output.Branch)

				if project != "" {
					logEvent.Str("project", project)

					tagger.SetProjectName(project)
				}

				switch {
				case !release:
					logEvent.Msg("no new release")
					break
				case release && ctx.DryRunFlag:
					logEvent.Msg("dry-run enabled, next release found")
					break
				default:
					logEvent.Msg("new release found")

					err = tagger.TagRepository(repository, semver, commitHash)
					if err != nil {
						return fmt.Errorf("tagging repository: %w", err)
					}

					ctx.Logger.Debug().Str("tag", tagger.Format(semver)).Msg("new tag added to repository")

					err = origin.PushTag(tagger.Format(semver))
					if err != nil {
						return fmt.Errorf("pushing tag to remote: %w", err)
					}
				}
			}

			return nil
		},
	}

	return releaseCmd
}

func configureRules(ctx *appcontext.AppContext) (rule.Rules, error) {
	flag := ctx.RulesFlag

	if flag.String() == "{}" {
		return rule.Default, nil
	}

	rulesJSON := map[string][]string(flag)

	unmarshalledRules, err := rule.Unmarshall(rulesJSON)
	if err != nil {
		return unmarshalledRules, fmt.Errorf("parsing rules configuration: %w", err)
	}

	return unmarshalledRules, nil
}

func configureBranches(ctx *appcontext.AppContext) ([]branch.Branch, error) {
	branchesJSON := []map[string]any(ctx.BranchesFlag)

	unmarshalledBranches, err := branch.Unmarshall(branchesJSON)
	if err != nil {
		return nil, fmt.Errorf("parsing branches configuration: %w", err)
	}

	return unmarshalledBranches, nil
}

func configureProjects(ctx *appcontext.AppContext) ([]monorepo.Project, error) {
	flag := ctx.MonorepositoryFlag

	if flag.String() == "[]" {
		return nil, nil
	}

	monorepoJSON := []map[string]string(flag)

	projects, err := monorepo.Unmarshall(monorepoJSON)
	if err != nil {
		return nil, fmt.Errorf("parsing monorepository projects configuration: %w", err)
	}

	return projects, nil
}

func configureGPGKey(ctx *appcontext.AppContext) (*openpgp.Entity, error) {
	flag := ctx.GPGKeyPathFlag

	if flag == "" {
		return nil, nil
	}

	ctx.Logger.Debug().Str("path", ctx.GPGKeyPathFlag).Msg("using the following armored key for signing")

	armoredKeyFile, err := os.ReadFile(ctx.GPGKeyPathFlag)
	if err != nil {
		return nil, fmt.Errorf("reading armored key: %w", err)
	}

	entity, err := gpg.FromArmored(bytes.NewReader(armoredKeyFile))
	if err != nil {
		return nil, fmt.Errorf("loading armored key: %w", err)
	}

	return entity, nil
}

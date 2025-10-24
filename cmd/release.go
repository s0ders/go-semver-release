package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"

	"github.com/s0ders/go-semver-release/v6/internal/appcontext"
	"github.com/s0ders/go-semver-release/v6/internal/ci"
	"github.com/s0ders/go-semver-release/v6/internal/gpg"
	"github.com/s0ders/go-semver-release/v6/internal/parser"
	"github.com/s0ders/go-semver-release/v6/internal/remote"
	"github.com/s0ders/go-semver-release/v6/internal/rule"
	"github.com/s0ders/go-semver-release/v6/internal/tag"
)

const (
	MessageDryRun       string = "dry-run enabled, next release found"
	MessageNewRelease   string = "new release found"
	MessageNoNewRelease string = "no new release"
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

			if ctx.RulesCfg.String() == "{}" {
				ctx.Logger.Debug().Msg("no rules configuration provided, using default release rules")

				b, err := json.Marshal(rule.Default)
				if err != nil {
					return fmt.Errorf("marshalling default rules: %w", err)
				}

				if err = ctx.RulesCfg.Set(string(b)); err != nil {
					return fmt.Errorf("setting default rules flag: %w", err)
				}
			}

			origin = remote.New(ctx.RemoteName, ctx.AccessToken)

			repository, err = origin.Clone(args[0])
			if err != nil {
				return fmt.Errorf("cloning Git repository: %w", err)
			}

			outputs, err := parser.New(ctx).Run(repository)
			if err != nil {
				return fmt.Errorf("computing new semver: %w", err)
			}

			tagger := tag.NewTagger(ctx.GitName, ctx.GitEmail, tag.WithTagPrefix(ctx.TagPrefix), tag.WithSignKey(entity))

			for _, output := range outputs {
				semver := output.Semver
				release := output.NewRelease
				commitHash := output.CommitHash
				project := output.Project.Name

				err = ci.GenerateGitHubOutput(semver, output.Branch, ci.WithNewRelease(release), ci.WithTagPrefix(ctx.TagPrefix), ci.WithProject(project))
				if err != nil {
					return fmt.Errorf("generating GitHub output: %w", err)
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
					logEvent.Msg(MessageNoNewRelease)
				case release && ctx.DryRun:
					logEvent.Msg(MessageDryRun)
				default:
					logEvent.Msg(MessageNewRelease)

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

func configureGPGKey(ctx *appcontext.AppContext) (*openpgp.Entity, error) {
	flag := ctx.GPGKeyPath

	if flag == "" {
		return nil, nil
	}

	ctx.Logger.Debug().Str("path", ctx.GPGKeyPath).Msg("using the following armored key for signing")

	armoredKeyFile, err := os.ReadFile(ctx.GPGKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading armored key: %w", err)
	}

	entity, err := gpg.FromArmored(bytes.NewReader(armoredKeyFile))
	if err != nil {
		return nil, fmt.Errorf("loading armored key: %w", err)
	}

	return entity, nil
}

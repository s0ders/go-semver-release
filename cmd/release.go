package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"

	"github.com/s0ders/go-semver-release/v8/internal/appcontext"
	"github.com/s0ders/go-semver-release/v8/internal/ci"
	"github.com/s0ders/go-semver-release/v8/internal/gpg"
	"github.com/s0ders/go-semver-release/v8/internal/parser"
	"github.com/s0ders/go-semver-release/v8/internal/remote"
	"github.com/s0ders/go-semver-release/v8/internal/rule"
	"github.com/s0ders/go-semver-release/v8/internal/tag"
)

const (
	defaultConfigFile = ".semver"
	configFileFormat  = "yaml"
)

const (
	AccessTokenConfiguration     = "access-token"
	BranchesConfiguration        = "branches"
	BuildMetadataConfiguration   = "build-metadata"
	DryRunConfiguration          = "dry-run"
	GitEmailConfiguration        = "git-email"
	GitNameConfiguration         = "git-name"
	GPGPathConfiguration         = "gpg-key-path"
	LightweightTagsConfiguration = "lightweight-tags"
	MonorepoConfiguration        = "monorepo"
	RemoteNameConfiguration      = "remote-name"
	RulesConfiguration           = "rules"
	TagPrefixConfiguration       = "tag-prefix"
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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd, ctx)
		},
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

			repository, err = origin.Open(args[0])
			if err != nil {
				return fmt.Errorf("opening Git repository: %w", err)
			}

			defer func() {
				if closeErr := origin.Close(); closeErr != nil {
					err = errors.Join(err, closeErr)
				}
			}()

			outputs, err := parser.New(ctx).Run(repository)
			if err != nil {
				return fmt.Errorf("computing new semver: %w", err)
			}

			tagger := tag.NewTagger(
				ctx.GitName,
				ctx.GitEmail,
				tag.WithTagPrefix(ctx.TagPrefix),
				tag.WithSignKey(entity),
				tag.WithLightweight(ctx.LightweightTags),
			)

			jsonOutput := ci.NewJSONOutput()

			// New tags are pushed all at once after the loop, in a single push.
			var newTags []string

			for _, output := range outputs {
				semver := output.Semver
				release := output.NewRelease
				commitHash := output.CommitHash
				project := output.Project.Name

				err = ci.GenerateGitHubOutput(semver, output.Branch, ci.WithNewRelease(release), ci.WithTagPrefix(ctx.TagPrefix), ci.WithProject(project))
				if err != nil {
					return fmt.Errorf("generating GitHub output: %w", err)
				}

				// Determine the message
				var message string
				switch {
				case !release:
					message = MessageNoNewRelease
				case release && ctx.DryRun:
					message = MessageDryRun
				default:
					message = MessageNewRelease
				}

				jsonOutput.AddRelease(release, semver.String(), output.Branch, project, message)

				// Tag and push if new release and not dry-run
				if release && !ctx.DryRun {
					if project != "" {
						tagger.SetProjectName(project)
					}

					err = tagger.TagRepository(repository, semver, commitHash)
					if err != nil {
						// No-op if the tag was somehow already there
						if errors.Is(err, tag.ErrTagAlreadyExists) {
							continue
						}

						return fmt.Errorf("tagging repository: %w", err)
					}

					tagName := tagger.Format(semver)
					ctx.Logger.Debug().Str("tag", tagName).Msg("new tag added to repository")

					newTags = append(newTags, tagName)
				}
			}

			if err = origin.PushTags(newTags...); err != nil {
				return fmt.Errorf("pushing tag to remote: %w", err)
			}

			if err := jsonOutput.Write(cmd.OutOrStdout()); err != nil {
				return fmt.Errorf("writing JSON output: %w", err)
			}

			return nil
		},
	}

	// Release-specific flags
	releaseCmd.Flags().StringVar(&ctx.AccessToken, AccessTokenConfiguration, "", "Access token used to push tag to Git remote")
	releaseCmd.Flags().VarP(&ctx.BranchesCfg, BranchesConfiguration, "b", "An array of branches configuration such as [{\"name\": \"main\"}, {\"name\": \"rc\", \"prerelease\": true}]")
	releaseCmd.Flags().StringVar(&ctx.BuildMetadata, BuildMetadataConfiguration, "", "Build metadata that will be appended to the SemVer")
	releaseCmd.Flags().StringVar(&ctx.CfgFile, "config", "", "Configuration file path (default \"./"+defaultConfigFile+"."+configFileFormat+"\")")
	releaseCmd.Flags().BoolVarP(&ctx.DryRun, DryRunConfiguration, "d", false, "Only compute the next SemVer, do not push any tag")
	releaseCmd.Flags().StringVar(&ctx.GitEmail, GitEmailConfiguration, "go-semver@release.ci", "Email used in semantic version tags")
	releaseCmd.Flags().StringVar(&ctx.GitName, GitNameConfiguration, "Go Semver Release", "Name used in semantic version Git tags")
	releaseCmd.Flags().StringVar(&ctx.GPGKeyPath, GPGPathConfiguration, "", "Path to an armored GPG key used to sign produced Git tags")
	releaseCmd.Flags().BoolVar(&ctx.LightweightTags, LightweightTagsConfiguration, false, "Create lightweight tags instead of annotated tags")
	releaseCmd.Flags().Var(&ctx.MonorepositoryCfg, MonorepoConfiguration, "An array of monorepository configuration such as [{\"name\": \"foo\", \"path\": \"./foo/\"}]")
	releaseCmd.Flags().StringVar(&ctx.RemoteName, RemoteNameConfiguration, "origin", "Name of the Git repository remote")
	releaseCmd.Flags().Var(&ctx.RulesCfg, RulesConfiguration, "A hashmap of array such as {\"minor\": [\"feat\"], \"patch\": [\"fix\", \"perf\"]} ]")
	releaseCmd.Flags().StringVar(&ctx.TagPrefix, TagPrefixConfiguration, "v", "Prefix added to the version tag name")

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

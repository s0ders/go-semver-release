package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/s0ders/go-semver-release/v7/internal/appcontext"
	"github.com/s0ders/go-semver-release/v7/internal/branch"
	"github.com/s0ders/go-semver-release/v7/internal/ci"
	"github.com/s0ders/go-semver-release/v7/internal/gpg"
	"github.com/s0ders/go-semver-release/v7/internal/monorepo"
	"github.com/s0ders/go-semver-release/v7/internal/parser"
	"github.com/s0ders/go-semver-release/v7/internal/remote"
	"github.com/s0ders/go-semver-release/v7/internal/rule"
	"github.com/s0ders/go-semver-release/v7/internal/tag"
)

const (
	defaultConfigFile = ".semver"
	configFileFormat  = "yaml"
)

const (
	AccessTokenConfiguration   = "access-token"
	BranchesConfiguration      = "branches"
	BuildMetadataConfiguration = "build-metadata"
	DryRunConfiguration        = "dry-run"
	GitEmailConfiguration      = "git-email"
	GitNameConfiguration       = "git-name"
	GPGPathConfiguration       = "gpg-key-path"
	MonorepoConfiguration      = "monorepo"
	RemoteNameConfiguration    = "remote-name"
	RulesConfiguration         = "rules"
	TagPrefixConfiguration     = "tag-prefix"
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

	// Release-specific flags
	releaseCmd.Flags().StringVar(&ctx.AccessToken, AccessTokenConfiguration, "", "Access token used to push tag to Git remote")
	releaseCmd.Flags().VarP(&ctx.BranchesCfg, BranchesConfiguration, "b", "An array of branches configuration such as [{\"name\": \"main\"}, {\"name\": \"rc\", \"prerelease\": true}]")
	releaseCmd.Flags().StringVar(&ctx.BuildMetadata, BuildMetadataConfiguration, "", "Build metadata that will be appended to the SemVer")
	releaseCmd.Flags().StringVar(&ctx.CfgFile, "config", "", "Configuration file path (default \"./"+defaultConfigFile+"."+configFileFormat+"\")")
	releaseCmd.Flags().BoolVarP(&ctx.DryRun, DryRunConfiguration, "d", false, "Only compute the next SemVer, do not push any tag")
	releaseCmd.Flags().StringVar(&ctx.GitEmail, GitEmailConfiguration, "go-semver@release.ci", "Email used in semantic version tags")
	releaseCmd.Flags().StringVar(&ctx.GitName, GitNameConfiguration, "Go Semver Release", "Name used in semantic version Git tags")
	releaseCmd.Flags().StringVar(&ctx.GPGKeyPath, GPGPathConfiguration, "", "Path to an armored GPG key used to sign produced Git tags")
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

// initializeConfig manages how the configuration variables of the application are initialized.
// It loads configuration with the following order of precedence:
// (1) command line flags, (2) environment variables, (3) viper configuration file.
func initializeConfig(cmd *cobra.Command, ctx *appcontext.AppContext) error {
	if ctx.CfgFile != "" {
		ctx.Viper.SetConfigFile(ctx.CfgFile)
	} else {
		ctx.Viper.AddConfigPath(".")
		ctx.Viper.SetConfigType(configFileFormat)
		ctx.Viper.SetConfigName(defaultConfigFile)
	}

	absCfgPath, err := filepath.Abs(ctx.CfgFile)
	if err != nil {
		return fmt.Errorf("getting configuration file absolute path: %w", err)
	}
	ctx.Logger.Debug().Str("path", absCfgPath).Msg("using the following configuration file")

	ctx.Viper.SetEnvPrefix("GO_SEMVER_RELEASE")
	ctx.Viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	ctx.Viper.AutomaticEnv()

	if err = ctx.Viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError

		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}

	if err = bindFlags(cmd, ctx.Viper); err != nil {
		return err
	}

	return nil
}

// bindFlags binds Viper configuration value to their corresponding Cobra flag if, for a given configuration value,
// the flag has not been set and the Viper configuration has been.
func bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	var err error

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err != nil {
			return
		}

		configName := f.Name

		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)

			switch flagType := f.Value.(type) {
			case *branch.Flag, *rule.Flag, *monorepo.Flag:
				jsonStr, jsonErr := json.Marshal(val)
				if jsonErr != nil {
					err = fmt.Errorf("marshaling %q value: %w", configName, jsonErr)
				}

				err = flagType.Set(string(jsonStr))
			default:
				err = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			}

			f.Changed = true
		}
	})

	return err
}

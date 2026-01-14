package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/s0ders/go-semver-release/v7/internal/appcontext"
	"github.com/s0ders/go-semver-release/v7/internal/branch"
	"github.com/s0ders/go-semver-release/v7/internal/monorepo"
	"github.com/s0ders/go-semver-release/v7/internal/rule"
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

func NewRootCommand(ctx *appcontext.AppContext) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "go-semver-release",
		Short: "go-semver-release - Automate semantic versioning of Git repositories",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx.Logger = zerolog.New(cmd.OutOrStdout()).Level(zerolog.InfoLevel)

			if ctx.Verbose {
				ctx.Logger = ctx.Logger.Level(zerolog.DebugLevel)
			}

			return initializeConfig(cmd, ctx)
		},
		TraverseChildren: true,
	}

	rootCmd.PersistentFlags().StringVar(&ctx.AccessToken, AccessTokenConfiguration, "", "Access token used to push tag to Git remote")
	rootCmd.PersistentFlags().VarP(&ctx.BranchesCfg, BranchesConfiguration, "b", "An array of branches configuration such as [{\"name\": \"main\"}, {\"name\": \"rc\", \"prerelease\": true}]")
	rootCmd.PersistentFlags().StringVar(&ctx.BuildMetadata, BuildMetadataConfiguration, "", "Build metadata that will be appended to the SemVer")
	rootCmd.PersistentFlags().StringVar(&ctx.CfgFile, "config", "", "Configuration file path (default \"./"+defaultConfigFile+"."+configFileFormat+"\")")
	rootCmd.PersistentFlags().BoolVarP(&ctx.DryRun, DryRunConfiguration, "d", false, "Only compute the next SemVer, do not push any tag")
	rootCmd.PersistentFlags().StringVar(&ctx.GitEmail, GitEmailConfiguration, "go-semver@release.ci", "Email used in semantic version tags")
	rootCmd.PersistentFlags().StringVar(&ctx.GitName, GitNameConfiguration, "Go Semver Release", "Name used in semantic version Git tags")
	rootCmd.PersistentFlags().StringVar(&ctx.GPGKeyPath, GPGPathConfiguration, "", "Path to an armored GPG key used to sign produced Git tags")
	rootCmd.PersistentFlags().Var(&ctx.MonorepositoryCfg, MonorepoConfiguration, "An array of monorepository configuration such as [{\"name\": \"foo\", \"path\": \"./foo/\"}]")
	rootCmd.PersistentFlags().StringVar(&ctx.RemoteName, RemoteNameConfiguration, "origin", "Name of the Git repository remote")
	rootCmd.PersistentFlags().Var(&ctx.RulesCfg, RulesConfiguration, "A hashmap of array such as {\"minor\": [\"feat\"], \"patch\": [\"fix\", \"perf\"]} ]")
	rootCmd.PersistentFlags().StringVar(&ctx.TagPrefix, TagPrefixConfiguration, "v", "Prefix added to the version tag name")
	rootCmd.PersistentFlags().BoolVarP(&ctx.Verbose, "verbose", "v", false, "Verbose output")

	releaseCmd := NewReleaseCmd(ctx)
	versionCmd := NewVersionCmd()

	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}

// initializeConfig manages how the configuration variables of the application are initialized.
// It loads configuration with the following order of precedence:
// (1) command line flags, (2) environment variables, (3) viper configuration file.
// The key point to understand how configuration management is handled throughout this application is that everything
// ends up being mapped to the variable inside appcontext.AppContext that are either populated by Cobra using CLI flag
// values, or by Viper using configuration file or environment variables. Viper is only used to bind flags and
// configuration file values.
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

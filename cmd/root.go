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

	"github.com/s0ders/go-semver-release/v6/internal/appcontext"
	"github.com/s0ders/go-semver-release/v6/internal/branch"
	"github.com/s0ders/go-semver-release/v6/internal/monorepo"
	"github.com/s0ders/go-semver-release/v6/internal/rule"
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

func NewAppContext() *appcontext.AppContext {
	return &appcontext.AppContext{
		Viper: viper.New(),
	}
}

func NewRootCommand(ctx *appcontext.AppContext) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "go-semver-release",
		Short: "go-semver-release - Automate semantic versioning of Git repositories",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx.Logger = zerolog.New(cmd.OutOrStdout()).Level(zerolog.InfoLevel)

			if ctx.VerboseFlag {
				ctx.Logger = ctx.Logger.Level(zerolog.DebugLevel)
			}

			return initializeConfig(cmd, ctx)
		},
		TraverseChildren: true,
	}

	rootCmd.PersistentFlags().StringVar(&ctx.AccessTokenFlag, AccessTokenConfiguration, "", "Access token used to push tag to Git remote")
	rootCmd.PersistentFlags().VarP(&ctx.BranchesFlag, BranchesConfiguration, "b", "An array of branches such as [{\"name\": \"main\"}, {\"name\": \"rc\", \"prerelease\": true}]")
	rootCmd.PersistentFlags().StringVar(&ctx.BuildMetadataFlag, BuildMetadataConfiguration, "", "Build metadata (e.g. build number) that will be appended to the SemVer")
	rootCmd.PersistentFlags().StringVar(&ctx.CfgFileFlag, "config", "", "Configuration file path (default \"./"+defaultConfigFile+"."+configFileFormat+"\")")
	rootCmd.PersistentFlags().BoolVarP(&ctx.DryRunFlag, DryRunConfiguration, "d", false, "Only compute the next SemVer, do not push any tag")
	rootCmd.PersistentFlags().StringVar(&ctx.GitEmailFlag, GitEmailConfiguration, "go-semver@release.ci", "Email used in semantic version tags")
	rootCmd.PersistentFlags().StringVar(&ctx.GitNameFlag, GitNameConfiguration, "Go Semver Release", "Name used in semantic version tags")
	rootCmd.PersistentFlags().StringVar(&ctx.GPGKeyPathFlag, GPGPathConfiguration, "", "Path to an armored GPG key used to sign produced tags")
	rootCmd.PersistentFlags().Var(&ctx.MonorepositoryFlag, MonorepoConfiguration, "An array of branches such as [{\"name\": \"foo\", \"path\": \"./foo/\"}]")
	rootCmd.PersistentFlags().StringVar(&ctx.RemoteNameFlag, RemoteNameConfiguration, "origin", "Name of the Git repository remote")
	rootCmd.PersistentFlags().Var(&ctx.RulesFlag, RulesConfiguration, "An hashmap of array such as {\"minor\": [\"feat\"], \"patch\": [\"fix\", \"perf\"]} ]")
	rootCmd.PersistentFlags().StringVar(&ctx.TagPrefixFlag, TagPrefixConfiguration, "v", "Prefix added to the version tag name")
	rootCmd.PersistentFlags().BoolVarP(&ctx.VerboseFlag, "verbose", "v", false, "Verbose output")

	releaseCmd := NewReleaseCmd(ctx)
	versionCmd := NewVersionCmd()

	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}

func initializeConfig(cmd *cobra.Command, ctx *appcontext.AppContext) error {
	if ctx.CfgFileFlag != "" {
		ctx.Viper.SetConfigFile(ctx.CfgFileFlag)
	} else {
		ctx.Viper.AddConfigPath(".")
		ctx.Viper.SetConfigType(configFileFormat)
		ctx.Viper.SetConfigName(defaultConfigFile)
	}

	absCfgPath, err := filepath.Abs(ctx.CfgFileFlag)
	if err != nil {
		return fmt.Errorf("getting configuration file absolute path: %w", err)
	}
	ctx.Logger.Debug().Str("path", absCfgPath).Msg("using the following configuration file")

	ctx.Viper.SetEnvPrefix("GO_SEMVER_RELEASE")
	ctx.Viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	ctx.Viper.AutomaticEnv()

	if err := ctx.Viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError

		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}

	if err := bindFlags(cmd, ctx.Viper); err != nil {
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

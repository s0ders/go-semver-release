package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/s0ders/go-semver-release/v5/internal/branch"
	"github.com/s0ders/go-semver-release/v5/internal/monorepo"
	"github.com/s0ders/go-semver-release/v5/internal/rule"
)

const (
	defaultConfigFile = ".semver"
	configFileFormat  = "yaml"

	MonorepoConfiguration = "monorepo"
	RulesConfiguration    = "rules"
	BranchesConfiguration = "branches"
)

var (
	cfgFile        string
	gitName        string
	gitEmail       string
	tagPrefix      string
	accessToken    string
	remoteName     string
	armoredKeyPath string
	verbose        bool
	remoteMode     bool
	branches       branch.Flag
	monorepository monorepo.Flag
	rules          rule.Flag
)

var viperInstance = viper.New()

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Configuration file path (default is ./"+defaultConfigFile+""+configFileFormat+")")
	rootCmd.PersistentFlags().StringVar(&gitName, "git-name", "Go Semver Release", "Name used in semantic version tags")
	rootCmd.PersistentFlags().StringVar(&gitEmail, "git-email", "go-semver@release.ci", "Email used in semantic version tags")
	rootCmd.PersistentFlags().StringVar(&tagPrefix, "tag-prefix", "v", "Prefix added to the version tag name")
	rootCmd.PersistentFlags().StringVar(&accessToken, "access-token", "", "Access token used to push tag to Git remote")
	rootCmd.PersistentFlags().StringVar(&remoteName, "remote-name", "origin", "Name of the Git repository remote")
	rootCmd.PersistentFlags().StringVar(&armoredKeyPath, "gpg-key-path", "", "Path to an armored GPG key used to sign produced tags")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&remoteMode, "remote", false, "Version a remote repository, a token is required")
	rootCmd.PersistentFlags().Var(&branches, BranchesConfiguration, "An array of branches such as [{\"name\": \"main\"}, {\"name\": \"rc\", \"prerelease\": true}]")
	rootCmd.PersistentFlags().Var(&monorepository, MonorepoConfiguration, "An array of branches such as [{\"name\": \"foo\", \"path\": \"./foo/\"}]")
	rootCmd.PersistentFlags().Var(&rules, RulesConfiguration, "An hashmap of array such as {\"minor\": [\"feat\"], \"patch\": [\"fix\", \"perf\"]} ]")

	rootCmd.MarkFlagsRequiredTogether("remote", "remote-name", "access-token")
}

var rootCmd = &cobra.Command{
	Use:   "go-semver-release",
	Short: "go-semver-release - CLI to automate semantic versioning of Git repositories",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func initializeConfig(cmd *cobra.Command) error {
	if cfgFile != "" {
		viperInstance.SetConfigFile(cfgFile)
	} else {
		viperInstance.AddConfigPath(".")
		viperInstance.SetConfigType(configFileFormat)
		viperInstance.SetConfigName(defaultConfigFile)
	}

	if err := viperInstance.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError

		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}

	if err := bindFlags(cmd, viperInstance); err != nil {
		return err
	}

	return nil
}

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
		}
	})

	return err
}

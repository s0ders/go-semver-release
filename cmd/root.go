package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultConfigFile = ".semver"
	configFileFormat  = "yaml"
	envPrefix         = "GO_SEMVER_RELEASE"
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
	monorepository bool
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
	rootCmd.PersistentFlags().BoolVar(&remoteMode, "remote", false, "Version a remote repository, a token is required")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&monorepository, "monorepo", false, "Operating in monorepo mode versioning multiple projects separately")

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

	viperInstance.SetEnvPrefix(envPrefix)
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viperInstance.AutomaticEnv()

	if err := bindFlags(cmd, viperInstance); err != nil {
		return err
	}

	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	var err error

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name

		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			err = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				return
			}
		}
	})

	return err
}

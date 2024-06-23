package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultConfigFile = ".semver"
	configFileFormat  = "yaml"
)

var (
	cfgFile   string
	gitName   string
	gitEmail  string
	tagPrefix string
	verbose   bool
)

var viperInstance = viper.New()

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path (default is ./"+defaultConfigFile+""+configFileFormat+")")
	rootCmd.PersistentFlags().StringVar(&gitName, "git-name", "Go Semver Release", "Name used in semantic version tags")
	rootCmd.PersistentFlags().StringVar(&gitEmail, "git-email", "go-semver@release.ci", "Email used in semantic version tags")
	rootCmd.PersistentFlags().StringVar(&tagPrefix, "tag-prefix", "v", "Prefix added to the version tag name")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
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

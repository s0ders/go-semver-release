package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	verbose  bool
	gitName  string
	gitEmail string
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path (default is ./.semver.json)")
	rootCmd.PersistentFlags().StringVar(&gitName, "git-name", "Go Semver Release", "Name used in semantic version tags")
	rootCmd.PersistentFlags().StringVar(&gitEmail, "git-email", "go-semver@release.ci", "Email used in semantic version tags")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose ci")

	cobra.CheckErr(viper.BindPFlag("git-name", rootCmd.PersistentFlags().Lookup("git-name")))
	cobra.CheckErr(viper.BindPFlag("git-email", rootCmd.PersistentFlags().Lookup("git-email")))
}

var rootCmd = &cobra.Command{
	Use:   "go-semver-release",
	Short: "go-semver-release - CLI to automate semantic versioning of git repositories",
}

func Execute() error {
	return rootCmd.Execute()
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		wd, err := os.Getwd()
		cobra.CheckErr(err)

		viper.AddConfigPath(wd)
		viper.SetConfigType("json")
		viper.SetConfigName(".semver")
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

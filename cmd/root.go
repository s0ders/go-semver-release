package cmd

import (
	"github.com/spf13/cobra"
)

var verbose bool

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose ci")
}

var rootCmd = &cobra.Command{
	Use:   "go-semver-release",
	Short: "go-semver-release - CLI to automate semantic versioning of git repositories",
	Long:  "go-semver-release - open source CLI to automate semantic versioning of git repositories using a formatted commit history",
}

func Execute() error {
	return rootCmd.Execute()
}

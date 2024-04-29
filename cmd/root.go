package cmd

import (
	"github.com/spf13/cobra"
)

var (
	cliVersion string
	rootCmd    = &cobra.Command{
		Use:     "go-semver-release",
		Version: cliVersion,
		Short:   "go-semver-release - CLI to automate semantic versioning of git repositories",
		Long:    "go-semver-release - open source CLI to automate semantic versioning of git repositories using a formatted commit history",
	}
)

func Execute() error {
	return rootCmd.Execute()
}

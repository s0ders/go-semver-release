package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version     string
	buildNumber string
	commitHash  string
)

func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display CLI current version",
		Long:  "Display CLI current version and the associated build number and commit hash",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\nBuild: %s\nCommit: %s\n", version, buildNumber, commitHash)

			return nil
		},
	}

	return versionCmd
}

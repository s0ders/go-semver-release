package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdVersion      string
	buildNumber     string
	buildCommitHash string
)

func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display CLI current version",
		Long:  "Display CLI current version, the associated build number and commit hash",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\nBuild: %s\nCommit: %s\n", cmdVersion, buildNumber, buildCommitHash)

			return nil
		},
	}

	return versionCmd
}

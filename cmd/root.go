package cmd

import (
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/s0ders/go-semver-release/v8/internal/appcontext"
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

			return nil
		},
		TraverseChildren: true,
	}

	rootCmd.PersistentFlags().BoolVarP(&ctx.Verbose, "verbose", "v", false, "Verbose output")

	releaseCmd := NewReleaseCmd(ctx)
	validateCmd := NewValidateCmd()
	versionCmd := NewVersionCmd()

	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}

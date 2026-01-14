package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// These can be set via ldflags for CI builds, but are optional.
// If not set, values are read from debug.ReadBuildInfo().
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
		RunE: func(cmd *cobra.Command, args []string) error {
			version, commit, modified := getVersionInfo()

			versionStr := version
			if modified {
				versionStr += " (modified)"
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Version: %s\n", versionStr)

			if buildNumber != "" {
				fmt.Fprintf(out, "Build: %s\n", buildNumber)
			}

			fmt.Fprintf(out, "Commit: %s\n", commit)

			return nil
		},
	}

	return versionCmd
}

// getVersionInfo returns version, commit hash, and whether the build has uncommitted changes.
// It prefers values set via ldflags, falling back to debug.ReadBuildInfo().
func getVersionInfo() (version, commit string, modified bool) {
	version = "unknown"
	commit = "unknown"

	// Use ldflags values if set
	if cmdVersion != "" {
		version = cmdVersion
	}
	if buildCommitHash != "" {
		commit = buildCommitHash
	}

	// If ldflags not set, try to read from build info
	if cmdVersion == "" || buildCommitHash == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if cmdVersion == "" && info.Main.Version != "" {
				version = info.Main.Version
			}

			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if buildCommitHash == "" {
						commit = setting.Value
						if len(commit) > 12 {
							commit = commit[:12]
						}
					}
				case "vcs.modified":
					modified = setting.Value == "true"
				}
			}
		}
	}

	return version, commit, modified
}

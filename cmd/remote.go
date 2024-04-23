package cmd

import (
	"log/slog"
	"os"

	"github.com/s0ders/go-semver-release/internal/output"

	"github.com/s0ders/go-semver-release/internal/commitanalyzer"
	"github.com/s0ders/go-semver-release/internal/releaserules"
	"github.com/s0ders/go-semver-release/internal/tagger"

	"github.com/s0ders/go-semver-release/internal/cloner"
	"github.com/spf13/cobra"
)

var (
	rulesPath     string
	gitURL        string
	token         string
	tagPrefix     string
	releaseBranch string
	dryRun        bool
	verbose       bool
)

func init() {
	remoteCmd.Flags().StringVarP(&rulesPath, "rules-path", "rules", "", "Path to the JSON containing the release rules")
	remoteCmd.Flags().StringVarP(&gitURL, "git-url", "url", "", "URL of the git repository to version")
	remoteCmd.Flags().StringVar(&token, "token", "", "Secret token to access the git repository")
	remoteCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "tag", "v", "Prefix added to the version tag name")
	remoteCmd.Flags().StringVarP(&releaseBranch, "release-branch", "branch", "main", "Branch to fetch commits from")
	remoteCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Don't actually release changes")
	remoteCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	remoteCmd.MarkFlagRequired("git-url")
	remoteCmd.MarkFlagRequired("token")

	rootCmd.AddCommand(remoteCmd)
}

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Attempts to spoof a network interface MAC address",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		repository, path, err := cloner.New(logger).Clone(gitURL, releaseBranch, token)
		if err != nil {
			return err
		}

		defer func(path string) {
			err = os.RemoveAll(path)
			if err != nil {
				return
			}
		}(path)

		rules, err := releaserules.New(logger).Read(rulesPath).Parse()
		if err != nil {
			return err
		}

		semver, release, err := commitanalyzer.New(logger, rules, verbose).ComputeNewSemver(repository)
		if err != nil {
			return err
		}

		err = output.NewOutput(logger).Generate(tagPrefix, semver, release)
		if err != nil {
			return err
		}

		if !release {
			logger.Info("no new release", "current-version", semver)
		}

		if release && dryRun {
			logger.Info("new release found, dry-run is enabled", "next-version", semver)
		}

		err = tagger.NewTagger(tagPrefix).PushTagToRemote(repository, token, semver)
		if err != nil {
			return err
		}

		return nil
	},
}

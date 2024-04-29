package cmd

import (
	"log/slog"
	"os"

	"github.com/s0ders/go-semver-release/v2/internal/ci"
	"github.com/s0ders/go-semver-release/v2/internal/parser"
	"github.com/s0ders/go-semver-release/v2/internal/remote"
	"github.com/s0ders/go-semver-release/v2/internal/rules"
	"github.com/s0ders/go-semver-release/v2/internal/tagger"
	"github.com/spf13/cobra"
)

var (
	rulesPath     string
	token         string
	tagPrefix     string
	releaseBranch string
	remoteName    string
	dryRun        bool
	verbose       bool
	jsonOutput    bool
)

func init() {
	remoteCmd.Flags().StringVarP(&rulesPath, "rules-path", "r", "", "Path to the JSON or YAML file containing the release rules")
	remoteCmd.Flags().StringVarP(&token, "token", "t", "", "Secret token to access the git repository")
	remoteCmd.Flags().StringVarP(&tagPrefix, "tag-prefix", "p", "v", "Prefix added to the version tag name")
	remoteCmd.Flags().StringVarP(&releaseBranch, "release-branch", "b", "main", "Branch to fetch commits from")
	remoteCmd.Flags().StringVar(&remoteName, "remote-name", "origin", "Name of the remote to push to")
	remoteCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Only compute the next semver, do not push any tag")
	remoteCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose ci")
	remoteCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "JSON formatted output")

	remoteCmd.MarkFlagRequired("token")

	rootCmd.AddCommand(remoteCmd)
}

var remoteCmd = &cobra.Command{
	Use:   "remote <GIT_URL>",
	Short: "Version a remote repository and push the semver tag back to the remote",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var logHandler slog.Handler

		if jsonOutput {
			logHandler = slog.NewJSONHandler(cmd.OutOrStdout(), nil)
		} else {
			logHandler = slog.NewTextHandler(cmd.OutOrStdout(), nil)
		}

		logger := slog.New(logHandler)

		remote := remote.New(logger, token, remoteName, verbose)
		if err != nil {
			return err
		}

		gitURL := args[0]

		repository, repositoryPath, err := remote.Clone(gitURL, releaseBranch)

		defer func(path string) {
			err = os.RemoveAll(path)
			if err != nil {
				return
			}
		}(repositoryPath)

		rulesReader, err := rules.New(logger).Read(rulesPath)
		if err != nil {
			return err
		}

		rules, err := rulesReader.Parse()
		if err != nil {
			return err
		}

		semver, release, err := parser.New(logger, rules, verbose).ComputeNewSemver(repository)
		if err != nil {
			return err
		}

		err = ci.New(logger).GenerateGitHub(tagPrefix, semver, release)
		if err != nil {
			return err
		}

		switch {
		case !release:
			logger.Info("no new release", "current-version", semver.NormalVersion())
			return nil
		case release && dryRun:
			logger.Info("new release found, dry-run is enabled", "next-version", semver)
			return nil
		default:
			err = tagger.New(logger, tagPrefix, verbose).AddTagToRepository(repository, semver)
			if err != nil {
				return err
			}

			err = remote.PushTagToRemote(repository, semver)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

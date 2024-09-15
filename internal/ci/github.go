// Package ci provides function to generate output for CI/CD tools.
package ci

import (
	"fmt"
	"os"
	"strings"

	"github.com/s0ders/go-semver-release/v4/internal/semver"
)

type GitHubOutput struct {
	Semver     *semver.Semver
	Branch     string
	TagPrefix  string
	NewRelease bool
}

func (g GitHubOutput) String() string {
	branch := strings.ToUpper(g.Branch)

	versionKey := branch + "_SEMVER"
	releaseKey := branch + "_NEW_RELEASE"

	str := "\n"
	str += fmt.Sprintf("%s=%s\n", versionKey, g.TagPrefix+g.Semver.String())
	str += fmt.Sprintf("%s=%t\n", releaseKey, g.NewRelease)

	return str
}

type OptionFunc func(*GitHubOutput)

func WithNewRelease(b bool) OptionFunc {
	return func(o *GitHubOutput) {
		o.NewRelease = b
	}
}

func WithTagPrefix(tagPrefix string) OptionFunc {
	return func(o *GitHubOutput) {
		o.TagPrefix = tagPrefix
	}
}

func GenerateGitHubOutput(semver *semver.Semver, branch string, options ...OptionFunc) (err error) {
	path, exists := os.LookupEnv("GITHUB_OUTPUT")

	if !exists {
		return nil
	}

	output := &GitHubOutput{Semver: semver, Branch: branch}

	for _, option := range options {
		option(output)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening ci file: %w", err)
	}

	defer func() {
		err = f.Close()
		if err != nil {
			return
		}
	}()

	_, err = f.WriteString(output.String())
	if err != nil {
		return fmt.Errorf("writing to ci file: %w", err)
	}

	return
}

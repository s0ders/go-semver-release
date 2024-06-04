// Package ci provides function to generate output for CI/CD pipelines.
package ci

import (
	"fmt"
	"os"
	"strings"

	"github.com/s0ders/go-semver-release/v2/internal/semver"
)

func GenerateGitHubOutput(branch string, prefix string, semver *semver.Semver, release bool) (err error) {
	path, exists := os.LookupEnv("GITHUB_OUTPUT")

	if !exists {
		return nil
	}

	output := fmt.Sprintf("\n%s_SEMVER=%s%s\n%s_NEW_RELEASE=%t\n", strings.ToUpper(branch), prefix, semver.String(), strings.ToUpper(branch), release)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening ci file: %w", err)
	}

	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			return
		}
	}(f)

	_, err = f.WriteString(output)
	if err != nil {
		return fmt.Errorf("writing to ci file: %w", err)
	}

	return nil
}

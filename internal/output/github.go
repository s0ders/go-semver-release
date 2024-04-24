package output

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/s0ders/go-semver-release/internal/semver"
)

type Output struct {
	logger *slog.Logger
}

func NewOutput(logger *slog.Logger) Output {
	return Output{
		logger: logger,
	}
}

func (o Output) Generate(prefix string, semver *semver.Semver, release bool) (err error) {
	path, exists := os.LookupEnv("GITHUB_OUTPUT")

	if !exists {
		return nil
	}

	output := fmt.Sprintf("\nSEMVER=%s%s\nNEW_RELEASE=%t\n", prefix, semver.NormalVersion(), release)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("error opening output file: %w", err)
	}

	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			return
		}
	}(f)

	_, err = f.WriteString(output)
	if err != nil {
		return fmt.Errorf("error writing to output file: %w", err)
	}

	return nil
}

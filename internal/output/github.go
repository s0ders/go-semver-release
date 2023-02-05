package output

import (
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/s0ders/go-semver-release/internal/semver"
)

type Output struct {
	logger *log.Logger
}

func NewOutput() Output {
	return Output{
		logger: log.New(os.Stdout, "output-generator", log.Default().Flags()),
	}
}

func (o Output) Generate(prefix string, semver *semver.Semver, release bool) {
	path, exists := os.LookupEnv("GITHUB_OUTPUT")

	if !exists {
		return
	}

	output := fmt.Sprintf("\nSEMVER=%s%s\nNEW_RELEASE=%t\n", prefix, semver.NormalVersion(), release)

	if err := os.WriteFile(path, []byte(output), fs.ModeAppend); err != nil {
		o.logger.Fatalf("failed to open output file: %s", err)
	}

}
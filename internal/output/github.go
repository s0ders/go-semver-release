package output

import (
	"fmt"
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

	o.logger.Printf("$GITHUB_OUTPUT=%s", path)

	info, err := os.Stat(path)
	if err != nil {
		o.logger.Fatalf("failed to get stat on output file: %s", err)
	}

	o.logger.Printf("%+v", info)

	output := fmt.Sprintf("\nSEMVER=%s%s\nNEW_RELEASE=%t\n", prefix, semver.NormalVersion(), release)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		o.logger.Fatalf("failed to open output file: %s", err)
	}

	defer f.Close()

	_, err = f.WriteString(output)
	if err != nil {
		o.logger.Fatalf("failed to write output to file: %s", err)
	}
}
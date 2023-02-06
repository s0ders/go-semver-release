package releaserules

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/s0ders/go-semver-release/internal/helpers"
)

type ReleaseRuleReader struct {
	l *log.Logger
}

func NewReleaseRuleReader() ReleaseRuleReader {
	logger := log.New(os.Stdout, fmt.Sprintf("%-20s ", "[releas-rule-reader]"), log.Default().Flags())
	return ReleaseRuleReader{
		l: logger,
	}
}

func (r ReleaseRuleReader) Read(path string) (io.Reader, error) {
	if path == "" {
		return strings.NewReader(helpers.DefaultReleaseRules), nil
	}

	return os.Open(path)
}

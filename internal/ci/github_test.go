package ci

import (
	"github.com/s0ders/go-semver-release/internal/semver"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestOutput_GenerateGitHub(t *testing.T) {

	outputDir, err := os.MkdirTemp("./", "output-*")
	if err != nil {
		t.Fatalf("failed to make temp dir: %s", err)
	}
	defer os.RemoveAll(outputDir)

	_, err = os.OpenFile("output", os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		t.Fatalf("error creating output file: %v", err)
	}

	outputPath := filepath.Join(outputDir, "output")

	err = os.Setenv("GITHUB_OUTPUT", outputPath)
	if err != nil {
		t.Fatalf("failed to set GITHUB_OUTPUT: %s", err)
	}

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	output := New(logger)

	version, err := semver.New(1, 2, 3, "")
	if err != nil {
		t.Fatalf("failed to create version: %s", err)
	}

	err = output.GenerateGitHub("v", version, true)
	if err != nil {
		t.Fatalf("failed to generate GitHub: %s", err)
	}

	writtenOutput, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %s", err)
	}

	want := "\nSEMVER=v1.2.3\nNEW_RELEASE=true\n"

	got := string(writtenOutput)

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

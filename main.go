package main

import (
	"os"

	"github.com/s0ders/go-semver-release/v8/cmd"
	"github.com/s0ders/go-semver-release/v8/internal/appcontext"
)

func main() {
	ctx := appcontext.New()
	rootCmd := cmd.NewRootCommand(ctx)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

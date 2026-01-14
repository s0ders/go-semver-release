package main

import (
	"os"

	"github.com/s0ders/go-semver-release/v7/cmd"
	"github.com/s0ders/go-semver-release/v7/internal/appcontext"
)

func main() {
	ctx := appcontext.New()
	rootCmd := cmd.NewRootCommand(ctx)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

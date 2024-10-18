package main

import (
	"os"

	"github.com/s0ders/go-semver-release/v6/cmd"
)

func main() {
	ctx := cmd.NewAppContext()
	rootCmd := cmd.NewRootCommand(ctx)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

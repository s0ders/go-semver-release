package main

import (
	"github.com/s0ders/go-semver-release/v3/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

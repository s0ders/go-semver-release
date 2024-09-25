package main

import (
	"os"

	"github.com/s0ders/go-semver-release/v5/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

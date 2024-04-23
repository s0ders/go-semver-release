package main

import (
	"fmt"
	"os"

	"github.com/s0ders/go-semver-release/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

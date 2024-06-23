// Package branch provides functions to handle branch configuration.
package branch

import (
	"errors"
)

var (
	ErrNoBranch = errors.New("no branch configuration")
	ErrNoName   = errors.New("no name in branch configuration")
)

type Branch struct {
	Name       string
	Prerelease bool
}

// Unmarshall takes a raw Viper configuration and returns a slice of Branch representing a branch configuration.
func Unmarshall(input []map[string]string) ([]Branch, error) {
	if len(input) == 0 {
		return nil, ErrNoBranch
	}

	branches := make([]Branch, len(input))

	for i, b := range input {

		pattern, ok := b["name"]
		if !ok {
			return nil, ErrNoName
		}

		branch := Branch{Name: pattern}

		_, ok = b["prerelease"]
		if ok {
			branch.Prerelease = true
		}

		branches[i] = branch
	}

	return branches, nil
}

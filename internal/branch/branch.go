// Package branch provides functions to handle branch configuration.
package branch

import (
	"errors"
	"fmt"
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
func Unmarshall(input []map[string]any) ([]Branch, error) {
	if len(input) == 0 {
		return nil, ErrNoBranch
	}

	branches := make([]Branch, len(input))

	for i, b := range input {

		name, ok := b["name"]
		if !ok {
			return nil, ErrNoName
		}

		stringName, ok := name.(string)
		if !ok {
			return nil, fmt.Errorf("could not assert that the \"name\" property of the branch configuration is a string")
		}

		branch := Branch{Name: stringName}

		prerelease, ok := b["prerelease"]
		if ok {
			boolPrerelease, ok := prerelease.(bool)
			if !ok {
				return nil, fmt.Errorf("could not assert that the \"prerelease\" property of the branch configuration is a bool")
			}

			branch.Prerelease = boolPrerelease
		}

		branches[i] = branch
	}

	return branches, nil
}

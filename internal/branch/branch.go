package branch

import (
	"errors"
)

var (
	ErrNoBranch  = errors.New("no branch configuration")
	ErrNoPattern = errors.New("no pattern in branch configuration")
)

type Branch struct {
	Pattern              string
	Prerelease           bool
	PrereleaseIdentifier string
}

func Unmarshall(input []map[string]string) ([]Branch, error) {
	if len(input) == 0 {
		return nil, ErrNoBranch
	}

	branches := make([]Branch, len(input))

	for i, b := range input {

		pattern, ok := b["pattern"]
		if !ok {
			return nil, ErrNoPattern
		}

		branch := Branch{Pattern: pattern}

		_, ok = b["prerelease"]
		if ok {
			branch.Prerelease = true
		}

		prereleaseID, ok := b["prerelease-identifier"]
		if ok {
			branch.PrereleaseIdentifier = prereleaseID
		}

		branches[i] = branch
	}

	return branches, nil
}

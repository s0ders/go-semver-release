// Package semver provides basic primitives to work with semantic version numbers.
package semver

import (
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/object"
	"regexp"
	"strconv"
)

var (
	Regex = regexp.MustCompile(`(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
)

type Semver struct {
	Major         int
	Minor         int
	Patch         int
	Prerelease    string
	BuildMetadata string
}

func (s *Semver) BumpPatch() {
	s.Patch++
}

func (s *Semver) BumpMinor() {
	s.Patch = 0
	s.Minor++
}

func (s *Semver) BumpMajor() {
	s.Patch = 0
	s.Minor = 0
	s.Major++
}

// IsZero checks if all component of a semantic version number are equal to zero.
func (s *Semver) IsZero() bool {
	isZero := s.Major == s.Minor && s.Minor == s.Patch && s.Patch == 0
	return isZero
}

func (s *Semver) String() string {
	str := fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)

	if s.Prerelease != "" {
		str += "-" + s.Prerelease
	}

	if s.BuildMetadata != "" {
		str += "+" + s.BuildMetadata
	}

	return str
}

// FromGitTag returns a semver struct corresponding to the Git annotated tag used as an input.
func FromGitTag(tag *object.Tag) (*Semver, error) {
	submatch := Regex.FindStringSubmatch(tag.Name)

	if len(submatch) < 4 {
		return nil, fmt.Errorf("tag cannot be converted to a valid semver")
	}

	major, err := strconv.Atoi(submatch[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert major component: %w", err)
	}
	minor, err := strconv.Atoi(submatch[2])
	if err != nil {
		return nil, fmt.Errorf("failed to convert minor component: %w", err)
	}
	patch, err := strconv.Atoi(submatch[3])
	if err != nil {
		return nil, fmt.Errorf("failed to convert patch component: %w", err)
	}

	prerelease := submatch[4]
	buildMetadata := submatch[5]

	semver := &Semver{Major: major, Minor: minor, Patch: patch, Prerelease: prerelease, BuildMetadata: buildMetadata}

	return semver, nil
}

// Precedence returns an integer representing which of the two versions s or s2 is the most recent. 1 meaning s1 is the
// most recent, -1 that it is s2 and 0 that they are equal.
// TODO: take in account prerelease component
func (s *Semver) Precedence(s2 *Semver) int {
	switch {
	case s.Major > s2.Major:
		return 1
	case s.Major < s2.Major:
		return -1
	case s.Minor > s2.Minor:
		return 1
	case s.Minor < s2.Minor:
		return -1
	case s.Patch > s2.Patch:
		return 1
	case s.Patch < s2.Patch:
		return -1
	case s.Prerelease == "" && s2.Prerelease != "":
		return 1
	case s.Prerelease != "" && s2.Prerelease == "":
		return -1
	default:
		return 0
	}
}

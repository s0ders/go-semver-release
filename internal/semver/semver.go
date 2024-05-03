// Package semver provides basic primitives to work with semantic version numbers.
package semver

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/go-git/go-git/v5/plumbing/object"
)

var Regex = `(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`

type Semver struct {
	Major int
	Minor int
	Patch int
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

func (s *Semver) Valid() bool {
	return s.Patch >= 0 && s.Minor >= 0 && s.Major >= 0
}

func (s *Semver) String() string {
	return fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
}

// FromGitTag returns a semver struct corresponding to the Git annotated tag used as an input.
func FromGitTag(tag *object.Tag) (*Semver, error) {
	regex := regexp.MustCompile(Regex)
	submatch := regex.FindStringSubmatch(tag.Name)

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

	semver := &Semver{major, minor, patch}

	return semver, nil
}

// Precedence returns an integer representing which of the two versions s or s2 is the most recent. 1 meaning s1 is the
// most recent, -1 that it is s2 and 0 that they are equal.
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
	default:
		return 0
	}
}

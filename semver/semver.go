package semver

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// TODO: Handle prerelease tags
type Semver struct {
	Major int
	Minor int
	Patch int
}

func (s *Semver) IncrPatch() {
	s.Patch++
}

func (s *Semver) IncrMinor() {
	s.Patch = 0
	s.Minor++
}

func (s *Semver) IncrMajor() {
	s.Patch = 0
	s.Minor = 0
	s.Major++
}

func (s Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", s.Major, s.Minor, s.Patch)
}

func NewSemver(major, minor, patch int) (*Semver, error) {
	if major < 0 || minor < 0 || patch < 0 {
		return nil, errors.New("Semantic version components cannot be negative")
	}

	return &Semver{major, minor, patch}, nil
}

// NewSemverFromTag returns a semver struct corresponding to
// the tag used as an input.
func NewSemverFromTag(tag *object.Tag) (*Semver, error) {

	semver := strings.Replace(tag.Name, "v", "", 1)
	components := strings.Split(semver, ".")

	if len(components) != 3 {
		return nil, errors.New("Invalid semantic version number")
	}

	major, err := strconv.Atoi(components[0])
	failOnError(err)
	minor, err := strconv.Atoi(components[1])
	failOnError(err)
	patch, err := strconv.Atoi(components[2])
	failOnError(err)

	return NewSemver(major, minor, patch)
}

// compareSemver returns an integer representing
// which of the two versions v1 or v2 passed is
// the most recent. 1 meaning v1 is the most recent,
// -1 that it is v2 and 0 that they are equal.
func CompareSemver(v1, v2 Semver) int {
	switch {
	case v1.Major > v2.Major:
		return 1
	case v1.Major < v2.Major:
		return -1
	case v1.Minor > v2.Minor:
		return 1
	case v1.Minor < v2.Minor:
		return -1
	case v1.Patch > v2.Patch:
		return 1
	case v1.Patch < v2.Patch:
		return -1
	default:
		return 0
	}
}

func failOnError(e error) {
	if e != nil {
		log.Fatalf("Error: %s\n", e)
	}
}

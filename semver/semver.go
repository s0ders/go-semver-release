package semver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-playground/validator/v10"
)

var SemverRegex = `^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`

type Semver struct {
	Major         int    `validate:"gte=0"`
	Minor         int    `validate:"gte=0"`
	Patch         int    `validate:"gte=0"`
	BuildMetadata string // `validate:"omitempty,alphanum"`
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

func (s Semver) NormalVersion() string {
	return fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
}

func (s Semver) String() string {
	if s.BuildMetadata != "" {
		return fmt.Sprintf("%d.%d.%d-%s", s.Major, s.Minor, s.Patch, s.BuildMetadata)
	}

	return s.NormalVersion()
}

// TODO: return a value not a pointer
func NewSemver(major, minor, patch int, metadata string) (*Semver, error) {

	version := &Semver{major, minor, patch, metadata}
	validate := validator.New()

	if err := validate.Struct(version); err != nil {
		return nil, fmt.Errorf("NewSemver: failed to validate struct: %w", err)
	}

	return version, nil
}

// NewSemverFromGitTag returns a semver struct corresponding to
// the Git annotated tag used as an input.
func NewSemverFromGitTag(tag *object.Tag) (*Semver, error) {

	version := strings.Replace(tag.Name, "v", "", 1)
	components := strings.Split(version, ".")

	if len(components) != 3 {
		return nil, errors.New("invalid semantic version number")
	}

	major, err := strconv.Atoi(components[0])
	if err != nil {
		return nil, fmt.Errorf("NewSemver: failed to convert major component: %w", err)
	}
	minor, err := strconv.Atoi(components[1])
	if err != nil {
		return nil, fmt.Errorf("NewSemver: failed to convert minor component: %w", err)
	}
	patch, err := strconv.Atoi(components[2])
	if err != nil {
		return nil, fmt.Errorf("NewSemver: failed to convert patch component: %w", err)
	}

	semver, err := NewSemver(major, minor, patch, "")

	if err != nil {
		return nil, fmt.Errorf("NewSemverFromGitTag: failed to build SemVer: %w", err)
	}

	return semver, nil
}

// Precedence returns an integer representing which of the
// two versions s1 or s2 is the most recent. 1 meaning s1 is
// the most recent, -1 that it is s2 and 0 that they are equal.
func (s1 Semver) Precedence(s2 Semver) int {
	switch {
	case s1.Major > s2.Major:
		return 1
	case s1.Major < s2.Major:
		return -1
	case s1.Minor > s2.Minor:
		return 1
	case s1.Minor < s2.Minor:
		return -1
	case s1.Patch > s2.Patch:
		return 1
	case s1.Patch < s2.Patch:
		return -1
	default:
		return 0
	}
}

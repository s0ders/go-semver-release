// Package semver provides basic primitives to work with semantic versions.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var Regex = regexp.MustCompile(`(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Metadata   string
}

func (v *Version) BumpMajor() {
	v.Major++
	v.Minor = 0
	v.Patch = 0
	v.Prerelease = ""
	v.Metadata = ""
}

func (v *Version) BumpMinor() {
	v.Minor++
	v.Patch = 0
	v.Prerelease = ""
	v.Metadata = ""
}

func (v *Version) BumpPatch() {
	v.Patch++
	v.Prerelease = ""
	v.Metadata = ""
}

// IsZero checks if all component of a semantic version number are equal to zero.
func (v *Version) IsZero() bool {
	isZero := v.Major == v.Minor && v.Minor == v.Patch && v.Patch == 0
	return isZero
}

func (v *Version) String() string {
	str := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)

	if v.Prerelease != "" {
		str += "-" + v.Prerelease
	}

	if v.Metadata != "" {
		str += "+" + v.Metadata
	}

	return str
}

// NewFromString returns a semver struct corresponding to the string used as an input.
func NewFromString(str string) (*Version, error) {
	submatch := Regex.FindStringSubmatch(str)

	if len(submatch) < 4 {
		return nil, fmt.Errorf("string cannot be converted to a valid semver")
	}

	major, err := strconv.Atoi(submatch[1])
	if err != nil {
		return nil, fmt.Errorf("converting major component: %w", err)
	}
	minor, err := strconv.Atoi(submatch[2])
	if err != nil {
		return nil, fmt.Errorf("converting minor component: %w", err)
	}
	patch, err := strconv.Atoi(submatch[3])
	if err != nil {
		return nil, fmt.Errorf("converting patch component: %w", err)
	}

	prerelease := submatch[4]
	buildMetadata := submatch[5]

	semver := &Version{Major: major, Minor: minor, Patch: patch, Prerelease: prerelease, Metadata: buildMetadata}

	return semver, nil
}

// Compare returns an integer representing the precedence of two semantic versions. The result will be 0 if a == b,
// -1 if a < b, and +1 if a > b.
func Compare(a, b *Version) int {
	switch {
	case a.Major > b.Major:
		return 1
	case a.Major < b.Major:
		return -1
	case a.Minor > b.Minor:
		return 1
	case a.Minor < b.Minor:
		return -1
	case a.Patch > b.Patch:
		return 1
	case a.Patch < b.Patch:
		return -1
	case a.Prerelease == "" && b.Prerelease != "":
		return 1
	case a.Prerelease != "" && b.Prerelease == "":
		return -1
	case a.Prerelease != "" && b.Prerelease != "":
		return strings.Compare(a.Prerelease, b.Prerelease)
	default:
		return 0
	}
}

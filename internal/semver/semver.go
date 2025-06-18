// Package semver provides basic primitives to work with semantic versions.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var Regex = regexp.MustCompile(`(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

type Prerelease struct {
	Name  string
	Build int
}

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease *Prerelease
	Metadata   string
}

func (v *Version) BumpMajor() {
	if v.Prerelease != nil && v.Minor == 0 && v.Patch == 0 && v.Prerelease.Build != 0 {
		v.Prerelease.Build++
	} else {
		v.Major++
		if v.Prerelease != nil {
			v.Prerelease.Build = 1
		}
	}
	v.Minor = 0
	v.Patch = 0
	v.Metadata = ""
}

func (v *Version) BumpMinor() {
	if v.Prerelease != nil && v.Patch == 0 && v.Prerelease.Build != 0 {
		v.Prerelease.Build++
	} else {
		v.Minor++
		if v.Prerelease != nil {
			v.Prerelease.Build = 1
		}
	}
	v.Patch = 0
	v.Metadata = ""
}

func (v *Version) BumpPatch() {
	if v.Prerelease != nil && v.Prerelease.Build != 0 {
		v.Prerelease.Build++
	} else {
		v.Patch++
		if v.Prerelease != nil {
			v.Prerelease.Build = 1
		}
	}
	v.Metadata = ""
}

// clones the given semver struct
func (v *Version) Clone() *Version {
	if v == nil {
		return nil
	}
	result := &Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch, Metadata: v.Metadata}
	if v.Prerelease != nil {
		result.Prerelease = &Prerelease{Name: v.Prerelease.Name, Build: v.Prerelease.Build}
	}
	return result
}

// IsZero checks if all component of a semantic version number are equal to zero.
func (v *Version) IsZero() bool {
	isZero := v.Major == v.Minor && v.Minor == v.Patch && v.Patch == 0
	return isZero
}

func (v *Version) String() string {
	str := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)

	if v.Prerelease != nil && v.Prerelease.Name != "" {
		str += "-" + v.Prerelease.String()
	}

	if v.Metadata != "" {
		str += "+" + v.Metadata
	}

	return str
}

func (p *Prerelease) String() string {
	return fmt.Sprintf("%s.%d", p.Name, p.Build)
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

	prerelease, err := PrereleaseFromString(submatch[4])
	if err != nil {
		return nil, err
	}
	buildMetadata := submatch[5]

	semver := &Version{Major: major, Minor: minor, Patch: patch, Prerelease: prerelease, Metadata: buildMetadata}

	return semver, nil
}

// PrereleaseFromString returns a semver conform prerelease struct.
func PrereleaseFromString(str string) (*Prerelease, error) {
	var name, build = "", 0
	parts := strings.Split(str, ".")
	length := len(parts)
	if length == 0 {
		return nil, fmt.Errorf("Prerelease `%s` has the wrong format", str)
	}
	if length >= 1 {
		name = parts[0]
	}
	if length == 2 {
		var err error
		build, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("converting prerelease build component: %w", err)
		}
	}
	if name != "" {
		return &Prerelease{Name: name, Build: build}, nil
	} else {
		return nil, nil
	}
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
	case a.Prerelease != nil && b.Prerelease == nil:
		return 1
	case a.Prerelease == nil && b.Prerelease != nil:
		return -1
	case a.Prerelease != nil && b.Prerelease != nil:
		if a.Prerelease.Name == b.Prerelease.Name {
			switch {
			case a.Prerelease.Build > b.Prerelease.Build:
				return 1
			case a.Prerelease.Build < b.Prerelease.Build:
				return -1
			}
			return 0
		} else {
			return strings.Compare(a.Prerelease.Name, b.Prerelease.Name)
		}
	default:
		return 0
	}
}

// Compare semver against a channel (branch). If the semver tier is the same tier as the channel returns 0, lower -1 and higher 1.
func CompareChannel(s *Version, c string) int {
	switch {
	case s.Prerelease == nil && c == "":
		return 0
	case s.Prerelease == nil && c != "":
		return 1
	case s.Prerelease != nil && c == "":
		return -1
	case s.Prerelease != nil && c != "":
		return strings.Compare(s.Prerelease.Name, c)
	default:
		return 0
	}
}

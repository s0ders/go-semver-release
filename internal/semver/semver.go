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
	Major            int
	Minor            int
	Patch            int
	PrereleaseLabel  string // e.g., "alpha", "beta", "rc"
	PrereleaseNumber int    // e.g., 1, 2, 3 (0 means no number)
	Metadata         string
}

// Prerelease returns the full prerelease string (e.g., "rc.1" or "alpha")
func (v *Version) Prerelease() string {
	if v.PrereleaseLabel == "" {
		return ""
	}
	if v.PrereleaseNumber > 0 {
		return fmt.Sprintf("%s.%d", v.PrereleaseLabel, v.PrereleaseNumber)
	}
	return v.PrereleaseLabel
}

// SetPrerelease sets the prerelease label and resets the number to 1
func (v *Version) SetPrerelease(label string) {
	v.PrereleaseLabel = label
	v.PrereleaseNumber = 1
}

// ClearPrerelease removes prerelease information
func (v *Version) ClearPrerelease() {
	v.PrereleaseLabel = ""
	v.PrereleaseNumber = 0
}

// HasPrerelease returns true if this version has a prerelease identifier
func (v *Version) HasPrerelease() bool {
	return v.PrereleaseLabel != ""
}

// BumpPrerelease increments the prerelease number
func (v *Version) BumpPrerelease() {
	if v.PrereleaseNumber > 0 {
		v.PrereleaseNumber++
	}
}

func (v *Version) BumpMajor() {
	v.Major++
	v.Minor = 0
	v.Patch = 0
	v.ClearPrerelease()
	v.Metadata = ""
}

func (v *Version) BumpMinor() {
	v.Minor++
	v.Patch = 0
	v.ClearPrerelease()
	v.Metadata = ""
}

func (v *Version) BumpPatch() {
	v.Patch++
	v.ClearPrerelease()
	v.Metadata = ""
}

// IsZero checks if all component of a semantic version number are equal to zero.
func (v *Version) IsZero() bool {
	isZero := v.Major == v.Minor && v.Minor == v.Patch && v.Patch == 0
	return isZero
}

func (v *Version) String() string {
	str := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)

	if prerelease := v.Prerelease(); prerelease != "" {
		str += "-" + prerelease
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

	prereleaseStr := submatch[4]
	buildMetadata := submatch[5]

	version := &Version{Major: major, Minor: minor, Patch: patch, Metadata: buildMetadata}

	// Parse prerelease: could be "rc", "rc.1", "alpha.2", etc.
	if prereleaseStr != "" {
		label, number := parsePrerelease(prereleaseStr)
		version.PrereleaseLabel = label
		version.PrereleaseNumber = number
	}

	return version, nil
}

// parsePrerelease splits a prerelease string into label and number.
// Examples: "rc" -> ("rc", 0), "rc.1" -> ("rc", 1), "alpha.2" -> ("alpha", 2)
func parsePrerelease(prerelease string) (label string, number int) {
	// Try to find the last dot followed by a number
	lastDot := strings.LastIndex(prerelease, ".")
	if lastDot == -1 {
		return prerelease, 0
	}

	// Check if everything after the last dot is a number
	suffix := prerelease[lastDot+1:]
	num, err := strconv.Atoi(suffix)
	if err != nil {
		// Not a number, treat the whole thing as the label
		return prerelease, 0
	}

	return prerelease[:lastDot], num
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
	case !a.HasPrerelease() && b.HasPrerelease():
		// Release version > prerelease version
		return 1
	case a.HasPrerelease() && !b.HasPrerelease():
		// Prerelease version < release version
		return -1
	case a.HasPrerelease() && b.HasPrerelease():
		// Compare prerelease labels first
		labelCmp := strings.Compare(a.PrereleaseLabel, b.PrereleaseLabel)
		if labelCmp != 0 {
			return labelCmp
		}
		// Same label, compare numbers
		switch {
		case a.PrereleaseNumber > b.PrereleaseNumber:
			return 1
		case a.PrereleaseNumber < b.PrereleaseNumber:
			return -1
		default:
			return 0
		}
	default:
		return 0
	}
}

// CoreVersion returns a copy of the version without prerelease or metadata
func (v *Version) CoreVersion() Version {
	return Version{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}
}

// SameCoreVersion returns true if two versions have the same major.minor.patch
func (v *Version) SameCoreVersion(other *Version) bool {
	return v.Major == other.Major && v.Minor == other.Minor && v.Patch == other.Patch
}

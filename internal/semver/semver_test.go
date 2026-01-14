package semver

import (
	"testing"

	assertion "github.com/stretchr/testify/assert"
)

func TestSemver_Compare(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		s1, s2 Version
		want   int
	}

	matrix := []test{
		{s1: Version{Major: 1, Minor: 0, Patch: 2}, s2: Version{Major: 1, Minor: 0, Patch: 1}, want: 1},
		{Version{Major: 1, Minor: 0, Patch: 2}, Version{Major: 1, Minor: 0, Patch: 3}, -1},
		{Version{Major: 1, Minor: 0, Patch: 2}, Version{Major: 1, Minor: 1, Patch: 0}, -1},
		{Version{Major: 0, Minor: 0, Patch: 1}, Version{Major: 1, Minor: 1, Patch: 0}, -1},
		{Version{Major: 2, Minor: 0, Patch: 0}, Version{Major: 1, Minor: 99, Patch: 99}, 1},
		{Version{Major: 99, Minor: 0, Patch: 0}, Version{Major: 2, Minor: 99, Patch: 99}, 1},
		{s1: Version{Major: 1, Minor: 0, Patch: 0}, s2: Version{Major: 1, Minor: 0, Patch: 0}},
		{s1: Version{Major: 0, Minor: 2, Patch: 0}, s2: Version{Major: 0, Minor: 1, Patch: 0}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, Metadata: "foo"}, s2: Version{Major: 0, Minor: 1, Patch: 0}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0}, s2: Version{Major: 0, Minor: 1, Patch: 0, Metadata: "foo"}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "rc"}, s2: Version{Major: 0, Minor: 1, Patch: 0}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "rc"}, s2: Version{Major: 0, Minor: 2, Patch: 0}, want: -1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0}, s2: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "rc"}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, Metadata: "foo"}, s2: Version{Major: 0, Minor: 2, Patch: 0, Metadata: "bar"}, want: 0},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "rc"}, s2: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "alpha"}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "alpha"}, s2: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "beta"}, want: -1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "rc"}, s2: Version{Major: 0, Minor: 2, Patch: 0, PrereleaseLabel: "rc"}, want: 0},
		// New tests for prerelease numbers
		{s1: Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 2}, s2: Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 1}, want: 1},
		{s1: Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 1}, s2: Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 2}, want: -1},
		{s1: Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 1}, s2: Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 1}, want: 0},
		{s1: Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "beta", PrereleaseNumber: 99}, s2: Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 1}, want: -1},
	}

	for _, tc := range matrix {
		got := Compare(&tc.s1, &tc.s2)
		assert.Equal(got, tc.want, "semver precedence is not correct")
	}
}

func TestSemver_IsZero(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		semver Version
		want   bool
	}

	matrix := []test{
		{Version{Major: 0, Minor: 0, Patch: 0}, true},
		{Version{Major: 0, Minor: 0, Patch: 1}, false},
		{Version{Major: 0, Minor: 1, Patch: 0}, false},
		{Version{Major: 1, Minor: 0, Patch: 0}, false},
		{Version{Major: 1, Minor: 1, Patch: 1}, false},
	}

	for _, tc := range matrix {
		got := tc.semver.IsZero()
		assert.Equal(got, tc.want, "semver has not been correctly classified as zero")
	}
}

func TestSemver_String(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		have Version
		want string
	}

	tests := []test{
		{Version{Major: 1, Minor: 0, Patch: 0}, "1.0.0"},
		{Version{Major: 1, Minor: 0, Patch: 1, PrereleaseLabel: "rc"}, "1.0.1-rc"},
		{Version{Major: 1, Minor: 0, Patch: 1, Metadata: "metadata"}, "1.0.1+metadata"},
		{Version{Major: 1, Minor: 0, Patch: 1, PrereleaseLabel: "alpha", Metadata: "metadata"}, "1.0.1-alpha+metadata"},
		// New tests for numbered prereleases
		{Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 1}, "1.0.0-rc.1"},
		{Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "alpha", PrereleaseNumber: 2}, "1.0.0-alpha.2"},
		{Version{Major: 2, Minor: 1, Patch: 0, PrereleaseLabel: "beta", PrereleaseNumber: 10, Metadata: "build123"}, "2.1.0-beta.10+build123"},
	}

	for _, tc := range tests {
		assert.Equal(tc.want, tc.have.String(), "the strings should be equal")
	}
}

func TestSemver_NewFromString_HappyScenario(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		got  string
		want Version
	}

	matrix := []test{
		{"1.2.3", Version{Major: 1, Minor: 2, Patch: 3}},
		{"1.2.3-rc", Version{Major: 1, Minor: 2, Patch: 3, PrereleaseLabel: "rc"}},
		{"1.2.3+metadata", Version{Major: 1, Minor: 2, Patch: 3, Metadata: "metadata"}},
		{"1.2.3-rc+metadata", Version{Major: 1, Minor: 2, Patch: 3, PrereleaseLabel: "rc", Metadata: "metadata"}},
		// New tests for numbered prereleases
		{"1.2.3-rc.1", Version{Major: 1, Minor: 2, Patch: 3, PrereleaseLabel: "rc", PrereleaseNumber: 1}},
		{"1.2.3-alpha.2", Version{Major: 1, Minor: 2, Patch: 3, PrereleaseLabel: "alpha", PrereleaseNumber: 2}},
		{"1.2.3-beta.10+metadata", Version{Major: 1, Minor: 2, Patch: 3, PrereleaseLabel: "beta", PrereleaseNumber: 10, Metadata: "metadata"}},
	}

	for _, tt := range matrix {
		semver, err := NewFromString(tt.got)
		assert.NoError(err, "should have created a semver from string")

		assert.Equal(tt.want, *semver, "version should be equal")
	}
}

func TestSemver_NewFromString_BadScenario(t *testing.T) {
	assert := assertion.New(t)

	invalidStrings := []string{
		"",
		"foo",
		"-1.-1.-1",
		"1.0.0+$@",
		"1.0.0-$@",
	}

	for _, str := range invalidStrings {
		_, err := NewFromString(str)
		assert.Error(err, "should have failed to create a semver from invalid string")
	}
}

func TestSemver_Bump(t *testing.T) {
	assert := assertion.New(t)

	s := &Version{Major: 0, Minor: 0, Patch: 0}

	s.BumpPatch()
	assert.Equal(s.String(), "0.0.1", "the strings should be equal")
	assert.Empty(s.Prerelease(), "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMinor()
	assert.Equal(s.String(), "0.1.0", "the strings should be equal")
	assert.Empty(s.Prerelease(), "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMajor()
	assert.Equal(s.String(), "1.0.0", "the strings should be equal")
	assert.Empty(s.Prerelease(), "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")
}

func TestSemver_BumpPrerelease(t *testing.T) {
	assert := assertion.New(t)

	s := &Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 1}

	s.BumpPrerelease()
	assert.Equal("1.0.0-rc.2", s.String(), "prerelease should be bumped")

	s.BumpPrerelease()
	assert.Equal("1.0.0-rc.3", s.String(), "prerelease should be bumped again")
}

func TestSemver_SetPrerelease(t *testing.T) {
	assert := assertion.New(t)

	s := &Version{Major: 1, Minor: 0, Patch: 0}

	s.SetPrerelease("alpha")
	assert.Equal("1.0.0-alpha.1", s.String(), "prerelease should be set")
	assert.Equal("alpha", s.PrereleaseLabel, "prerelease label should be set")
	assert.Equal(1, s.PrereleaseNumber, "prerelease number should be 1")

	s.SetPrerelease("beta")
	assert.Equal("1.0.0-beta.1", s.String(), "prerelease should be reset")
}

func TestSemver_ClearPrerelease(t *testing.T) {
	assert := assertion.New(t)

	s := &Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 3}

	s.ClearPrerelease()
	assert.Equal("1.0.0", s.String(), "prerelease should be cleared")
	assert.False(s.HasPrerelease(), "should not have prerelease")
}

func TestSemver_HasPrerelease(t *testing.T) {
	assert := assertion.New(t)

	s1 := &Version{Major: 1, Minor: 0, Patch: 0}
	assert.False(s1.HasPrerelease(), "should not have prerelease")

	s2 := &Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc"}
	assert.True(s2.HasPrerelease(), "should have prerelease")

	s3 := &Version{Major: 1, Minor: 0, Patch: 0, PrereleaseLabel: "rc", PrereleaseNumber: 1}
	assert.True(s3.HasPrerelease(), "should have prerelease")
}

func TestSemver_SameCoreVersion(t *testing.T) {
	assert := assertion.New(t)

	s1 := &Version{Major: 1, Minor: 2, Patch: 3}
	s2 := &Version{Major: 1, Minor: 2, Patch: 3, PrereleaseLabel: "rc", PrereleaseNumber: 1}
	s3 := &Version{Major: 1, Minor: 2, Patch: 4}

	assert.True(s1.SameCoreVersion(s2), "should have same core version")
	assert.False(s1.SameCoreVersion(s3), "should not have same core version")
}

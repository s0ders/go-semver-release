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
		{s1: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, s2: Version{Major: 0, Minor: 1, Patch: 0}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, s2: Version{Major: 0, Minor: 2, Patch: 0}, want: -1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0}, s2: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, Metadata: "foo"}, s2: Version{Major: 0, Minor: 2, Patch: 0, Metadata: "bar"}, want: 0},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, s2: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "alpha"}, want: 1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "alpha"}, s2: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "beta"}, want: -1},
		{s1: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, s2: Version{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, want: 0},
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
		{Version{Major: 1, Minor: 0, Patch: 1, Prerelease: "rc"}, "1.0.1-rc"},
		{Version{Major: 1, Minor: 0, Patch: 1, Metadata: "metadata"}, "1.0.1+metadata"},
		{Version{Major: 1, Minor: 0, Patch: 1, Prerelease: "alpha", Metadata: "metadata"}, "1.0.1-alpha+metadata"},
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
		{"1.2.3-rc", Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "rc"}},
		{"1.2.3+metadata", Version{Major: 1, Minor: 2, Patch: 3, Metadata: "metadata"}},
		{"1.2.3-rc+metadata", Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "rc", Metadata: "metadata"}},
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
	assert.Empty(s.Prerelease, "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMinor()
	assert.Equal(s.String(), "0.1.0", "the strings should be equal")
	assert.Empty(s.Prerelease, "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMajor()
	assert.Equal(s.String(), "1.0.0", "the strings should be equal")
	assert.Empty(s.Prerelease, "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")
}

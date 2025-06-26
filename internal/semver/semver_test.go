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
		{
			s1:   Version{Major: 0, Minor: 2, Patch: 0, Metadata: "foo"},
			s2:   Version{Major: 0, Minor: 1, Patch: 0},
			want: 1,
		},
		{
			s1:   Version{Major: 0, Minor: 2, Patch: 0, Metadata: "foo"},
			s2:   Version{Major: 0, Minor: 2, Patch: 0, Metadata: "bar"},
			want: 0,
		},
		{
			s1:   Version{Major: 0, Minor: 2, Patch: 0},
			s2:   Version{Major: 0, Minor: 1, Patch: 0, Metadata: "foo"},
			want: 1,
		},
		{
			s1:   Version{Major: 0, Minor: 2, Patch: 0},
			s2:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 1}},
			want: -1,
		},
		{
			s1:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 1}},
			s2:   Version{Major: 0, Minor: 1, Patch: 0},
			want: 1,
		},
		{
			s1:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 2}},
			s2:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 1}},
			want: 1,
		},
		{
			s1:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 1}},
			s2:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 2}},
			want: -1,
		},
		{
			s1:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 2}},
			s2:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 2}},
			want: 0,
		},
		{
			s1:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "beta", Build: 1}},
			s2:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "alpha", Build: 1}},
			want: 1,
		},
		{
			s1:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "alpha", Build: 1}},
			s2:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "beta", Build: 1}},
			want: -1,
		},
		{
			s1:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "beta", Build: 1}},
			s2:   Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "beta", Build: 1}},
			want: 0,
		},
	}

	for i, tc := range matrix {
		got := Compare(&tc.s1, &tc.s2)
		assert.Equal(got, tc.want, "semver precedence is not correct [%d]", i)
	}
}

func TestSemver_CompareChannel(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		s    Version
		c    string
		want int
	}

	matrix := []test{
		{s: Version{Major: 1, Minor: 0, Patch: 2}, c: "", want: 0},
		{s: Version{Major: 1, Minor: 0, Patch: 2}, c: "beta", want: 1},
		{s: Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "alpha"}}, c: "", want: -1},
		{s: Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "alpha"}}, c: "alpha", want: 0},
		{s: Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "alpha"}}, c: "beta", want: -1},
		{s: Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "beta"}}, c: "alpha", want: 1},
	}

	for i, tc := range matrix {
		got := CompareChannel(&tc.s, tc.c)
		assert.Equal(got, tc.want, "semver precedence is not correct [%d]", i)
	}
}

func TestSemver_Clone(t *testing.T) {
	assert := assertion.New(t)

	tests := []*Version{
		nil,
		{Major: 1, Minor: 0, Patch: 0},
		{Major: 0, Minor: 1, Patch: 0},
		{Major: 0, Minor: 0, Patch: 1},
		{Major: 2, Minor: 1, Patch: 0},
		{Major: 2, Minor: 2, Patch: 1},
		{Major: 1, Minor: 0, Patch: 0, Metadata: "metadata"},
		{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{}},
		{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "alpha"}},
		{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "alpha", Build: 1}},
	}

	for i, tc := range tests {
		assert.Equal(tc, tc.Clone(), "semver clone should be equal [%d]", i)
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

	for i, tc := range matrix {
		got := tc.semver.IsZero()
		assert.Equal(got, tc.want, "semver has not been correctly classified as zero [%d]", i)
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
		{Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 1}}, "1.0.0-rc.1"},
		{Version{Major: 1, Minor: 0, Patch: 1, Metadata: "metadata"}, "1.0.1+metadata"},
		{Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "alpha", Build: 99}, Metadata: "metadata"}, "1.0.0-alpha.99+metadata"},
	}

	for i, tc := range tests {
		assert.Equal(tc.want, tc.have.String(), "the strings should be equal [%d]", i)
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
		{"0.1.2-alpha", Version{Major: 0, Minor: 1, Patch: 2, Prerelease: &Prerelease{Name: "alpha", Build: 0}}},
		{"1.0.0-rc.123", Version{Major: 1, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 123}}},
		{"1.2.3+metadata", Version{Major: 1, Minor: 2, Patch: 3, Metadata: "metadata"}},
		{"3.0.0-rc.123+metadata", Version{Major: 3, Minor: 0, Patch: 0, Prerelease: &Prerelease{Name: "rc", Build: 123}, Metadata: "metadata"}},
	}

	for i, tt := range matrix {
		semver, err := NewFromString(tt.got)
		assert.NoErrorf(err, "should have created a semver from string [%d]", i)

		assert.Equal(tt.want, *semver, "version should be equal [%d]", i)
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

	for i, str := range invalidStrings {
		_, err := NewFromString(str)
		assert.Errorf(err, "should have failed to create a semver from invalid string [%d]", i)
	}
}

func TestSemver_Bump(t *testing.T) {
	assert := assertion.New(t)

	s := &Version{Major: 0, Minor: 1, Patch: 0}

	s.BumpPatch()
	assert.Equal("0.1.1", s.String(), "the strings should be equal")
	assert.Nil(s.Prerelease, "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMinor()
	assert.Equal("0.2.0", s.String(), "the strings should be equal")
	assert.Nil(s.Prerelease, "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMajor()
	assert.Equal("1.0.0", s.String(), "the strings should be equal")
	assert.Nil(s.Prerelease, "version prerelease should be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.Prerelease = &Prerelease{Name: "beta"}
	s.BumpMajor()
	assert.Equal("2.0.0-beta.1", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	// first version in new prerelease branch
	s = &Version{Major: 0, Minor: 1, Patch: 1, Prerelease: &Prerelease{Name: "beta"}}
	s.BumpPatch()
	assert.Equal("0.1.2-beta.1", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s = &Version{Major: 0, Minor: 1, Patch: 0, Prerelease: &Prerelease{Name: "beta"}}
	s.BumpMinor()
	assert.Equal("0.2.0-beta.1", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s = &Version{Major: 0, Minor: 1, Patch: 1, Prerelease: &Prerelease{Name: "beta"}}
	s.BumpMinor()
	assert.Equal("0.2.0-beta.1", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s = &Version{Major: 0, Minor: 1, Patch: 1, Prerelease: &Prerelease{Name: "beta"}}
	s.BumpMajor()
	assert.Equal("1.0.0-beta.1", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	// prerelease branch version bumps
	s = &Version{Major: 0, Minor: 1, Patch: 1, Prerelease: &Prerelease{Name: "alpha", Build: 4}}
	s.BumpPatch()
	assert.Equal("0.1.1-alpha.5", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMinor()
	assert.Equal("0.2.0-alpha.1", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMinor()
	assert.Equal("0.2.0-alpha.2", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpPatch()
	assert.Equal("0.2.0-alpha.3", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMajor()
	assert.Equal("1.0.0-alpha.1", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMinor()
	assert.Equal("1.0.0-alpha.2", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpPatch()
	assert.Equal("1.0.0-alpha.3", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")

	s.BumpMajor()
	assert.Equal("1.0.0-alpha.4", s.String(), "the strings should be equal")
	assert.NotNil(s.Prerelease, "version prerelease should not be empty after bump")
	assert.Empty(s.Metadata, "version metadata should be empty after bump")
}

package semver

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
)

func TestSemver_Precedence(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		s1, s2 *Semver
		want   int
	}

	matrix := []test{
		{s1: &Semver{Major: 1, Minor: 0, Patch: 2}, s2: &Semver{Major: 1, Minor: 0, Patch: 1}, want: 1},
		{&Semver{Major: 1, Minor: 0, Patch: 2}, &Semver{Major: 1, Minor: 0, Patch: 3}, -1},
		{&Semver{Major: 1, Minor: 0, Patch: 2}, &Semver{Major: 1, Minor: 1, Patch: 0}, -1},
		{&Semver{Major: 0, Minor: 0, Patch: 1}, &Semver{Major: 1, Minor: 1, Patch: 0}, -1},
		{&Semver{Major: 2, Minor: 0, Patch: 0}, &Semver{Major: 1, Minor: 99, Patch: 99}, 1},
		{&Semver{Major: 99, Minor: 0, Patch: 0}, &Semver{Major: 2, Minor: 99, Patch: 99}, 1},
		{s1: &Semver{Major: 1, Minor: 0, Patch: 0}, s2: &Semver{Major: 1, Minor: 0, Patch: 0}},
		{s1: &Semver{Major: 0, Minor: 2, Patch: 0}, s2: &Semver{Major: 0, Minor: 1, Patch: 0}, want: 1},
		{s1: &Semver{Major: 0, Minor: 2, Patch: 0, BuildMetadata: "foo"}, s2: &Semver{Major: 0, Minor: 1, Patch: 0}, want: 1},
		{s1: &Semver{Major: 0, Minor: 2, Patch: 0}, s2: &Semver{Major: 0, Minor: 1, Patch: 0, BuildMetadata: "foo"}, want: 1},
		{s1: &Semver{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, s2: &Semver{Major: 0, Minor: 1, Patch: 0}, want: 1},
		{s1: &Semver{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, s2: &Semver{Major: 0, Minor: 2, Patch: 0}, want: -1},
		{s1: &Semver{Major: 0, Minor: 2, Patch: 0}, s2: &Semver{Major: 0, Minor: 2, Patch: 0, Prerelease: "rc"}, want: 1},
		{s1: &Semver{Major: 0, Minor: 2, Patch: 0, BuildMetadata: "foo"}, s2: &Semver{Major: 0, Minor: 2, Patch: 0, BuildMetadata: "bar"}, want: 0},
	}

	for _, test := range matrix {
		got := test.s1.Precedence(test.s2)
		assert.Equal(got, test.want, "semver precedence is not correct")
	}
}

func TestSemver_IsZero(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		semver Semver
		want   bool
	}

	matrix := []test{
		{Semver{Major: 0, Minor: 0, Patch: 0}, true},
		{Semver{Major: 0, Minor: 0, Patch: 1}, false},
		{Semver{Major: 0, Minor: 1, Patch: 0}, false},
		{Semver{Major: 1, Minor: 0, Patch: 0}, false},
		{Semver{Major: 1, Minor: 1, Patch: 1}, false},
	}

	for _, test := range matrix {
		got := test.semver.IsZero()
		assert.Equal(got, test.want, "semver has not been correctly classified as zero")
	}
}

func TestSemver_String(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		have Semver
		want string
	}

	tests := []test{
		{Semver{Major: 1, Minor: 0, Patch: 0}, "1.0.0"},
		{Semver{Major: 1, Minor: 0, Patch: 1, Prerelease: "rc"}, "1.0.1-rc"},
		{Semver{Major: 1, Minor: 0, Patch: 1, BuildMetadata: "metadata"}, "1.0.1+metadata"},
		{Semver{Major: 1, Minor: 0, Patch: 1, Prerelease: "alpha", BuildMetadata: "metadata"}, "1.0.1-alpha+metadata"},
	}

	for _, testCase := range tests {
		assert.Equal(testCase.want, testCase.have.String(), "the strings should be equal")
	}

}

func TestSemver_FromGitTag(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		tag  *object.Tag
		want string
	}

	tag1 := &object.Tag{
		Name:    "v1.2.3",
		Message: "1.2.3",
		Tagger: object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	}

	tag2 := &object.Tag{
		Name:    "1.2.3",
		Message: "1.2.3",
		Tagger: object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	}

	tag3 := &object.Tag{
		Name:    "v.1.2.3",
		Message: "1.2.3",
		Tagger: object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	}

	tag4 := &object.Tag{
		Name:    "version1.2.3-rc+buildmetadata",
		Message: "1.2.3",
		Tagger: object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	}

	matrix := []test{
		{tag1, "1.2.3"},
		{tag2, "1.2.3"},
		{tag3, "1.2.3"},
		{tag4, "1.2.3-rc+buildmetadata"},
	}

	for _, test := range matrix {
		semver, err := FromGitTag(test.tag)
		assert.NoError(err, "should have created a semver from Git tag")

		assert.Equal(test.want, semver.String(), "the strings should be equal")
	}
}

func TestSemver_FromGitTagInvalid(t *testing.T) {
	assert := assert.New(t)

	notSemverTag := &object.Tag{
		Name:    "foo",
		Message: "foo",
		Tagger: object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	}

	_, err := FromGitTag(notSemverTag)
	assert.Error(err, "should have failed to create a semver from invalid tag")
}

func TestSemver_Bump(t *testing.T) {
	assert := assert.New(t)

	s := Semver{Major: 0, Minor: 0, Patch: 0}

	s.BumpPatch()
	assert.Equal(s.String(), "0.0.1", "the strings should be equal")

	s.BumpMinor()
	assert.Equal(s.String(), "0.1.0", "the strings should be equal")

	s.BumpMajor()
	assert.Equal(s.String(), "1.0.0", "the strings should be equal")
}

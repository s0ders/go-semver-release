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
		{&Semver{1, 0, 2}, &Semver{1, 0, 1}, 1},
		{&Semver{1, 0, 2}, &Semver{1, 0, 3}, -1},
		{&Semver{1, 0, 2}, &Semver{1, 1, 0}, -1},
		{&Semver{0, 0, 1}, &Semver{1, 1, 0}, -1},
		{&Semver{2, 0, 0}, &Semver{1, 99, 99}, 1},
		{&Semver{99, 0, 0}, &Semver{2, 99, 99}, 1},
		{&Semver{1, 0, 0}, &Semver{1, 0, 0}, 0},
		{&Semver{0, 2, 0}, &Semver{0, 1, 0}, 1},
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
		{Semver{0, 0, 0}, true},
		{Semver{0, 0, 1}, false},
		{Semver{0, 1, 0}, false},
		{Semver{1, 0, 0}, false},
		{Semver{1, 1, 1}, false},
	}

	for _, test := range matrix {
		got := test.semver.IsZero()
		assert.Equal(got, test.want, "semver has not been correctly classified as zero")
	}
}

func TestSemver_String(t *testing.T) {
	assert := assert.New(t)
	s := Semver{1, 2, 3}

	want := "1.2.3"
	assert.Equal(s.String(), want, "the strings should be equal")
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

	matrix := []test{
		{tag1, "1.2.3"},
		{tag2, "1.2.3"},
		{tag3, "1.2.3"},
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

	s := Semver{0, 0, 0}

	s.BumpPatch()
	assert.Equal(s.String(), "0.0.1", "the strings should be equal")

	s.BumpMinor()
	assert.Equal(s.String(), "0.1.0", "the strings should be equal")

	s.BumpMajor()
	assert.Equal(s.String(), "1.0.0", "the strings should be equal")
}

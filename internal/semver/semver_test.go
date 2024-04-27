package semver

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
)

func TestPrecedence(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		s1, s2 *Semver
		want   int
	}

	matrix := []test{
		{&Semver{1, 0, 2, ""}, &Semver{1, 0, 1, ""}, 1},
		{&Semver{1, 0, 2, ""}, &Semver{1, 0, 3, ""}, -1},
		{&Semver{1, 0, 2, ""}, &Semver{1, 1, 0, ""}, -1},
		{&Semver{0, 0, 1, ""}, &Semver{1, 1, 0, ""}, -1},
		{&Semver{2, 0, 0, ""}, &Semver{1, 99, 99, ""}, 1},
		{&Semver{99, 0, 0, ""}, &Semver{2, 99, 99, ""}, 1},
		{&Semver{1, 0, 0, ""}, &Semver{1, 0, 0, ""}, 0},
		{&Semver{0, 1, 0, "d364937"}, &Semver{0, 1, 0, "f61d9c2"}, 0},
	}

	for _, test := range matrix {
		got := test.s1.Precedence(test.s2)
		assert.Equal(got, test.want, "semver precedence is not correct")
	}
}

func TestIsZero(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		semver Semver
		want   bool
	}

	matrix := []test{
		{Semver{0, 0, 0, ""}, true},
		{Semver{0, 0, 1, ""}, false},
		{Semver{0, 1, 0, ""}, false},
		{Semver{1, 0, 0, ""}, false},
		{Semver{1, 1, 1, ""}, false},
	}

	for _, test := range matrix {
		got := test.semver.IsZero()
		assert.Equal(got, test.want, "semver has not been correctly classified as zero")
	}
}

func TestNormalVersion(t *testing.T) {
	assert := assert.New(t)

	version, err := New(1, 2, 3, "d364937ad663484d80c28485f60a91cf2af2f932")
	assert.NoError(err, "should have been able to create semver")

	want := "1.2.3+d364937ad663484d80c28485f60a91cf2af2f932"
	assert.Equal(version.String(), want, "the strings should be equal")

	want = "1.2.3"
	assert.Equal(version.NormalVersion(), want, "the strings should be equal")
}

func TestNegativeSemver(t *testing.T) {
	assert := assert.New(t)

	_, err := New(-1, 0, 0, "")
	assert.Error(err, "should have failed to create a negative semver")
}

func TestSemverString(t *testing.T) {
	assert := assert.New(t)
	s, _ := New(1, 2, 3, "")

	want := "1.2.3"
	assert.Equal(s.String(), want, "the strings should be equal")
}

func TestNewSemverFromGitTag(t *testing.T) {
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

func TestBump(t *testing.T) {
	assert := assert.New(t)

	s, err := New(0, 0, 0, "")
	assert.NoError(err, "should have created a semver")

	s.BumpPatch()
	assert.Equal(s.String(), "0.0.1", "the strings should be equal")

	s.BumpMinor()
	assert.Equal(s.String(), "0.1.0", "the strings should be equal")

	s.BumpMajor()
	assert.Equal(s.String(), "1.0.0", "the strings should be equal")
}

func BenchmarkPrecedence(b *testing.B) {
	s1, _ := New(1, 0, 2, "f61d9c2")
	s2, _ := New(1, 0, 3, "d364937")

	for i := 0; i < b.N; i++ {
		s1.Precedence(s2)
	}
}

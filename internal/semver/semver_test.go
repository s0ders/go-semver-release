package semver

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestPrecedence(t *testing.T) {
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
		if got := test.s1.Precedence(test.s2); got != test.want {
			t.Fatalf("got: %d want: %d\n", got, test.want)
		}
	}
}

func TestIsZero(t *testing.T) {
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
		if got := test.semver.IsZero(); got != test.want {
			t.Fatalf("got: %t want: %t", got, test.want)
		}
	}
}

func TestNormalVersion(t *testing.T) {
	version, err := New(1, 2, 3, "d364937ad663484d80c28485f60a91cf2af2f932")
	if err != nil {
		t.Fatalf("Failed to create semver: %s", err)
	}

	want := "1.2.3+d364937ad663484d80c28485f60a91cf2af2f932"
	if got := version.String(); got != want {
		t.Fatalf("Got: %s Want: %s", got, want)
	}

	want = "1.2.3"
	if got := version.NormalVersion(); got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

func TestNegativeSemver(t *testing.T) {
	_, err := New(-1, 0, 0, "")

	if err == nil {
		t.Fatalf("managed to create negative semver")
	}
}

func TestSemverString(t *testing.T) {
	s, _ := New(1, 2, 3, "")

	want := "1.2.3"
	if got := s.String(); got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

func TestNewSemverFromGitTag(t *testing.T) {
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
		if err != nil {
			t.Fatalf("failed to create semver: %s", err)
		}

		if got := semver.String(); got != test.want {
			t.Fatalf("got: %s want: %s", got, test.want)
		}
	}
}

func TestBump(t *testing.T) {
	s, err := New(0, 0, 0, "")
	if err != nil {
		t.Fatalf("failed to created semver: %s", err)
	}

	s.BumpPatch()
	if got := s.String(); got != "0.0.1" {
		t.Fatalf("got: %s want: %s", got, "0.0.1")
	}

	s.BumpMinor()
	if got := s.String(); got != "0.1.0" {
		t.Fatalf("got: %s want: %s", got, "0.1.0")
	}

	s.BumpMajor()
	if got := s.String(); got != "1.0.0" {
		t.Fatalf("got: %s want: %s", got, "1.0.0")
	}
}

func BenchmarkPrecedence(b *testing.B) {
	s1, _ := New(1, 0, 2, "f61d9c2")
	s2, _ := New(1, 0, 3, "d364937")

	for i := 0; i < b.N; i++ {
		s1.Precedence(s2)
	}
}

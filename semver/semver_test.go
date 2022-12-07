package semver

import (
	"testing"
)

func TestCompareSemver(t *testing.T) {

	type test struct {
		s1, s2 Semver
		want   int
	}

	matrix := []test{
		{Semver{1, 0, 2, ""}, Semver{1, 0, 1, ""}, 1},
		{Semver{1, 0, 2, ""}, Semver{1, 0, 3, ""}, -1},
		{Semver{1, 0, 2, ""}, Semver{1, 1, 0, ""}, -1},
		{Semver{0, 0, 1, ""}, Semver{1, 1, 0, ""}, -1},
		{Semver{2, 0, 0, ""}, Semver{1, 99, 99, ""}, 1},
		{Semver{99, 0, 0, ""}, Semver{2, 99, 99, ""}, 1},
		{Semver{1, 0, 0, ""}, Semver{1, 0, 0, ""}, 0},
	}

	for _, test := range matrix {

		if got := test.s1.Precedence(test.s2); got != test.want {
			t.Fatalf("Got: %d Want: %d\n", got, test.want)
		}
	}

}

func TestNegativeSemver(t *testing.T) {
	_, err := NewSemver(-1, 0, 0, "")

	if err == nil {
		t.Fatalf("Error, managed to create negative semver")
	}
}

func TestSemverString(t *testing.T) {
	s, _ := NewSemver(1, 2, 3, "")

	want := "1.2.3"
	if got := s.String(); got != want {
		t.Fatalf("Got: %s Want: %s", got, want)
	}
}

func BenchmarkCompareSemver(b *testing.B) {
	s1, _ := NewSemver(1, 0, 2, "")
	s2, _ := NewSemver(1, 0, 3, "")

	for i := 0; i < b.N; i++ {
		s1.Precedence(*s2)
	}
}

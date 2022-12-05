package main

import (
	"testing"
)

func TestCompare(t *testing.T) {

	type test struct {
		s1, s2 Semver
		want   int
	}

	matrix := []test{
		{Semver{1, 0, 2}, Semver{1, 0, 1}, 1},
		{Semver{1, 0, 2}, Semver{1, 0, 3}, -1},
		{Semver{1, 0, 2}, Semver{1, 1, 0}, -1},
		{Semver{0, 0, 1}, Semver{1, 1, 0}, -1},
		{Semver{2, 0, 0}, Semver{1, 99, 99}, 1},
		{Semver{99, 0, 0}, Semver{2, 99, 99}, 1},
		{Semver{1, 0, 0}, Semver{1, 0, 0}, 0},
	}

	for _, test := range matrix {

		if got := CompareSemver(test.s1, test.s2); got != test.want {
			t.Fatalf("Got: %d Want: %d\n", got, test.want)
		}
	}

}

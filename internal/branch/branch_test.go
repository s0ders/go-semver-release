package branch

import (
	"testing"

	assertion "github.com/stretchr/testify/assert"
)

func TestBranch_UnmarshallError(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		have []map[string]string
		want error
	}

	tests := []test{
		{have: []map[string]string{}, want: ErrNoBranch},
		{have: []map[string]string{{"prerelease": "true"}}, want: ErrNoPattern},
		{have: []map[string]string{{"pattern": "alpha", "prerelease": "true", "prerelease-identifier": "alpha"}}, want: nil},
	}

	for _, tc := range tests {
		_, err := Unmarshall(tc.have)
		assert.Equal(tc.want, err)
	}
}

func TestBranch_Unmarshall(t *testing.T) {
	assert := assertion.New(t)

	have := []map[string]string{{"pattern": "main"}, {"pattern": "alpha", "prerelease": "true", "prerelease-identifier": "alpha"}}
	want := []Branch{
		{Name: "main"},
		{Name: "alpha", Prerelease: true, PrereleaseIdentifier: "alpha"},
	}

	branches, err := Unmarshall(have)
	if err != nil {
		t.Fatalf("unmarshalling branches: %s", err)
	}

	assert.Equal(want, branches)
}

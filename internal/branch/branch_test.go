package branch

import (
	"testing"

	assertion "github.com/stretchr/testify/assert"
)

func TestBranch_Unmarshall(t *testing.T) {
	assert := assertion.New(t)

	have := []map[string]string{{"name": "main"}, {"name": "alpha", "prerelease": "true"}}
	want := []Branch{
		{Name: "main"},
		{Name: "alpha", Prerelease: true},
	}

	branches, err := Unmarshall(have)
	if err != nil {
		t.Fatalf("unmarshalling branches: %s", err)
	}

	assert.Equal(want, branches)
}

func TestBranch_UnmarshallErrors(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		have []map[string]string
		want error
	}

	tests := []test{
		{have: []map[string]string{}, want: ErrNoBranch},
		{have: []map[string]string{{"prerelease": "true"}}, want: ErrNoName},
		{have: []map[string]string{{"name": "alpha", "prerelease": "true"}}, want: nil},
	}

	for _, tc := range tests {
		_, err := Unmarshall(tc.have)
		assert.Equal(tc.want, err)
	}
}

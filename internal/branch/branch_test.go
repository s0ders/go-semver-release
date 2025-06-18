package branch

import (
	"testing"

	assertion "github.com/stretchr/testify/assert"
)

func TestBranch_Unmarshall(t *testing.T) {
	assert := assertion.New(t)

	have := []map[string]any{
		{"name": "alpha", "prerelease": true},
		{"name": "main"},
		{"name": "beta", "prerelease": true},
	}
	want := []Branch{
		{Name: "main"},
		{Name: "beta", Prerelease: true},
		{Name: "alpha", Prerelease: true},
	}

	branches, err := Unmarshall(have)
	if err != nil {
		t.Fatalf("unmarshalling branches: %s", err)
	}

	assert.Equal(want, branches, "should return all branches in order")
}

func TestBranch_UnmarshallErrors(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		have []map[string]any
		want error
	}

	tests := []test{
		{have: []map[string]any{}, want: ErrNoBranch},
		{have: []map[string]any{{"prerelease": true}}, want: ErrNoName},
		{have: []map[string]any{{"name": "alpha", "prerelease": true}}, want: nil},
	}

	for _, tc := range tests {
		_, err := Unmarshall(tc.have)
		assert.Equal(tc.want, err)
	}
}

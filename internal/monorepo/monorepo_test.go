package monorepo

import (
	"path/filepath"
	"testing"

	assertion "github.com/stretchr/testify/assert"
)

func TestMonorepo_Unmarshall(t *testing.T) {
	assert := assertion.New(t)

	have := []map[string]any{{"name": "bar", "path": "./bar/"}, {"name": "foo", "path": "./xyz/foo/"}}

	localizedPath, _ := filepath.Localize("xyz/foo")

	want := []Project{
		{Name: "bar", Path: "bar", Release: true},
		{Name: "foo", Path: localizedPath, Release: true},
	}

	branches, err := Unmarshall(have)
	if err != nil {
		t.Fatalf("unmarshalling projects: %s", err)
	}

	assert.Equal(want, branches)
}

func TestMonorepo_UnmarshallErrors(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		have []map[string]any
		want error
	}

	tests := []test{
		{have: []map[string]any{{"path": "./foo/"}}, want: ErrNoName},
		{have: []map[string]any{{"name": "foo"}}, want: ErrNoPath},
		{have: []map[string]any{}, want: ErrNoProjects},
		{have: []map[string]any{{"name": "foo", "path": "./foo/"}}, want: nil},
	}

	for _, tc := range tests {
		_, err := Unmarshall(tc.have)
		assert.Equal(tc.want, err)
	}
}

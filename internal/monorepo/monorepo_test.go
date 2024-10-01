package monorepo

import (
	"path/filepath"
	"testing"

	assertion "github.com/stretchr/testify/assert"
)

func TestMonorepo_Unmarshall(t *testing.T) {
	assert := assertion.New(t)

	have := []map[string]string{{"name": "bar", "path": "./bar/"}, {"name": "foo", "path": "./xyz/foo/"}}

	localizedPath, _ := filepath.Localize("xyz/foo")

	want := []Project{
		{Name: "bar", Path: "bar"},
		{Name: "foo", Path: localizedPath},
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
		have []map[string]string
		want error
	}

	tests := []test{
		{have: []map[string]string{{"path": "./foo/"}}, want: ErrNoName},
		{have: []map[string]string{{"name": "foo"}}, want: ErrNoPath},
		{have: []map[string]string{}, want: ErrNoProjects},
		{have: []map[string]string{{"name": "foo", "path": "./foo/"}}, want: nil},
	}

	for _, tc := range tests {
		_, err := Unmarshall(tc.have)
		assert.Equal(tc.want, err)
	}
}

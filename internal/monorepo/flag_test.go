package monorepo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchFlag_String(t *testing.T) {
	assert := assert.New(t)

	monorepoConfiguration := []map[string]any{{"name": "foo", "path": "./foo/"}, {"name": "bar", "path": "./bar./"}}
	monorepoConfigurationFlag := Flag(monorepoConfiguration)

	var emptyFlag Flag

	type test struct {
		got  *Flag
		want string
	}

	tests := []test{
		{got: &monorepoConfigurationFlag, want: "[{\"name\":\"foo\",\"path\":\"./foo/\"},{\"name\":\"bar\",\"path\":\"./bar./\"}]"},
		{got: &emptyFlag, want: "[]"},
	}

	for _, tc := range tests {
		assert.Equal(tc.want, tc.got.String())
	}
}

func TestBranchFlag_Set(t *testing.T) {
	var flag Flag

	err := flag.Set("[{\"name\": \"foo\"}]")
	assert.NoError(t, err, "should not have errored")

	err = flag.Set("{\"name\": \"foo\"}")
	assert.Error(t, err, "should have errored, invalid JSON string")
}

func TestBranchFlag_Type(t *testing.T) {
	var f Flag

	assert.Equal(t, FlagType, f.Type())
}

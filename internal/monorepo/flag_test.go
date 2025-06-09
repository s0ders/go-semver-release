package monorepo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMonorepoFlag_String(t *testing.T) {
	assert := assert.New(t)

	monorepoConfiguration := []Item{
		{Name: "foo", Path: "./foo/"},
		{Name: "bar", Path: "./bar./"},
	}

	monorepoConfigurationFlag := Flag(monorepoConfiguration)

	var emptyFlag Flag

	type test struct {
		got  *Flag
		want string
	}

	tests := []test{
		{got: &monorepoConfigurationFlag, want: "[{\"name\":\"foo\",\"path\":\"./foo/\",\"paths\":null},{\"name\":\"bar\",\"path\":\"./bar./\",\"paths\":null}]"},
		{got: &emptyFlag, want: "[]"},
	}

	for _, tc := range tests {
		assert.Equal(tc.want, tc.got.String())
	}
}

func TestMonorepoFlag_Set(t *testing.T) {
	var flag Flag

	err := flag.Set("[{\"name\": \"foo\"}]")
	assert.NoError(t, err, "should not have errored")

	err = flag.Set("{\"name\": \"foo\"}")
	assert.Error(t, err, "should have errored, invalid JSON string")
}

func TestMonorepoFlag_Type(t *testing.T) {
	var f Flag

	assert.Equal(t, FlagType, f.Type())
}

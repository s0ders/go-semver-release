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

func TestMonorepoFlag_Set_ExclusivePathAndPaths(t *testing.T) {
	var f Flag

	// Should error when both Path and Paths are set
	err := f.Set(`[{"name":"test","path":"./path","paths":["./other"]}]`)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExclusiveFlag)
}

func TestMonorepoFlag_Set_EmptyValues(t *testing.T) {
	var f Flag

	// Empty string should succeed and result in empty flag
	err := f.Set("")
	assert.NoError(t, err)
	assert.Len(t, f, 0)

	// Empty array should succeed and result in empty flag
	err = f.Set("[]")
	assert.NoError(t, err)
	assert.Len(t, f, 0)
}

func TestMonorepoFlag_Set_ClearsPreviousValues(t *testing.T) {
	var f Flag

	// Set initial value
	err := f.Set(`[{"name":"first","path":"./first"}]`)
	assert.NoError(t, err)
	assert.Len(t, f, 1)
	assert.Equal(t, "first", f[0].Name)

	// Set new value - should clear the previous one
	err = f.Set(`[{"name":"second","path":"./second"}]`)
	assert.NoError(t, err)
	assert.Len(t, f, 1)
	assert.Equal(t, "second", f[0].Name)
}

func TestMonorepoFlag_Set_MultiplePaths(t *testing.T) {
	var f Flag

	// Using Paths array instead of Path
	err := f.Set(`[{"name":"multi","paths":["./path1","./path2"]}]`)
	assert.NoError(t, err)
	assert.Len(t, f, 1)
	assert.Equal(t, "multi", f[0].Name)
	assert.Len(t, f[0].Paths, 2)
	assert.Equal(t, "./path1", f[0].Paths[0])
	assert.Equal(t, "./path2", f[0].Paths[1])
}

func TestMonorepoFlag_GetItems(t *testing.T) {
	// Test nil pointer returns nil
	var nilFlag *Flag
	assert.Nil(t, nilFlag.GetItems())

	// Test empty flag returns empty slice (nil in Go)
	var emptyFlag Flag
	items := emptyFlag.GetItems()
	assert.Len(t, items, 0)

	// Test with items
	f := Flag{{Name: "foo", Path: "./foo"}, {Name: "bar", Path: "./bar"}}
	items = f.GetItems()
	assert.Len(t, items, 2)
	assert.Equal(t, "foo", items[0].Name)
	assert.Equal(t, "bar", items[1].Name)
}

func TestMonorepoFlag_String_NilPointer(t *testing.T) {
	var nilFlag *Flag
	assert.Equal(t, "[]", nilFlag.String())
}

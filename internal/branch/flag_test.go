package branch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchFlag_String(t *testing.T) {
	assert := assert.New(t)

	normalBranchConfiguration := []Item{
		{Name: "master", Prerelease: false},
		{Name: "rc", Prerelease: true},
	}
	normalBranchConfigurationFlag := Flag(normalBranchConfiguration)

	var emptyFlag Flag

	type test struct {
		got  *Flag
		want string
	}

	tests := []test{
		{got: &normalBranchConfigurationFlag, want: "[{\"name\":\"master\",\"prerelease\":false},{\"name\":\"rc\",\"prerelease\":true}]"},
		{got: &emptyFlag, want: "[]"},
	}

	for _, tc := range tests {
		assert.Equal(tc.want, tc.got.String())
	}
}

func TestBranchFlag_Set(t *testing.T) {
	var flag Flag

	err := flag.Set("[{\"name\": \"main\"}]")
	assert.NoError(t, err, "should not have errored")

	err = flag.Set("{\"name\": \"main\"}")
	assert.Error(t, err, "should have errored, invalid JSON string")
}

func TestBranchFlag_Type(t *testing.T) {
	var f Flag

	assert.Equal(t, FlagType, f.Type())
}

func TestBranchFlag_Set_EmptyValues(t *testing.T) {
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

func TestBranchFlag_Set_ClearsPreviousValues(t *testing.T) {
	var f Flag

	// Set initial value
	err := f.Set(`[{"name":"first","prerelease":false}]`)
	assert.NoError(t, err)
	assert.Len(t, f, 1)
	assert.Equal(t, "first", f[0].Name)

	// Set new value - should clear the previous one
	err = f.Set(`[{"name":"second","prerelease":true}]`)
	assert.NoError(t, err)
	assert.Len(t, f, 1)
	assert.Equal(t, "second", f[0].Name)
	assert.True(t, f[0].Prerelease)
}

func TestBranchFlag_Set_WithPrereleaseBase(t *testing.T) {
	var f Flag

	err := f.Set(`[{"name":"rc","prerelease":true,"prereleaseBase":"main"}]`)
	assert.NoError(t, err)
	assert.Len(t, f, 1)
	assert.Equal(t, "rc", f[0].Name)
	assert.True(t, f[0].Prerelease)
	assert.Equal(t, "main", f[0].PrereleaseBase)
}

func TestBranchFlag_GetItems(t *testing.T) {
	// Test nil pointer returns nil
	var nilFlag *Flag
	assert.Nil(t, nilFlag.GetItems())

	// Test empty flag returns empty slice (nil in Go)
	var emptyFlag Flag
	items := emptyFlag.GetItems()
	assert.Len(t, items, 0)

	// Test with items
	f := Flag{
		{Name: "main", Prerelease: false},
		{Name: "rc", Prerelease: true},
	}
	items = f.GetItems()
	assert.Len(t, items, 2)
	assert.Equal(t, "main", items[0].Name)
	assert.Equal(t, "rc", items[1].Name)
}

func TestBranchFlag_String_NilPointer(t *testing.T) {
	var nilFlag *Flag
	assert.Equal(t, "[]", nilFlag.String())
}

func TestBranchItem(t *testing.T) {
	// Test Item struct with all fields
	item := Item{
		Name:           "rc",
		Prerelease:     true,
		PrereleaseBase: "main",
	}

	assert.Equal(t, "rc", item.Name)
	assert.True(t, item.Prerelease)
	assert.Equal(t, "main", item.PrereleaseBase)
}

func TestBranchConfig(t *testing.T) {
	// Test Config struct
	config := Config{
		Items: []Item{
			{Name: "main", Prerelease: false},
			{Name: "rc", Prerelease: true, PrereleaseBase: "main"},
		},
	}

	assert.Len(t, config.Items, 2)
	assert.Equal(t, "main", config.Items[0].Name)
	assert.False(t, config.Items[0].Prerelease)
	assert.Equal(t, "rc", config.Items[1].Name)
	assert.True(t, config.Items[1].Prerelease)
}

func TestBranchErrorVariables(t *testing.T) {
	// Test that error variables are defined and have expected messages
	assert.NotNil(t, ErrNoBranch)
	assert.NotNil(t, ErrNoName)
	assert.Contains(t, ErrNoBranch.Error(), "branch")
	assert.Contains(t, ErrNoName.Error(), "name")
}

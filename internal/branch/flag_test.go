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

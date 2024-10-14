package rule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchFlag_String(t *testing.T) {
	assert := assert.New(t)

	normalBranchConfiguration := map[string][]string{"minor": {"feat"}, "patch": {"fix"}}
	normalBranchConfigurationFlag := Flag(normalBranchConfiguration)

	var emptyFlag Flag

	type test struct {
		got  *Flag
		want string
	}

	tests := []test{
		{got: &normalBranchConfigurationFlag, want: "{\"minor\":[\"feat\"],\"patch\":[\"fix\"]}"},
		{got: &emptyFlag, want: "{}"},
	}

	for _, tc := range tests {
		assert.Equal(tc.want, tc.got.String())
	}
}

func TestBranchFlag_Set(t *testing.T) {

	var flag Flag

	err := flag.Set("[{\"name\": \"main\"}]")
	assert.Error(t, err, "should have errored, invalid JSON string")

	err = flag.Set("{\"minor\": [\"feat\"]}")
	assert.NoError(t, err, "should not have errored")
}

func TestBranchFlag_Type(t *testing.T) {
	var f Flag

	assert.Equal(t, FlagType, f.Type())
}

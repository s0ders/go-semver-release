package rule

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRulesFlag_String(t *testing.T) {
	assert := assert.New(t)

	normalBranchConfiguration := map[string]string{
		"feat": "minor",
		"fix":  "patch",
	}
	normalBranchConfigurationFlag := Flag(normalBranchConfiguration)

	var emptyFlag Flag

	type test struct {
		got  *Flag
		want string
	}

	tests := []test{
		{got: &normalBranchConfigurationFlag, want: "{\"feat\":\"minor\",\"fix\":\"patch\"}"},
		{got: &emptyFlag, want: "{}"},
	}

	for _, tc := range tests {
		assert.Equal(tc.want, tc.got.String())
	}
}

func TestRulesFlag_Type(t *testing.T) {
	var f Flag

	assert.Equal(t, FlagType, f.Type())
}

func TestRulesFlag_Set_HappyScenario(t *testing.T) {
	assert := assert.New(t)

	var flag Flag

	have, err := json.Marshal(map[string][]string{"minor": {"feat"}, "patch": {"fix", "perf", "revert"}})
	if err != nil {
		t.Fatalf("failed to marshal rules: %s", err)
	}

	want := map[string]string{
		"feat":   "minor",
		"fix":    "patch",
		"perf":   "patch",
		"revert": "patch",
	}

	if err = flag.Set(string(have)); err != nil {
		t.Fatalf("failed to set rules: %s", err)
	}

	assert.Equal(Flag(want), flag)
}

func TestRulesFlag_Set_BadScenario(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		have map[string][]string
		want error
	}

	tests := []test{
		{have: map[string][]string{"minor": {"feat"}, "patch": {"fix", "perf"}}, want: nil},
		{have: map[string][]string{"unknown": {"feat"}, "patch": {"perf"}}, want: ErrInvalidReleaseType},
		{have: map[string][]string{"minor": {"unknown"}, "patch": {"perf"}}, want: ErrInvalidCommitType},
		{have: map[string][]string{"minor": {"feat"}, "patch": {"fix", "feat"}}, want: ErrDuplicateReleaseRule},
	}

	for _, tc := range tests {
		var flag Flag

		b, err := json.Marshal(tc.have)
		if err != nil {
			t.Fatalf("failed to marshal rules: %s", err)
		}

		err = flag.Set(string(b))
		assert.ErrorIs(err, tc.want, "should have failed to set rules")
	}
}

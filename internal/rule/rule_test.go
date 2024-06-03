package rule

import (
	"testing"

	assertion "github.com/stretchr/testify/assert"
)

func TestRule_UnmarshallError(t *testing.T) {
	assert := assertion.New(t)

	type test struct {
		have map[string][]string
		want error
	}

	tests := []test{
		{have: map[string][]string{"minor": {"feat"}, "patch": {"fix", "perf"}}, want: nil},
		{have: map[string][]string{"unknown": {"feat"}, "patch": {"perf"}}, want: ErrInvalidReleaseType},
		{have: map[string][]string{"minor": {"unknown"}, "patch": {"perf"}}, want: ErrInvalidCommitType},
		{have: map[string][]string{"minor": {"feat"}, "patch": {"fix", "feat"}}, want: ErrDuplicateReleaseRule},
		{have: map[string][]string{}, want: ErrNoRules},
	}

	for _, tc := range tests {
		_, err := Unmarshall(tc.have)
		assert.Equal(tc.want, err)
	}
}

func TestRule_Unmarshall(t *testing.T) {
	assert := assertion.New(t)

	have := map[string][]string{"minor": {"feat"}, "patch": {"fix", "perf", "revert"}}
	want := Rules{Map: map[string]string{
		"feat":   "minor",
		"fix":    "patch",
		"perf":   "patch",
		"revert": "patch",
	}}

	rules, err := Unmarshall(have)
	if err != nil {
		t.Fatalf("unmarshalling rules: %s", err)
	}

	assert.Equal(want, rules)
}

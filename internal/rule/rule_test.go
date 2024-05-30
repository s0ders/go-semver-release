package rule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRule_Validate(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		have Rules
		want error
	}

	tests := []test{
		{have: Rules{Unmarshalled: map[string][]string{"minor": {"feat"}, "patch": {"fix", "perf"}}}, want: nil},
		{have: Rules{Unmarshalled: map[string][]string{"unknown": {"feat"}, "patch": {"perf"}}}, want: ErrInvalidReleaseType},
		{have: Rules{Unmarshalled: map[string][]string{"minor": {"unknown"}, "patch": {"perf"}}}, want: ErrInvalidCommitType},
	}

	for _, tc := range tests {
		assert.Equal(tc.want, tc.have.Validate())
	}
}

func TestRule_Map(t *testing.T) {
	assert := assert.New(t)

	have := Rules{
		Unmarshalled: map[string][]string{"minor": {"feat"}, "patch": {"fix", "perf"}},
	}

	want := map[string]string{
		"feat": "minor",
		"fix":  "patch",
		"perf": "patch",
	}

	assert.Equal(want, have.Map())
}

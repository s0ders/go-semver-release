package releaserules

import (
	"fmt"
	"strings"
	"testing"
)

func TestMap(t *testing.T) {
	releaseRules, err := NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to parse release rules: %s", err)
	}

	got := releaseRules.Map()
	want := map[string]string{
		"feat": "minor",
		"perf": "minor",
		"fix": "patch",
	}

	if fmt.Sprintf("%+v", got) != fmt.Sprintf("%+v", want) {
		t.Fatalf("failed to map, got:\n %+v", got)
	}
}

func TestParseReleaseRules(t *testing.T) {

	releaseRules, err := NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to parse release rules: %s", err)
	}

	type test struct {
		commitType  string
		releaseType string
	}

	matrix := []test{
		{"feat", "minor"},
		{"perf", "minor"},
		{"fix", "patch"},
	}

	for i := 0; i < len(releaseRules.Rules); i++ {
		got := releaseRules.Rules[i]
		want := matrix[i]

		if got.CommitType != want.commitType {
			t.Fatalf("got: %s want: %s", got.CommitType, want.commitType)
		}
		if got.ReleaseType != want.releaseType {
			t.Fatalf("got: %s want: %s", got.ReleaseType, want.releaseType)
		}
	}
}

func TestSemanticallyIncorrectRules(t *testing.T) {
	const incorrectRules = `{
		"releaseRules": [
			{"type": "feat", "release": "minor"},
			{"type": "feat", "release": "major"},
			{"type": "fix", "release": "patch"}
		]
	}`

	reader := strings.NewReader(incorrectRules)

	ruleReader, err := NewReleaseRuleReader().setReader(reader).Parse()

	if err == nil {
		t.Fatalf("did not detect incorrect rules: %+v", ruleReader)
	}
}
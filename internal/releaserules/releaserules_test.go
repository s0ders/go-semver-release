package releaserules

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestMap(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	rulesReader, err := New(logger).Read("")
	if err != nil {
		t.Fatalf("failed to read rules: %s", err)
	}

	rules, err := rulesReader.Parse()
	if err != nil {
		t.Fatalf("failed to parse rules: %s", err)
	}

	got := rules.Map()
	want := map[string]string{
		"feat": "minor",
		"perf": "minor",
		"fix":  "patch",
	}

	if fmt.Sprintf("%+v", got) != fmt.Sprintf("%+v", want) {
		t.Fatalf("failed to map, got:\n %+v", got)
	}
}

func TestParseDefaultReleaseRules(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	rulesReader, err := New(logger).Read("")
	if err != nil {
		t.Fatalf("failed to read rules: %s", err)
	}

	rules, err := rulesReader.Parse()
	if err != nil {
		t.Fatalf("failed to parse rules: %s", err)
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

	for i := 0; i < len(rules.Rules); i++ {
		got := rules.Rules[i]
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
		"rules": [
			{"type": "feat", "release": "minor"},
			{"type": "feat", "release": "major"},
			{"type": "fix", "release": "patch"}
		]
	}`

	reader := strings.NewReader(incorrectRules)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ruleReader := New(logger)
	ruleReader.reader = reader
	_, err := ruleReader.Parse()

	if err == nil {
		t.Fatalf("did not detect incorrect rules: %+v", ruleReader)
	}
}

func TestIsJSON(t *testing.T) {
	type test struct {
		have string
		want bool
	}

	tests := []test{
		{have: "{\"foo\": \"bar\"}", want: true},
		{have: "not a valid json", want: false},
		{have: "foo: bar", want: false},
	}

	for _, testCase := range tests {
		if got := isJSON([]byte(testCase.have)); got != testCase.want {
			t.Errorf("got: %v, want: %v", got, testCase.want)
		}
	}
}

func TestIsYAML(t *testing.T) {
	type test struct {
		have string
		want bool
	}

	var validYAML = `foo: "ok"
bar: true
baz: 1.21
obj:
  prop1: "foo"
  prop2: "bar"
  anArray:
    - item1
    - item2`

	tests := []test{
		{have: validYAML, want: true},
		{have: "not a valid yaml", want: false},
		{have: "{\"foo\": \"bar\"}", want: true},
	}

	for _, testCase := range tests {
		if got := isYAML([]byte(testCase.have)); got != testCase.want {
			t.Errorf("got: %v, want: %v with %q", got, testCase.want, testCase.have)
		}
	}
}

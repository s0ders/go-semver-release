package rules

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseRules_Map(t *testing.T) {
	assert := assert.New(t)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	rulesReader, err := New(logger).Read("")
	assert.NoError(err, "should have been able to reed rules")

	rules, err := rulesReader.Parse()
	assert.NoError(err, "should have been able to parse rules")

	got := rules.Map()
	want := map[string]string{
		"feat": "minor",
		"perf": "minor",
		"fix":  "patch",
	}

	assert.Equal(want, got, "rule maps should match")
}

func TestReleaseRules_ParseDefault(t *testing.T) {
	assert := assert.New(t)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	rulesReader, err := New(logger).Read("")
	assert.NoError(err, "should have been able to reed rules")

	rules, err := rulesReader.Parse()
	assert.NoError(err, "should have been able to parse rules")

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

		assert.Equal(want.commitType, got.CommitType, "commit type should match")
		assert.Equal(want.releaseType, got.ReleaseType, "release type should match")
	}
}

func TestReleaseRules_IncorrectRules(t *testing.T) {
	assert := assert.New(t)

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

	assert.Error(err, "should have detected incorrect rules")
}

func TestReleaseRules_IsJSON(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		have string
		want bool
	}

	tests := []test{
		{have: "{\"foo\": \"bar\"}", want: true},
		{have: "not a valid json", want: false},
		{have: "foo: bar", want: false},
	}

	for _, test := range tests {
		got := isJSON([]byte(test.have))
		assert.Equal(test.want, got, "should have detected JSON")
	}
}

func TestReleaseRules_IsYAML(t *testing.T) {
	assert := assert.New(t)

	type test struct {
		have string
		want bool
	}

	validYAML := `foo: "ok"
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

	for _, test := range tests {
		got := isYAML([]byte(test.have))
		assert.Equal(test.want, got, "should have detected YAML")
	}
}

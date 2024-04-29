package rules

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRules_Map(t *testing.T) {
	assert := assert.New(t)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	rulesReader, err := New(logger).Read("")
	assert.NoError(err, "should have been able to reed rules")

	rules, err := rulesReader.Parse()
	assert.NoError(err, "should have been able to parse rules")

	got := rules.Map()
	want := map[string]string{
		"feat":   "minor",
		"fix":    "patch",
		"perf":   "patch",
		"revert": "patch",
	}

	assert.Equal(want, got, "rule maps should match")
}

func TestRules_ParseDefault(t *testing.T) {
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
		{"fix", "patch"},
		{"perf", "patch"},
		{"revert", "patch"},
	}

	for i := 0; i < len(rules.Rules); i++ {
		got := rules.Rules[i]
		want := matrix[i]

		assert.Equal(want.commitType, got.CommitType, "commit type should match")
		assert.Equal(want.releaseType, got.ReleaseType, "release type should match")
	}
}

func TestRules_IncorrectRules(t *testing.T) {
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

func TestRules_InvalidRulesFilePath(t *testing.T) {
	assert := assert.New(t)

	invalidFilePath := "./foo/bar/baz"

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	_, err := New(logger).Read(invalidFilePath)
	assert.Error(err, "should have failed trying to open invalid file path")
}

func TestRules_RulesFile(t *testing.T) {
	assert := assert.New(t)

	tempDir, err := os.MkdirTemp("", "rules-*")
	assert.NoError(err, "failed to create temp directory")

	defer func() {
		err = os.RemoveAll(tempDir)
		assert.NoError(err, "failed to remove temp. dir.")
	}()

	rulesFilePath := filepath.Join(tempDir, "rules.json")

	file, err := os.Create(rulesFilePath)
	assert.NoError(err, "failed to write to temp. rules file")

	defer func() {
		err = file.Close()
		assert.NoError(err, "failed to close rules file")
	}()

	_, err = file.Write([]byte(Default))
	assert.NoError(err, "failed to write to rules file")

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	_, err = New(logger).Read(rulesFilePath)
	assert.NoError(err, "failed trying to read rulesFile")

	// TODO: compare reader values
}

func TestRules_IsJSON(t *testing.T) {
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

func TestRules_IsYAML(t *testing.T) {
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

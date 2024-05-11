package rule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRules_Map(t *testing.T) {
	assert := assert.New(t)

	rules, err := Init()
	assert.NoError(err, "should have been able to parse rule")

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

	rules, err := Init()
	assert.NoError(err, "should have been able to parse rule")

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

func TestRules_DuplicateType(t *testing.T) {
	assert := assert.New(t)

	const duplicateRules = `{
		"rule": [
			{"type": "feat", "release": "minor"},
			{"type": "feat", "release": "patch"}
		]
	}`

	reader := strings.NewReader(duplicateRules)

	_, err := Init(WithReader(reader))
	assert.ErrorIs(err, ErrDuplicateReleaseRule, "should have detected incorrect rule")
}

func TestRules_NoRules(t *testing.T) {
	assert := assert.New(t)

	reader := strings.NewReader(`{"rule": []}`)

	_, err := Init(WithReader(reader))
	assert.ErrorIs(err, ErrNoRules, "should have detected empty rule")
}

func TestRules_InvalidType(t *testing.T) {
	assert := assert.New(t)

	const duplicateRules = `{
		"rule": [
			{"type": "feat", "release": "minor"},
			{"type": "unknown", "release": "patch"}
		]
	}`

	reader := strings.NewReader(duplicateRules)

	_, err := Init(WithReader(reader))
	assert.ErrorIs(err, ErrInvalidCommitType, "should have detected incorrect rule")
}

func TestRules_InvalidRelease(t *testing.T) {
	assert := assert.New(t)

	// Using a "release" of type major is forbidden since they are
	// reserved for breaking changes.
	const duplicateRules = `{
		"rule": [
			{"type": "feat", "release": "minor"},
			{"type": "fix", "release": "major"}
		]
	}`

	reader := strings.NewReader(duplicateRules)

	_, err := Init(WithReader(reader))
	assert.ErrorIs(err, ErrInvalidReleaseType, "should have detected incorrect rule")
}

func TestRules_EmptyFile(t *testing.T) {
	assert := assert.New(t)

	tempDir, err := os.MkdirTemp("", "rule-*")
	assert.NoError(err, "failed to create temp. dir.")

	defer func() {
		err = os.RemoveAll(tempDir)
		assert.NoError(err, "failed to remove temp. dir.")
	}()

	emptyFilePath := filepath.Join(tempDir, "empty.json")

	emptyFile, err := os.Create(emptyFilePath)
	assert.NoError(err, "failed to create empty rule file")

	_, err = emptyFile.Write([]byte("{}"))
	assert.NoError(err, "failed to write empty rule file")

	defer func() {
		err = emptyFile.Close()
		assert.NoError(err, "failed to close empty rule file")
	}()

	_, err = Init(WithReader(emptyFile))
	assert.Error(err, "should have failed to decode JSON")
}

package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRules_Map(t *testing.T) {
	assert := assert.New(t)

	opts := &Options{}

	rules, err := Init(opts)
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

	opts := &Options{}

	rules, err := Init(opts)
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

func TestRules_DuplicateType(t *testing.T) {
	assert := assert.New(t)

	const duplicateRules = `{
		"rules": [
			{"type": "feat", "release": "minor"},
			{"type": "feat", "release": "patch"}
		]
	}`

	reader := strings.NewReader(duplicateRules)

	opts := &Options{
		Reader: reader,
	}

	_, err := Init(opts)
	assert.ErrorIs(err, ErrDuplicateReleaseRule, "should have detected incorrect rules")
}

func TestRules_InvalidType(t *testing.T) {
	assert := assert.New(t)

	const duplicateRules = `{
		"rules": [
			{"type": "feat", "release": "minor"},
			{"type": "unknown", "release": "patch"}
		]
	}`

	reader := strings.NewReader(duplicateRules)

	opts := &Options{
		Reader: reader,
	}

	_, err := Init(opts)
	assert.ErrorIs(err, ErrInvalidCommitType, "should have detected incorrect rules")
}

func TestRules_InvalidRelease(t *testing.T) {
	assert := assert.New(t)

	// Using a "release" of type major is forbidden since they are
	// reserved for breaking changes.
	const duplicateRules = `{
		"rules": [
			{"type": "feat", "release": "minor"},
			{"type": "fix", "release": "major"}
		]
	}`

	reader := strings.NewReader(duplicateRules)

	opts := &Options{
		Reader: reader,
	}

	_, err := Init(opts)
	assert.ErrorIs(err, ErrInvalidReleaseType, "should have detected incorrect rules")
}

func TestRules_EmptyFile(t *testing.T) {
	assert := assert.New(t)

	tempDir, err := os.MkdirTemp("", "rules-*")
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

	opts := &Options{
		Reader: emptyFile,
	}

	_, err = Init(opts)
	assert.Error(err, "should have failed to decode JSON")
}

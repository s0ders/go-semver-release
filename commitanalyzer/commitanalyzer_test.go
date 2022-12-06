package commitanalyzer

import (
	"strings"
	"testing"
)

func TestSemverRegex(t *testing.T) {
	type test struct {
		tagName string
		isValidSemver bool
	}

	matrix := []test{
		{"foo", false},
		{"v1.2.3", true},
		{"v999.2.3", true},
		{"v1.2.3-pre", false},
		{"v1.2.3-pre+meta", false},
		{"1.2.3", false},
	}

	

	for _, item := range matrix {
		if got := semverRegex.MatchString(item.tagName); got != item.isValidSemver {
			t.Fatalf("Got: %t Want: %t with tag %s\n", got, item.isValidSemver, item.tagName)
		}
	}
}

func TestCommitTypeRegex(t *testing.T) {
	type test struct {
		commit string
		commitType string
	}

	matrix := []test{
		{"feat: implemented foo", "feat"},
		{"fix(foo.js): fixed foo", "fix"},
		{"chore(api): fixed doc typos", "chore"},
		{"test(../tests/): implemented unit tests", "test"},
		{"ci(ci.yaml): added stages to pipeline", "ci"},
	}

	for _, item := range matrix {
		got := conventionalCommitRegex.FindStringSubmatch(item.commit)[1]
		if got != item.commitType {
			t.Fatalf("Got: %s Want: %s\n", got, item.commitType)
		}
	}
}

func TestBreakingChangeRegex(t *testing.T) {
	type test struct {
		commit string
		isBreaking bool
	}

	matrix := []test{
		{"feat: implemented foo", false},
		{"fix(foo.js)!: fixed foo", true},
		{"chore(docs): fixed doc typos BREAKING CHANGE: delete some APIs", true},
	}

	for _, item := range matrix {
		submatch := conventionalCommitRegex.FindStringSubmatch(item.commit)
		got := strings.Contains(submatch[3], "!") || strings.Contains(submatch[0], "BREAKING CHANGE")
		if got != item.isBreaking {
			t.Fatalf("Got: %t Want: %t with commit %s\n", got, item.isBreaking, item.commit)
		}
	}
}
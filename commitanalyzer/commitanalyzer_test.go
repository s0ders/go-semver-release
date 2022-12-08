package commitanalyzer

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	defaultReleaseRules = `{
		"releaseRules": [
			{"type": "feat", "release": "minor"},
			{"type": "perf", "release": "minor"},
			{"type": "fix", "release": "patch"}
		]
	}`
)

func TestCommitTypeRegex(t *testing.T) {
	type test struct {
		commit     string
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
		commit     string
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

func TestParseReleaseRules(t *testing.T) {
	
	releaseRules, err := ParseReleaseRules(strings.NewReader(defaultReleaseRules))
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
		want :=  matrix[i]

		if got.CommitType != want.commitType {
			t.Fatalf("got: %s want: %s", got.CommitType, want.commitType)
		}
		if got.ReleaseType != want.releaseType {
			t.Fatalf("got: %s want: %s", got.ReleaseType, want.releaseType)
		}
	}
}

func TestFetchLatestSemverTag(t *testing.T) {
	
	tempDirPath := filepath.Join(".", "commitanalyzer-test")
	err := os.Mkdir(tempDirPath, 0644)
	if err != nil {
		t.Fatalf("failed to create temp directory: %s", err)
	}

	defer os.RemoveAll(tempDirPath)

	r, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		t.Fatalf("failed to initialize git repository: %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %s", err)
	}

	tempFileName := "temp"
	tempFilePath := filepath.Join(tempDirPath, tempFileName)
	_, err = os.Create(tempFilePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %s", err)
	}

	err = os.WriteFile(tempFilePath, []byte("Hello world"), 0644)
	if err != nil {
		t.Fatalf("failed to write to temp file: %s", err)
	}

	_, err = w.Add(tempFileName)
	if err != nil {
		t.Fatalf("failed to add temp file to worktree: %s", err)
	}

	commit, err := w.Commit("firt commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})

	_, err = r.CommitObject(commit)
	if err != nil {
		t.Fatalf("failed to commit object %s", err)
	}

	h, err := r.Head()
	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	tags := []string{"2.0.0", "2.0.1", "3.0.0", "2.5.0", "0.0.2", "0.0.1", "0.1.0", "1.0.0"}

	for i, tag := range tags {
		r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
			Message: tag,
			Tagger:  &object.Signature{
				Name:  "Go Semver Release",
				Email: "ci@ci.ci",
				When:  time.Now().Add(time.Duration(i) * time.Hour),
			},
		})
	}

	commitAnalyzer, err := NewCommitAnalyzer(log.Default(), strings.NewReader(defaultReleaseRules))
	if err != nil {
		t.Fatalf("failed to create commit analyzer: %s", err)
	}

	latest, err := commitAnalyzer.FetchLatestSemverTag(r)
	if err != nil {
		t.Fatalf("faild to fetch latest semver tag: %s", err)
	}

	want := "3.0.0"
	if got := latest.Name; got != want  {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

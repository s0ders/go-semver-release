package commitanalyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/s0ders/go-semver-release/internal/releaserules"
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

func TestFetchLatestSemverTagWithNoTag(t *testing.T) {

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	rules, err := releaserules.NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to create rules: %s", err)
	}

	commitAnalyzer := NewCommitAnalyzer(rules, true)

	latest, err := commitAnalyzer.fetchLatestSemverTag(r)
	if err != nil {
		t.Fatalf("faild to fetch latest semver tag: %s", err)
	}

	want := "0.0.0"
	if got := latest.Name; got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

func TestFetchLatestSemverTagWithOneTag(t *testing.T) {

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	h, err := r.Head()
	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	tag := "1.0.0"

	r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Message: tag,
		Tagger: &object.Signature{
			Name:  "Go Semver Release",
			Email: "ci@ci.ci",
			When:  time.Now(),
		},
	})

	rules, err := releaserules.NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to create rules: %s", err)
	}

	commitAnalyzer := NewCommitAnalyzer(rules, true)

	latest, err := commitAnalyzer.fetchLatestSemverTag(r)
	if err != nil {
		t.Fatalf("faild to fetch latest semver tag: %s", err)
	}

	want := tag
	if got := latest.Name; got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

func TestFetchLatestSemverTagWithMultipleTags(t *testing.T) {

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	h, err := r.Head()
	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	tags := []string{"2.0.0", "2.0.1", "3.0.0", "2.5.0", "0.0.2", "0.0.1", "0.1.0", "1.0.0"}

	for i, tag := range tags {
		r.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
			Message: tag,
			Tagger: &object.Signature{
				Name:  "Go Semver Release",
				Email: "ci@ci.ci",
				When:  time.Now().Add(time.Duration(i) * time.Hour),
			},
		})
	}

	rules, err := releaserules.NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to create rules: %s", err)
	}

	commitAnalyzer := NewCommitAnalyzer(rules, true)

	latest, err := commitAnalyzer.fetchLatestSemverTag(r)
	if err != nil {
		t.Fatalf("faild to fetch latest semver tag: %s", err)
	}

	want := "3.0.0"
	if got := latest.Name; got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

func TestComputeNewSemverNumberWithUntaggedRepositoryWithoutNewRelease(t *testing.T) {

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	rules, err := releaserules.NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to create rules: %s", err)
	}

	ca := NewCommitAnalyzer(rules, true)

	version, _, err := ca.ComputeNewSemver(r)
	if err != nil {
		t.Fatalf("failed to compute new semver number: %s", err)
	}

	want := "0.0.0"

	if got := version.String(); got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

func TestComputeNewSemverNumberWithUntaggedRepositoryWitPatchRelease(t *testing.T) {

	r, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	rules, err := releaserules.NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to create rules: %s", err)
	}

	ca := NewCommitAnalyzer(rules, true)

	version, _, err := ca.ComputeNewSemver(r)
	if err != nil {
		t.Fatalf("failed to compute new semver number: %s", err)
	}

	want := "0.0.1"

	if got := version.String(); got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

func TestComputeNewSemverNumberWithUntaggedRepositoryWitMinorRelease(t *testing.T) {

	r, repositoryPath, err := createGitRepository("feat: commit that triggers a minor release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	rules, err := releaserules.NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to create rules: %s", err)
	}

	ca := NewCommitAnalyzer(rules, true)

	version, _, err := ca.ComputeNewSemver(r)
	if err != nil {
		t.Fatalf("failed to compute new semver number: %s", err)
	}

	want := "0.1.0"

	if got := version.String(); got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}
}

func TestComputeNewSemverNumberWithUntaggedRepositoryWitMajorRelease(t *testing.T) {

	r, repositoryPath, err := createGitRepository("feat!: commit that triggers a major release")
	if err != nil {
		t.Fatalf("failed to create git repository: %s", err)
	}

	defer os.RemoveAll(repositoryPath)

	addCommit(r, "fix: added hello feature")

	if err != nil {
		t.Fatalf("failed to fetch head: %s", err)
	}

	rules, err := releaserules.NewReleaseRuleReader().Read("").Parse()
	if err != nil {
		t.Fatalf("failed to create rules: %s", err)
	}

	ca := NewCommitAnalyzer(rules, true)

	version, newRelease, err := ca.ComputeNewSemver(r)
	if err != nil {
		t.Fatalf("failed to compute new semver number: %s", err)
	}

	want := "1.0.1"

	if got := version.String(); got != want {
		t.Fatalf("got: %s want: %s", got, want)
	}

	if newRelease != true {
		t.Fatalf("got: %t want: %t", newRelease, true)
	}
}

func createGitRepository(firstCommitMessage string) (*git.Repository, string, error) {

	tempDirPath, err := os.MkdirTemp("", "commitanalyzer-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	r, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize git repository: %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get worktree: %s", err)
	}

	tempFileName := "temp"
	tempFilePath := filepath.Join(tempDirPath, tempFileName)
	_, err = os.Create(tempFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp file: %s", err)
	}

	err = os.WriteFile(tempFilePath, []byte("Hello world"), 0644)
	if err != nil {
		return nil, "", fmt.Errorf("failed to write to temp file: %s", err)
	}

	_, err = w.Add(tempFileName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to add temp file to worktree: %s", err)
	}

	commit, err := w.Commit(firstCommitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to create commit object %s", err)
	}

	_, err = r.CommitObject(commit)
	if err != nil {
		return nil, "", fmt.Errorf("failed to commit object %s", err)
	}

	return r, tempDirPath, nil
}

func addCommit(r *git.Repository, message string) error {
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("could not get worktree: %w", err)
	}

	tempDirPath, err := os.MkdirTemp("", "commit-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	defer os.RemoveAll(tempDirPath)

	tempFileName := "temp"
	tempFilePath := filepath.Join(tempDirPath, tempFileName)
	_, err = os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	err = os.WriteFile(tempFilePath, []byte("Hello world"), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	_, err = w.Add(tempFileName)
	if err != nil {
		return fmt.Errorf("failed to add temp file to worktree: %w", err)
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Go Semver Release",
			Email: "go-semver-release@ci.go",
			When:  time.Now(),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return nil
}
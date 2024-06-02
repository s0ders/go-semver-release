package parser

import (
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v2/internal/rule"
	"github.com/s0ders/go-semver-release/v2/internal/tag"
)

var (
	logger    = zerolog.New(io.Discard)
	tagger    = tag.NewTagger("foo", "foo")
	rules     = rule.Default
	signature = &object.Signature{
		Name:  "Go SemVer Release",
		Email: "go-semver@release.ci",
		When:  time.Now(),
	}
)

func TestParser_CommitTypeRegex(t *testing.T) {
	assert := assertion.New(t)

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

		assert.Equal(item.commitType, got, "commit type should be equal")
	}
}

func TestParser_BreakingChangeRegex(t *testing.T) {
	assert := assertion.New(t)

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

		assert.Equal(item.isBreaking, got, "breaking change should be equal")
	}
}

func TestParser_FetchLatestSemverTagWithNoTag(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	latest, err := FetchLatestSemverTag(r)
	if err != nil {
		t.Fatalf("fetching latest semver tag: %s", err)
	}

	assert.Nil(latest, "latest semver tag should be nil")
}

func TestParser_FetchLatestSemverTagWithOneTag(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	h, err := r.Head()
	if err != nil {
		t.Fatalf("fetching head: %s", err)
	}

	tagName := "1.0.0"

	_, err = r.CreateTag(tagName, h.Hash(), &git.CreateTagOptions{
		Message: tagName,
		Tagger:  signature,
	})

	if err != nil {
		t.Fatalf("creating tagName: %s", err)
	}

	latest, err := FetchLatestSemverTag(r)
	if err != nil {
		t.Fatalf("fetching latest semver tagName: %s", err)
	}

	assert.Equal(tagName, latest.Name, "latest semver tagName should be equal")
}

func TestParser_FetchLatestSemverTagWithMultipleTags(t *testing.T) {
	assert := assertion.New(t)

	r, path, err := createGitRepository("commit that does not trigger a release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(path)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	h, err := r.Head()
	if err != nil {
		t.Fatalf("fetching head: %s", err)
	}

	tags := []string{"2.0.0", "2.0.1", "3.0.0", "2.5.0", "0.0.2", "0.0.1", "0.1.0", "1.0.0"}

	for _, v := range tags {
		_, err = r.CreateTag(v, h.Hash(), &git.CreateTagOptions{
			Message: v,
			Tagger:  signature,
		})
		if err != nil {
			t.Fatalf("creating tag: %s", err)
		}
	}

	latest, err := FetchLatestSemverTag(r)
	if err != nil {
		t.Fatalf("fetching latest semver tag: %s", err)
	}

	want := "3.0.0"
	assert.Equal(want, latest.Name, "latest semver tag should be equal")
}

func TestParser_ComputeNewSemverNumberWithUntaggedRepositoryWithoutNewRelease(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	if err != nil {
		t.Fatalf("creating Git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	version, _, err := parser.ComputeNewSemver(r)
	if err != nil {
		t.Fatalf("computing new semver: %s", err)
	}

	want := "0.0.0"

	assert.Equal(want, version.String(), "version should be equal")
}

func TestParser_ComputeNewSemverNumberWithUntaggedRepositoryWitPatchRelease(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	version, _, err := parser.ComputeNewSemver(r)
	if err != nil {
		t.Fatalf("computing new semver: %s", err)
	}

	want := "0.0.1"
	assert.Equal(want, version.String(), "version should be equal")
}

func TestParser_UnknownReleaseType(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("fix: commit that trigger an unknown release")
	if err != nil {
		t.Fatalf("creating Git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	invalidRules := rule.Rules{Mapped: map[string]string{"fix": "unknown"}}

	parser := New(logger, tagger, invalidRules)

	_, _, err = parser.ComputeNewSemver(r)
	assert.Error(err, "should have been failed trying to compute semver")
}

func TestParser_ComputeNewSemverNumberOnUntaggedRepositoryWitMinorRelease(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("feat: commit that triggers a minor release")
	if err != nil {
		t.Fatalf("creating Git repository: %s", err)
	}

	defer func(path string) {
		err := os.RemoveAll(repositoryPath)
		assert.NoError(err, "should have able to remove git repository")
	}(repositoryPath)

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	version, _, err := parser.ComputeNewSemver(r)
	if err != nil {
		t.Fatalf("computing new semver: %s", err)
	}

	want := "0.1.0"
	assert.Equal(want, version.String(), "version should be equal")
}

func TestParser_ComputeNewSemverNumberOnUntaggedRepositoryWithMajorRelease(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("feat!: commit that triggers a major release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	_, err = addCommit(r, "fix: added hello feature")
	if err != nil {
		t.Fatalf("adding commit: %s", err)
	}

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	version, newRelease, err := parser.ComputeNewSemver(r)
	assert.NoError(err, "should have been able to compute newsemver")

	want := "1.0.1"

	assert.Equal(want, version.String(), "version should be equal")
	assert.Equal(true, newRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemverOnUninitializedRepository(t *testing.T) {
	assert := assertion.New(t)

	dir, err := os.MkdirTemp("", "parser-*")
	if err != nil {
		t.Fatalf("creating temporary directory: %s", err)
	}

	defer func() {
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("removing temporary directory: %s", err)
		}
	}()

	repository, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("initializing git repository: %s", err)
	}

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	_, _, err = parser.ComputeNewSemver(repository)
	assert.ErrorContains(err, "reference not found", "should have been failed trying to fetch latest semver tag from uninitialized repository")
}

func TestParser_ComputeNewSemverOnRepositoryWithNoHead(t *testing.T) {
	assert := assertion.New(t)

	tempDirPath, err := os.MkdirTemp("", "tag-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repository, err := git.PlainInit(tempDirPath, false)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	parser := New(logger, tagger, rules)

	_, _, err = parser.ComputeNewSemver(repository)
	assert.Error(err, "should have been failed trying to compute new semver from repository with no HEAD")
}

func TestParser_ComputeNewSemverWithBuildMetadata(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("feat!: commit that triggers a major release")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(repositoryPath)
		if err != nil {
			t.Fatalf("removing git repository: %s", err)
		}
	}()

	_, err = addCommit(r, "fix: added hello feature")
	if err != nil {
		t.Fatalf("adding commit: %s", err)
	}

	parser := New(logger, tagger, rules, WithReleaseBranch("master"), WithBuildMetadata("metadata"))

	version, newRelease, err := parser.ComputeNewSemver(r)
	if err != nil {
		t.Fatalf("computing new semver: %s", err)
	}

	want := "1.0.1+metadata"

	assert.Equal(want, version.String(), "version should be equal")
	assert.Equal(true, newRelease, "boolean should be equal")
}

func TestParser_ShortMessage(t *testing.T) {
	assert := assertion.New(t)

	msg := "This is a very long commit message that is over fifty character"
	short := shortenMessage(msg)

	expected := "This is a very long commit message that is over..."

	assert.Equal(expected, short, "short message should be equal")
}

func TestParser_TagPointsToCommitOnBranch(t *testing.T) {
	assert := assertion.New(t)

	// Create Git repository with a commit on "master"
	repo, path, err := createGitRepository("first commit")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(path)
		if err != nil {
			t.Fatalf("removing temp dir: %s", err)
		}
	}()

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Create a new branch "rc"
	rcBranchName := "rc"
	rcBranchRef := plumbing.NewBranchReferenceName(rcBranchName)
	err = w.Checkout(&git.CheckoutOptions{
		Branch: rcBranchRef,
		Create: true,
	})
	if err != nil {
		t.Fatalf("checking out branch: %s", err)
	}

	// Create a new commits on "rc"
	rcCommitHash, err := addCommit(repo, "feat: ...")
	if err != nil {
		t.Fatalf("adding commit: %s", err)
	}

	// Create a tag that points to that commit
	tagRef, err := repo.CreateTag("v0.0.1", rcCommitHash, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Go SemVer Release",
			Email: "go-semver@release.ci",
			When:  time.Now(),
		},
		Message: "v0.0.1",
	})
	if err != nil {
		t.Fatalf("creating tag: %s", err)
	}

	tagObject, err := repo.TagObject(tagRef.Hash())
	if err != nil {
		t.Fatalf("getting tag object: %s", err)
	}

	// Tests if the commit pointed by that tag is reachable from branch "rc"
	got, err := TagPointsToCommitOnBranch(repo, tagObject, rcBranchName)

	assert.Equal(true, got, "tag points to commit on a branch that does not point to a commit")
}

func TestParser_TagDoesNotPointToCommitOnBranch(t *testing.T) {
	assert := assertion.New(t)

	// Create Git repository with a commit on "master"
	repo, path, err := createGitRepository("first commit")
	if err != nil {
		t.Fatalf("creating git repository: %s", err)
	}

	defer func() {
		err = os.RemoveAll(path)
		if err != nil {
			t.Fatalf("removing temp dir: %s", err)
		}
	}()

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Create a new branch "rc"
	rcBranchName := "rc"
	rcBranchRef := plumbing.NewBranchReferenceName(rcBranchName)
	err = w.Checkout(&git.CheckoutOptions{
		Branch: rcBranchRef,
		Create: true,
	})
	if err != nil {
		t.Fatalf("checking out branch: %s", err)
	}

	// Create two new commits on "rc"
	for i := 0; i < 2; i++ {
		_, err = addCommit(repo, fmt.Sprintf("feat: new commit (%d)", i))
		if err != nil {
			t.Fatalf("adding commit: %s", err)
		}
	}

	// Checkout back to "master"
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.Master,
	})
	if err != nil {
		t.Fatalf("Failed to checkout 'master' branch: %v", err)
	}

	// Create a commit on "master"
	masterCommitHash, err := addCommit(repo, "fix: some fix")

	// Create a tag that points to that commit
	tagRef, err := repo.CreateTag("v0.0.1", masterCommitHash, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Go SemVer Release",
			Email: "go-semver@release.ci",
			When:  time.Now(),
		},
		Message: "v0.0.1",
	})
	if err != nil {
		t.Fatalf("creating tag: %s", err)
	}

	tagObject, err := repo.TagObject(tagRef.Hash())
	if err != nil {
		t.Fatalf("getting tag object: %s", err)
	}

	// Tests if the commit pointed by that tag is reachable from branch "rc"
	got, err := TagPointsToCommitOnBranch(repo, tagObject, rcBranchName)

	assert.Equal(false, got, "tag points to commit on a branch that does not point to a commit")
}

func createGitRepository(commitMsg string) (*git.Repository, string, error) {
	dirPath, err := os.MkdirTemp("", "parser-*")

	repository, err := git.PlainInit(dirPath, false)
	if err != nil {
		return nil, "", fmt.Errorf("initializing git repository: %s", err)
	}

	_, err = addCommit(repository, commitMsg)
	if err != nil {
		return nil, "", fmt.Errorf("adding commit: %s", err)
	}

	return repository, dirPath, nil
}

func addCommit(repo *git.Repository, message string) (plumbing.Hash, error) {
	w, err := repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("fetching worktree: %s", err)
	}

	fsRoot := w.Filesystem.Root()

	fileName := fmt.Sprintf("file_%d.txt", time.Now().UnixNano())
	filePath := filepath.Join(fsRoot, fileName)

	err = os.WriteFile(filePath, []byte(message), 0644)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	_, err = w.Add(fileName)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("adding file to worktree: %s", err)
	}
	commitHash, err := w.Commit(message, &git.CommitOptions{
		Author: signature,
	})
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("commiting file: %s", err)
	}

	return commitHash, nil
}

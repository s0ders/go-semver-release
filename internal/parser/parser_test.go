package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"
	assertion "github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v2/internal/rule"
	"github.com/s0ders/go-semver-release/v2/internal/semver"
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

func TestParser_FetchLatestSemverTag_NoTag(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	latest, err := FetchLatestSemverTag(repository)
	checkErr(t, "fetching latest semver tag", err)

	assert.Nil(latest, "latest semver tag should be nil")
}

func TestParser_FetchLatestSemverTag_OneTag(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	head, err := repository.Head()
	checkErr(t, "fetching head", err)

	tagName := "1.0.0"

	_, err = repository.CreateTag(tagName, head.Hash(), &git.CreateTagOptions{
		Message: tagName,
		Tagger:  signature,
	})
	checkErr(t, "creating tag", err)

	latest, err := FetchLatestSemverTag(repository)
	checkErr(t, "fetching latest semver tag", err)

	assert.Equal(tagName, latest.Name, "latest semver tagName should be equal")
}

func TestParser_FetchLatestSemverTag_MultipleTags(t *testing.T) {
	assert := assertion.New(t)

	repository, path, err := createGitRepository("commit that does not trigger a release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(path)
	})

	head, err := repository.Head()
	if err != nil {
		t.Fatalf("fetching head: %s", err)
	}

	tags := []string{"2.0.0", "2.0.1", "3.0.0", "2.5.0", "0.0.2", "0.0.1", "0.1.0", "1.0.0"}

	for _, v := range tags {
		_, err = repository.CreateTag(v, head.Hash(), &git.CreateTagOptions{
			Message: v,
			Tagger:  signature,
		})
		checkErr(t, "creating tag", err)
	}

	latest, err := FetchLatestSemverTag(repository)
	checkErr(t, "fetching latest semver tag", err)

	want := "3.0.0"
	assert.Equal(want, latest.Name, "latest semver tag should be equal")
}

func TestParser_ComputeNewSemver_UntaggedRepository_NoRelease(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("commit that does not trigger a release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	output, err := parser.ComputeNewSemver(repository)
	checkErr(t, "computing new semver", err)

	want := "0.0.0"

	assert.Equal(want, output.Semver.String(), "version should be equal")
}

func TestParser_ComputeNewSemver_UntaggedRepository_PatchRelease(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger a patch release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	output, err := parser.ComputeNewSemver(repository)
	checkErr(t, "computing new semver", err)

	want := "0.0.1"
	assert.Equal(want, output.Semver.String(), "version should be equal")
}

func TestParser_ComputeNewSemver_UntaggedRepository_MinorRelease(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("feat: commit that triggers a minor release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	output, err := parser.ComputeNewSemver(repository)
	checkErr(t, "computing new semver", err)

	want := "0.1.0"
	assert.Equal(want, output.Semver.String(), "version should be equal")
}

func TestParser_ComputeNewSemver_UntaggedRepository_MajorRelease(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("feat!: commit that triggers a major release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	_, err = addCommit(repository, "fix: added hello feature")
	checkErr(t, "adding commit", err)

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	output, err := parser.ComputeNewSemver(repository)
	checkErr(t, "computing new semver ", err)

	want := "1.0.1"

	assert.Equal(want, output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_UnknownReleaseType(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("fix: commit that trigger an unknown release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	invalidRules := rule.Rules{Map: map[string]string{"fix": "unknown"}}

	parser := New(logger, tagger, invalidRules)

	_, err = parser.ComputeNewSemver(repository)
	assert.Error(err, "should have been failed trying to compute semver")
}

func TestParser_ComputeNewSemver_UninitializedRepository(t *testing.T) {
	assert := assertion.New(t)

	tempPath, err := os.MkdirTemp("", "parser-*")
	checkErr(t, "creating temporary directory", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(tempPath)
	})

	repository, err := git.PlainInit(tempPath, false)
	checkErr(t, "initializing repository", err)

	parser := New(logger, tagger, rules, WithReleaseBranch("master"))

	_, err = parser.ComputeNewSemver(repository)
	assert.ErrorIs(err, plumbing.ErrReferenceNotFound)
}

// TODO: how is this different from test above ?
func TestParser_ComputeNewSemver_RepositoryWithNoHead(t *testing.T) {
	assert := assertion.New(t)

	tempPath, err := os.MkdirTemp("", "parser-*")
	checkErr(t, "creating temporary directory", err)

	repository, err := git.PlainInit(tempPath, false)
	checkErr(t, "initializing repository", err)

	parser := New(logger, tagger, rules)

	_, err = parser.ComputeNewSemver(repository)
	assert.Error(err, "should have been failed trying to compute new semver from repository with no HEAD")
}

func TestParser_ComputeNewSemver_BuildMetadata(t *testing.T) {
	assert := assertion.New(t)

	r, repositoryPath, err := createGitRepository("feat: commit that triggers a major release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	parser := New(logger, tagger, rules, WithReleaseBranch("master"), WithBuildMetadata("metadata"))

	output, err := parser.ComputeNewSemver(r)
	checkErr(t, "computing new semver", err)

	want := semver.Semver{
		Major:         0,
		Minor:         1,
		Patch:         0,
		BuildMetadata: "metadata",
	}

	assert.Equal(want.String(), output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ComputeNewSemver_Prerelease(t *testing.T) {
	assert := assertion.New(t)

	repository, repositoryPath, err := createGitRepository("feat: commit that triggers a major release")
	checkErr(t, "creating repository", err)

	t.Cleanup(func() {
		_ = os.RemoveAll(repositoryPath)
	})

	prereleaseID := "rc"

	parser := New(logger, tagger, rules, WithReleaseBranch("master"), WithPrereleaseMode(true), WithPrereleaseIdentifier(prereleaseID))

	output, err := parser.ComputeNewSemver(repository)
	checkErr(t, "computing new semver", err)

	want := semver.Semver{
		Major:      0,
		Minor:      1,
		Patch:      0,
		Prerelease: prereleaseID,
	}

	assert.Equal(want.String(), output.Semver.String(), "version should be equal")
	assert.Equal(true, output.NewRelease, "boolean should be equal")
}

func TestParser_ShortMessage(t *testing.T) {
	assert := assertion.New(t)

	msg := "This is a very long commit message that is over fifty character"
	short := shortenMessage(msg)

	expected := "This is a very long commit message that is over..."

	assert.Equal(expected, short, "short message should be equal")
}

func createGitRepository(commitMsg string) (*git.Repository, string, error) {
	dirPath, err := os.MkdirTemp("", "parser-*")
	if err != nil {
		return nil, "", fmt.Errorf("creating temporary directory: %s", err)
	}

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

	signature.When = time.Now()

	commitHash, err := w.Commit(message, &git.CommitOptions{
		Author: signature,
	})
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("committing file: %s", err)
	}

	return commitHash, nil
}

func checkErr(t *testing.T, msg string, err error) {
	if err != nil {
		t.Fatalf("%s: %s", msg, err.Error())
	}
}

package commit

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/s0ders/go-semver-release/v6/internal/gittest"
	assertion "github.com/stretchr/testify/assert"
)

func TestWalker_WithMerges(t *testing.T) {
	assert := assertion.New(t)

	testRepository, err := gittest.NewRepository()
	checkErr(t, err, "creating sample repository")

	defer func() {
		err = os.RemoveAll(testRepository.Path)
		checkErr(t, err, "removing repository")
	}()

	type e = []string
	steps := []gittest.Step{
		gittest.NewCommitStep("main", "main-1"),
		gittest.NewCallbackStep("", e{
			"main-1",
			"First commit",
		}),

		gittest.NewCommitStep("beta", "beta-1"),
		gittest.NewCommitStep("main", "main-2"),
		gittest.NewCommitStep("beta", "beta-2"),
		gittest.NewCommitStep("beta", "beta-3"),
		gittest.NewCallbackStep("main", e{
			"main-2",
			"main-1",
			"First commit",
		}),
		gittest.NewCallbackStep("beta", e{
			"beta-3",
			"beta-2",
			"beta-1",
			"main-1",
			"First commit",
		}),

		gittest.NewCommitStep("main", "main-3"),
		gittest.NewCommitStep("beta", "beta-4"),
		gittest.NewCommitStep("main", "main-4"),

		gittest.NewMergeStep("main", "beta", false),

		gittest.NewCommitStep("main", "main-5"),
		gittest.NewCommitStep("main", "main-6"),

		gittest.NewCallbackStep("", e{
			"main-6",
			"main-5",
			"Merge branch 'beta'\n",
			"beta-4",
			"beta-3",
			"beta-2",
			"beta-1",
			"main-4",
			"main-3",
			"main-2",
			"main-1",
			"First commit",
		}),
	}

	var startCommit *object.Commit
	err = gittest.ExecuteSteps(testRepository, steps, func(expected e) error {
		head, err := testRepository.Head()
		if err != nil {
			return fmt.Errorf("fetching head: %w", err)
		}

		c, err := testRepository.CommitObject(head.Hash())
		if err != nil {
			return fmt.Errorf("fetching commit: %w", err)
		}

		startCommit = c

		w := NewWalker(startCommit)

		actual := []string{}
		err = w.ForEach(func(c *object.Commit) error {
			actual = append(actual, strings.Split(c.Message, ":")[0])
			return nil
		})
		checkErr(t, err, "traverse commits")
		assert.Equal(expected, actual)

		return nil
	})
	checkErr(t, err, "execute test steps")
}

func checkErr(t *testing.T, err error, message string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", message, err)
	}
}

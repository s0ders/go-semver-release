package gittest

import (
	"fmt"
	"os"
	"os/exec"
)

type Step interface {
	// Execution method that will be called by ExecuteSteps
	exec(r *TestRepository) error
}

type branchStep struct {
	// Which branch to checkout before running the command
	// If branch is "" the command will be executed on the current branch
	branch string
}

type CallbackStep[E any] struct {
	branchStep

	// Optional expected to be used in callbacks assert.
	expected E

	// Callback function that will be called with the expected parameter.
	cb func(E) error
}

type CheckoutStep struct {
	branchStep

	// Name of the (new) branch
	name string

	// Whenever just checking out branch or create a new one with given name from HEAD
	create bool
}

type CommitStep struct {
	branchStep

	// Which commit type to prepend. E.g. `feat`, `fix`...
	commitType string
}

type CommitWithFileStep struct {
	CommitStep

	// File to be created for this commit.
	filePath string
}

type MergeStep struct {
	branchStep

	// Source branch to merge into current HEAD.
	source string

	// Whenever only to allow a fast-forward merge.
	fastForward bool
}

type OctopusMergeStep struct {
	branchStep

	// Source branches to merge into current HEAD.
	sources []string
}

type TagStep struct {
	branchStep

	// Name to be used for the new tag.
	name string
}

// NewCallbackStep creates a new merge step.
// This step holds a callback that can be called to assert the current state.
func NewCallbackStep[E any](branch string, expected E) *CallbackStep[E] {
	return &CallbackStep[E]{
		branchStep: branchStep{
			branch: branch,
		},
		expected: expected,
	}
}

// NewCommitStep creates a new commit step.
// This step creates a new commit to the current or provided branch
// with the given commit type (e.g. `feat`, `fix`...).
func NewCommitStep(branch string, commitType string) *CommitStep {
	return &CommitStep{
		branchStep: branchStep{
			branch: branch,
		},
		commitType: commitType,
	}
}

// NewCheckoutStep creates a new checkout step
// This step checkouts a branch or optionally forces creation of a new branch on the current HEAD
func NewCheckoutStep(branch string, name string, create bool) *CheckoutStep {
	return &CheckoutStep{
		branchStep: branchStep{
			branch: branch,
		},
		name:   name,
		create: create,
	}
}

// NewCommitWithFileStep creates a new commit step.
// This step creates a new commit to the current or provided branch
// with the given commit type (e.g. `feat`, `fix`...) and creates a dummy
// file on the provided file path.
func NewCommitWithFileStep(branch string, commitType string, filePath string) *CommitWithFileStep {
	return &CommitWithFileStep{
		CommitStep: CommitStep{
			branchStep: branchStep{
				branch: branch,
			},
			commitType: commitType,
		},
		filePath: filePath,
	}
}

// NewMergeStep creates a new merge step.
// This step merges a source branch into another branch or to the current HEAD if no
// branch is provided.
func NewMergeStep(branch string, source string, fastForward bool) *MergeStep {
	return &MergeStep{
		branchStep: branchStep{
			branch: branch,
		},
		source:      source,
		fastForward: fastForward,
	}
}

// NewOctopusMergeStep creates a new octopus merge step.
// This step merges multiple source branches into another branch or to the current HEAD if no
// branch is provided.
func NewOctopusMergeStep(branch string, sources []string) *OctopusMergeStep {
	return &OctopusMergeStep{
		branchStep: branchStep{
			branch: branch,
		},
		sources: sources,
	}
}

// NewTagStep creates a new merge step.
// This step greats a new tag with the given name on the given branch.
func NewTagStep(branch string, name string) *TagStep {
	return &TagStep{
		branchStep: branchStep{
			branch: branch,
		},
		name: name,
	}
}

// BranchStep.exec executes branch step.
func (s *branchStep) exec(r *TestRepository) error {
	if s.branch != "" {
		err := r.CheckoutOrCreateBranch(s.branch)
		if err != nil {
			return fmt.Errorf("checking out %s branch: %w", s.branch, err)
		}
	}
	return nil
}

// CommitStep.exec executes commit step.
func (s *CommitStep) exec(r *TestRepository) error {
	err := s.branchStep.exec(r)
	if err != nil {
		return nil
	}

	_, err = r.AddCommit(s.commitType)
	if err != nil {
		return err
	}
	return nil
}

// CheckoutStep.exec executes checkout step.
func (s *CheckoutStep) exec(r *TestRepository) error {
	err := s.branchStep.exec(r)
	if err != nil {
		return nil
	}

	err = r.CheckoutBranch(s.name, s.create)
	if err != nil {
		return fmt.Errorf("checkout branch: %w", err)
	}
	return nil
}

// CommitWithFileStep.exec executes commit with file step.
func (s *CommitWithFileStep) exec(r *TestRepository) error {
	err := s.branchStep.exec(r)
	if err != nil {
		return nil
	}

	_, err = r.AddCommitWithSpecificFile(s.commitType, s.filePath)
	if err != nil {
		return fmt.Errorf("creating commit with file: %w", err)
	}
	return nil
}

// MergeStep.exec executes merge step.
func (s *MergeStep) exec(r *TestRepository) error {
	err := s.branchStep.exec(r)
	if err != nil {
		return nil
	}

	args := []string{"merge"}
	if s.fastForward {
		args = append(args, "--ff-only")
	} else {
		args = append(args, "-X", "theirs")
	}
	args = append(args, s.source)
	return execMerge(r, args)
}

// MergeStep.exec executes merge step.
func (s *OctopusMergeStep) exec(r *TestRepository) error {
	err := s.branchStep.exec(r)
	if err != nil {
		return nil
	}

	args := []string{"merge", "-s", "octopus"}
	args = append(args, s.sources...)
	return execMerge(r, args)
}

// TagStep.exec executes tag step.
func (s *TagStep) exec(r *TestRepository) error {
	err := r.AddTagToBranch(s.name, s.branch)
	if err != nil {
		return fmt.Errorf("adding tag: %w", err)
	}
	return nil
}

// CallbackStep.exec executes tag step.
func (s *CallbackStep[E]) exec(r *TestRepository) error {
	err := s.branchStep.exec(r)
	if err != nil {
		return nil
	}

	if s.cb != nil {
		err := s.cb(s.expected)
		if err != nil {
			return err
		}
	}
	return nil
}

// execMerge will execute a git merge command
func execMerge(r *TestRepository, args []string) error {
	cmd := exec.Command("git", args...)

	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Go Semver Release",
		"GIT_AUTHOR_EMAIL=go-semver@release.ci",
		"GIT_COMMITTER_NAME=Go Semver Release",
		"GIT_COMMITTER_EMAIL=go-semver@release.ci",
	)
	cmd.Dir = r.Path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s : %w", string(output), err)
	}
	return nil
}

// ExecuteSteps executes steps on test repository.
func ExecuteSteps[E any](r *TestRepository, steps []Step, cb func(E) error) error {
	for _, step := range steps {
		if callbackStep, ok := step.(*CallbackStep[E]); ok {
			callbackStep.cb = cb
		}
		err := step.exec(r)
		if err != nil {
			return fmt.Errorf("executing step: %w", err)
		}
	}
	return nil
}

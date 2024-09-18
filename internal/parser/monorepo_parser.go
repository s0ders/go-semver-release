package parser

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/s0ders/go-semver-release/v4/internal/monorepo"
	"github.com/s0ders/go-semver-release/v4/internal/semver"
)

type ComputeProjectsNewSemverOutput struct {
	Project    monorepo.Project
	Semver     *semver.Semver
	CommitHash plumbing.Hash
	NewRelease bool
}

func (p *Parser) ComputeProjectsNewSemver(repository *git.Repository) ([]ComputeProjectsNewSemverOutput, error) {
	output := make([]ComputeProjectsNewSemverOutput, 0, len(p.projects))

	worktree, err := repository.Worktree()
	if err != nil {
		return output, fmt.Errorf("fetching worktree: %w", err)
	}

	if worktree == nil {
		return output, fmt.Errorf("no worktree, check that repository is initialized")
	}

	// Checkout to release branch
	releaseBranchRef := plumbing.NewBranchReferenceName(p.releaseBranch)
	branchCheckOutOpts := git.CheckoutOptions{
		Branch: releaseBranchRef,
		Force:  true,
	}

	err = worktree.Checkout(&branchCheckOutOpts)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return output, fmt.Errorf("branch %q does not exist: %w", p.releaseBranch, err)
		}
		return output, fmt.Errorf("checking out to release branch: %w", err)
	}

	latestSemverTags, err := p.FetchLatestSemverTagPerProjects(repository)
	if err != nil {
		return output, fmt.Errorf("fetching latest semver tag: %w", err)
	}

	for project, latestSemverTag := range latestSemverTags {

		projectOutput := ComputeProjectsNewSemverOutput{}

		var (
			latestSemver *semver.Semver
			history      []*object.Commit
			logOptions   git.LogOptions
		)

		if latestSemverTag == nil {
			p.logger.Debug().Msg("no previous tag, creating one")

			latestSemver = &semver.Semver{Major: 0, Minor: 0, Patch: 0}
		} else {
			p.logger.Debug().Str("tag", latestSemverTag.Name).Msg("latest semver tag found")

			latestSemver, err = semver.FromGitTag(latestSemverTag)
			if err != nil {
				return output, fmt.Errorf("building semver from git tag: %w", err)
			}

			latestSemverTagCommit, err := latestSemverTag.Commit()
			if err != nil {
				return output, fmt.Errorf("fetching latest semver tag commit: %w", err)
			}

			// Show all commit that are at least one second older than the latest one pointed by SemVer tag
			since := latestSemverTagCommit.Committer.When.Add(time.Second)
			logOptions.Since = &since
		}

		repositoryLogs, err := repository.Log(&logOptions)
		if err != nil {
			return output, fmt.Errorf("fetching commit history: %w", err)
		}

		// Create commit history
		_ = repositoryLogs.ForEach(func(c *object.Commit) error {
			history = append(history, c)
			return nil
		})

		// Sort commit history from oldest to most recent
		sort.Slice(history, func(i, j int) bool {
			return history[i].Committer.When.Before(history[j].Committer.When)
		})

		newRelease, commitHash, err := p.ParseHistoryForGivenProject(history, latestSemver, project)
		if err != nil {
			return output, fmt.Errorf("parsing commit history: %w", err)
		}

		latestSemver.BuildMetadata = p.buildMetadata

		projectOutput.Project = project
		projectOutput.Semver = latestSemver
		projectOutput.NewRelease = newRelease
		projectOutput.CommitHash = commitHash

		output = append(output, projectOutput)
	}

	return output, nil
}

// FetchLatestSemverTagPerProjects parses a Git repository to fetch the latest SemVer tag for each project, if any.
func (p *Parser) FetchLatestSemverTagPerProjects(repository *git.Repository) (map[monorepo.Project]*object.Tag, error) {
	tags, err := repository.TagObjects()
	if err != nil {
		return nil, fmt.Errorf("fetching tag objects: %w", err)
	}

	semverTags := make([]*object.Tag, 0)
	projectSemverTags := make(map[monorepo.Project][]*object.Tag)
	latestProjectSemver := make(map[monorepo.Project]*object.Tag)

	// Find all semver tags
	_ = tags.ForEach(func(tag *object.Tag) error {
		if semver.Regex.MatchString(tag.Name) {
			semverTags = append(semverTags, tag)
		}
		return nil
	})

	// Sort every semver tags per project by checking if semver tags match the pattern "<project>-<semver>"
	for _, project := range p.projects {
		for _, semverTag := range semverTags {
			match, err := regexp.MatchString(fmt.Sprintf(`^%s\-.*`, project.Name), semverTag.Name)
			if err != nil {
				return nil, err
			}

			if match {
				projectSemverTags[project] = append(projectSemverTags[project], semverTag)
			}
		}
	}

	// Find latest semver per project
	for project, projectSemverTag := range projectSemverTags {

		var (
			latestSemver *semver.Semver
			latestTag    *object.Tag
		)

		for _, semverTag := range projectSemverTag {
			currentSemver, err := semver.FromGitTag(semverTag)
			if err != nil {
				return nil, fmt.Errorf("converting tag to semver: %w", err)
			}

			if latestSemver == nil || semver.Compare(latestSemver, currentSemver) == -1 {
				latestSemver = currentSemver
				latestTag = semverTag
			}
		}

		latestProjectSemver[project] = latestTag
	}

	return latestProjectSemver, nil
}

func (p *Parser) ParseHistoryForGivenProject(commits []*object.Commit, latestSemver *semver.Semver, project monorepo.Project) (bool, plumbing.Hash, error) {
	newRelease := false
	latestReleaseCommitHash := plumbing.Hash{}
	rulesMap := p.rules.Map

	if p.prereleaseMode {
		latestSemver.Prerelease = p.prereleaseIdentifier
	}

	for _, commit := range commits {
		if !conventionalCommitRegex.MatchString(commit.Message) {
			continue
		}

		containsProjectFiles, err := commitContainsProjectFiles(commit, project.Path)
		if err != nil {
			return false, latestReleaseCommitHash, fmt.Errorf("checking if commit contains project files: %w", err)
		}

		if !containsProjectFiles {
			continue
		}

		match := conventionalCommitRegex.FindStringSubmatch(commit.Message)
		breakingChange := match[3] == "!" || strings.Contains(match[0], "BREAKING CHANGE")
		commitType := match[1]
		shortHash := commit.Hash.String()[0:7]
		shortMessage := shortenMessage(commit.Message)

		if breakingChange {
			p.logger.Debug().Str("commit-hash", shortHash).Str("commit-message", shortMessage).Msg("breaking change found")
			latestSemver.BumpMajor()
			latestReleaseCommitHash = commit.Hash
			newRelease = true
			continue
		}

		releaseType, ok := rulesMap[commitType]

		if !ok {
			continue
		}

		switch releaseType {
		case "patch":
			latestSemver.BumpPatch()
		case "minor":
			latestSemver.BumpMinor()
		default:
			return false, latestReleaseCommitHash, fmt.Errorf("unknown release type %q", releaseType)
		}

		latestReleaseCommitHash = commit.Hash
		newRelease = true

		p.logger.Debug().Str("commit-hash", shortHash).Str("commit-message", shortMessage).Str("release-type", releaseType).Msg("new release found")
	}

	return newRelease, latestReleaseCommitHash, nil
}

// commitContainsProjectFiles checks if a given commit changes contain at least one file whose path belongs to the
// given project's path.
func commitContainsProjectFiles(commit *object.Commit, projectPath string) (bool, error) {
	regex, err := regexp.Compile(fmt.Sprintf("^%s", projectPath))
	if err != nil {
		return false, fmt.Errorf("compiling project's path regexp: %w", err)
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return false, fmt.Errorf("getting commit tree: %w", err)
	}

	parentCommit := commit.Parents()
	parentTree := &object.Tree{}
	if parentCommit != nil {
		parent, err := parentCommit.Next()
		if err == nil {
			parentTree, err = parent.Tree()
			if err != nil {
				return false, fmt.Errorf("getting parent tree: %w", err)
			}
		}
	}

	changes, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return false, fmt.Errorf("getting diff tree: %w", err)
	}

	for _, change := range changes {
		if regex.MatchString(filepath.Dir(change.To.Name)) {
			return true, nil
		}
	}

	return false, nil
}

// Package parser provides functions to parse a Git repository commit history.
//
// This package is used to compute the semantic version number from a formatted Git repository commit history. To do so,
// it expects the commit history to follow the Conventional Commits specification.
package parser

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"

	"github.com/s0ders/go-semver-release/v8/internal/appcontext"
	"github.com/s0ders/go-semver-release/v8/internal/branch"
	"github.com/s0ders/go-semver-release/v8/internal/monorepo"
	"github.com/s0ders/go-semver-release/v8/internal/semver"
)

type BumpType int

const (
	BumpNone BumpType = iota
	BumpPatch
	BumpMinor
	BumpMajor
)

var conventionalCommitRegex = regexp.MustCompile(`^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([\w\-.\\\/]+\))?(!)?: ([\w ]+[\s\S]*)`)

type Parser struct {
	ctx *appcontext.AppContext

	// The caches below are keyed by Git object hashes, which are content addresses, so they never go stale and can
	// safely be shared across runs and repositories.

	// commitKinds memoizes the conventional commit parsing of each commit message.
	commitKinds map[plumbing.Hash]commitKind
	// pathEntries memoizes, per root tree hash, the tree entry found at each monitored project path.
	pathEntries map[plumbing.Hash]map[string]pathEntry
	// commitTrees maps commit hashes to their root tree hash, filled while walking history, so that a commit's
	// parent tree can be found without reading the parent commit object again.
	commitTrees map[plumbing.Hash]plumbing.Hash
}

// commitKind is the result of parsing a commit message against the Conventional Commits specification.
type commitKind struct {
	conventional bool
	commitType   string
	breaking     bool
}

// pathEntry describes the tree entry found at a given path inside a root tree, if any.
type pathEntry struct {
	hash  plumbing.Hash
	mode  filemode.FileMode
	found bool
}

// resolvedTag is a semver tag resolved to its commit and parsed version. Tags are resolved once per run and then
// filtered per branch and project, instead of iterating and resolving every tag for each lookup.
type resolvedTag struct {
	name    string
	commit  *object.Commit
	version *semver.Version
}

// TagInfo holds information about a Git tag, supporting both annotated and lightweight tags.
type TagInfo struct {
	Name   string
	Commit *object.Commit
}

func New(ctx *appcontext.AppContext) *Parser {
	parser := &Parser{
		ctx:         ctx,
		commitKinds: make(map[plumbing.Hash]commitKind),
		pathEntries: make(map[plumbing.Hash]map[string]pathEntry),
		commitTrees: make(map[plumbing.Hash]plumbing.Hash),
	}

	return parser
}

type ComputeNewSemverOutput struct {
	Semver     *semver.Version
	Project    monorepo.Item
	Branch     string
	CommitHash plumbing.Hash
	NewRelease bool
}

// Run executes a parser on a repository and analyzes the given branches and projects contained inside the given
// AppContext.
func (p *Parser) Run(repository *git.Repository) ([]ComputeNewSemverOutput, error) {
	var output []ComputeNewSemverOutput

	// Tags are resolved a single time and shared by every branch and project.
	tags, err := resolveSemverTags(repository)
	if err != nil {
		return nil, fmt.Errorf("computing new semver: %w", err)
	}

	// Sort branches: stable branches first, then prerelease branches
	// This ensures prerelease branches can see stable releases when computing versions
	sortedBranches := sortBranches(p.ctx.BranchesCfg)

	for _, gitBranch := range sortedBranches {
		// Reachable commits only depend on the branch, compute them once for all projects.
		reachable, err := p.branchReachableCommits(repository, gitBranch)
		if err != nil {
			if len(p.ctx.MonorepositoryCfg) == 0 {
				return nil, fmt.Errorf("computing new semver: %w", err)
			}
			return nil, fmt.Errorf("computing project %q new semver: %w", p.ctx.MonorepositoryCfg[0].Name, err)
		}

		if len(p.ctx.MonorepositoryCfg) == 0 {
			result, err := p.computeNewSemver(repository, monorepo.Item{}, gitBranch, reachable, tags)
			if err != nil {
				return nil, fmt.Errorf("computing new semver: %w", err)
			}
			output = append(output, result)
			continue
		}

		for _, project := range p.ctx.MonorepositoryCfg {
			result, err := p.computeNewSemver(repository, project, gitBranch, reachable, tags)
			if err != nil {
				return nil, fmt.Errorf("computing project %q new semver: %w", project.Name, err)
			}
			output = append(output, result)
		}
	}

	return output, nil
}

// sortBranches returns a copy of branches sorted with stable branches first, then prerelease branches.
func sortBranches(branches []branch.Item) []branch.Item {
	sorted := make([]branch.Item, len(branches))
	copy(sorted, branches)

	sort.SliceStable(sorted, func(i, j int) bool {
		// Stable branches (Prerelease=false) come before prerelease branches
		if !sorted[i].Prerelease && sorted[j].Prerelease {
			return true
		}
		return false
	})

	return sorted
}

// ComputeNewSemver returns the next, if any, semantic version number from a given Git repository by parsing its commit
// history.
func (p *Parser) ComputeNewSemver(repository *git.Repository, project monorepo.Item, gitBranch branch.Item) (ComputeNewSemverOutput, error) {
	reachable, err := p.branchReachableCommits(repository, gitBranch)
	if err != nil {
		return ComputeNewSemverOutput{}, err
	}

	tags, err := resolveSemverTags(repository)
	if err != nil {
		return ComputeNewSemverOutput{}, fmt.Errorf("fetching latest semver tag: %w", err)
	}

	return p.computeNewSemver(repository, project, gitBranch, reachable, tags)
}

// branchReachableCommits resolves the given branch reference and walks every commit reachable from it.
func (p *Parser) branchReachableCommits(repository *git.Repository, gitBranch branch.Item) (*ReachableCommits, error) {
	// Get branch reference without checking out
	ref, err := p.getBranchReference(repository, gitBranch.Name)
	if err != nil {
		return nil, fmt.Errorf("getting reference for branch %q: %w", gitBranch.Name, err)
	}

	reachable, err := BuildReachableCommits(repository, ref)
	if err != nil {
		return nil, fmt.Errorf("building reachable commits: %w", err)
	}

	for _, c := range reachable.Commits {
		p.commitTrees[c.Hash] = c.TreeHash
	}

	return reachable, nil
}

func (p *Parser) computeNewSemver(
	repository *git.Repository,
	project monorepo.Item,
	gitBranch branch.Item,
	reachable *ReachableCommits,
	tags []resolvedTag,
) (ComputeNewSemverOutput, error) {
	output := ComputeNewSemverOutput{}

	if project.Name != "" {
		output.Project = project
	}

	latestSemverTag := latestSemverTag(tags, project, reachable.HashSet)

	var (
		latestSemver *semver.Version
		history      []*object.Commit
	)

	if latestSemverTag == nil {
		p.ctx.Logger.Debug().Msg("no previous tag, creating one")

		latestSemver = &semver.Version{Major: 0, Minor: 0, Patch: 0}
		// No tag means all commits are part of history. The slice is shared by every project of the branch, so
		// copy it before sorting.
		history = append(make([]*object.Commit, 0, len(reachable.Commits)), reachable.Commits...)
	} else {
		p.ctx.Logger.Debug().Str("tag", latestSemverTag.name).Msg("latest semver tag found")

		// The resolved tag is shared across branches and projects, copy its version before bumping it.
		v := *latestSemverTag.version
		latestSemver = &v

		// Filter commits that are newer than the tagged commit
		sinceTime := latestSemverTag.commit.Committer.When.Add(time.Second)
		for _, c := range reachable.Commits {
			if c.Committer.When.After(sinceTime) || c.Committer.When.Equal(sinceTime) {
				history = append(history, c)
			}
		}
	}

	// Sort commit history from oldest to most recent
	sort.Slice(history, func(i, j int) bool {
		return history[i].Committer.When.Before(history[j].Committer.When)
	})

	var newRelease bool
	var commitHash plumbing.Hash

	var maxBumpType BumpType

	for _, commit := range history {
		processCommitOutput, err := p.ProcessCommit(repository.Storer, commit, project)
		if err != nil {
			return output, fmt.Errorf("parsing commit history: %w", err)
		}

		if processCommitOutput.NewRelease {
			newRelease = true
			commitHash = processCommitOutput.CommitHash
			maxBumpType = max(maxBumpType, processCommitOutput.BumpType)
		}
	}

	// Handle version computation based on branch type
	if gitBranch.Prerelease {
		// For prerelease branches, compute the next prerelease version
		newVersion, err := p.computePrereleaseVersion(tags, project, reachable.HashSet, latestSemver, maxBumpType, gitBranch, newRelease)
		if err != nil {
			return output, fmt.Errorf("computing prerelease version: %w", err)
		}
		latestSemver = newVersion
	} else {
		// For stable branches, apply the standard bump
		switch maxBumpType {
		case BumpMajor:
			latestSemver.BumpMajor()
		case BumpMinor:
			latestSemver.BumpMinor()
		case BumpPatch:
			latestSemver.BumpPatch()
		}

		// Automatic promotion: if a stable branch has a prerelease tag as its latest,
		// promote it to stable by clearing the prerelease suffix.
		// This handles the case where a prerelease branch is merged into a stable branch.
		if latestSemver.HasPrerelease() {
			p.ctx.Logger.Debug().
				Str("from", latestSemver.String()).
				Msg("promoting prerelease to stable release")
			latestSemver.ClearPrerelease()
			if !newRelease {
				newRelease = true
			}
		}
	}

	latestSemver.Metadata = p.ctx.BuildMetadata

	output.Semver = latestSemver
	output.Branch = gitBranch.Name
	output.CommitHash = commitHash
	output.NewRelease = newRelease

	return output, nil
}

// computePrereleaseVersion computes the next prerelease version for a prerelease branch.
// It follows the standard semver prerelease numbering scheme: X.Y.Z-label.N
func (p *Parser) computePrereleaseVersion(
	tags []resolvedTag,
	project monorepo.Item,
	reachableHashSet map[plumbing.Hash]struct{},
	latestSemver *semver.Version,
	maxBumpType BumpType,
	gitBranch branch.Item,
	hasNewCommits bool,
) (*semver.Version, error) {
	prereleaseLabel := gitBranch.Name

	// If the latest tag is already a prerelease for this branch
	if latestSemver.HasPrerelease() && latestSemver.PrereleaseLabel == prereleaseLabel {
		if hasNewCommits {
			// Check if we need a new version series (breaking change or major/minor bump)
			// or just bump the prerelease number
			if maxBumpType == BumpMajor || maxBumpType == BumpMinor {
				// New commits warrant a version bump - compute new core version
				newVersion := &semver.Version{
					Major: latestSemver.Major,
					Minor: latestSemver.Minor,
					Patch: latestSemver.Patch,
				}
				switch maxBumpType {
				case BumpMajor:
					newVersion.BumpMajor()
				case BumpMinor:
					newVersion.BumpMinor()
				case BumpPatch:
					newVersion.BumpPatch()
				}
				newVersion.SetPrerelease(prereleaseLabel)
				return newVersion, nil
			}

			// For patch-level changes or just new commits, bump prerelease number
			newVersion := &semver.Version{
				Major:            latestSemver.Major,
				Minor:            latestSemver.Minor,
				Patch:            latestSemver.Patch,
				PrereleaseLabel:  prereleaseLabel,
				PrereleaseNumber: latestSemver.PrereleaseNumber,
			}
			newVersion.BumpPrerelease()
			return newVersion, nil
		}
		// No new commits, return as-is
		return latestSemver, nil
	}

	// Latest tag is not a prerelease (or is a different prerelease label)
	// We need to compute the next version based on the stable version

	// First, get the latest stable version
	latestStable := latestStableTag(tags, project, reachableHashSet)

	var baseVersion *semver.Version
	if latestStable == nil {
		// No stable version exists, start from 0.0.0
		baseVersion = &semver.Version{Major: 0, Minor: 0, Patch: 0}
	} else {
		baseVersion = latestStable
	}

	// Compute what the next stable version would be
	nextVersion := &semver.Version{
		Major: baseVersion.Major,
		Minor: baseVersion.Minor,
		Patch: baseVersion.Patch,
	}

	if hasNewCommits {
		switch maxBumpType {
		case BumpMajor:
			nextVersion.BumpMajor()
		case BumpMinor:
			nextVersion.BumpMinor()
		case BumpPatch:
			nextVersion.BumpPatch()
		}
	}

	// Check if there's already a prerelease for this version
	existingPrerelease := latestPrereleaseTag(tags, project, reachableHashSet, nextVersion, prereleaseLabel)

	if existingPrerelease != nil {
		// There's already a prerelease for this version, bump the number
		if hasNewCommits {
			existingPrerelease.BumpPrerelease()
		}
		return existingPrerelease, nil
	}

	// No existing prerelease, create a new one
	nextVersion.SetPrerelease(prereleaseLabel)
	return nextVersion, nil
}

type ProcessCommitOutput struct {
	NewRelease bool
	CommitHash plumbing.Hash
	BumpType   BumpType
}

// ProcessCommit parse a commit message and bump the latest semantic version accordingly.
func (p *Parser) ProcessCommit(s storer.EncodedObjectStorer, commit *object.Commit, project monorepo.Item) (ProcessCommitOutput, error) {
	output := ProcessCommitOutput{
		NewRelease: false,
		CommitHash: plumbing.ZeroHash,
		BumpType:   BumpNone,
	}

	kind := p.parseCommitMessage(commit)
	if !kind.conventional {
		return output, nil
	}

	if project.Name != "" {
		containsProjectFiles, err := p.commitContainsProjectFiles(s, commit, project)
		if err != nil {
			return output, fmt.Errorf("checking if commit contains project files: %w", err)
		}
		if !containsProjectFiles {
			return output, nil
		}
	}

	commitType := kind.commitType

	if kind.breaking {
		output.BumpType = BumpMajor
		output.NewRelease = true
		output.CommitHash = commit.Hash

		return output, nil
	}

	releaseType, ok := p.ctx.RulesCfg[commitType]
	if !ok {
		return output, nil
	}

	output.NewRelease = true
	output.CommitHash = commit.Hash

	switch releaseType {
	case "patch":
		output.BumpType = BumpPatch
	case "minor":
		output.BumpType = BumpMinor
	default:
		return output, fmt.Errorf("unknown release type %q", releaseType)
	}

	return output, nil
}

// parseCommitMessage parses a commit message against the Conventional Commits specification. Results are memoized
// per commit hash since the same commit is evaluated for every project and every branch containing it.
func (p *Parser) parseCommitMessage(commit *object.Commit) commitKind {
	if kind, ok := p.commitKinds[commit.Hash]; ok {
		return kind
	}

	var kind commitKind
	if match := conventionalCommitRegex.FindStringSubmatch(commit.Message); match != nil {
		kind = commitKind{
			conventional: true,
			commitType:   match[1],
			breaking:     match[3] == "!" || strings.Contains(commit.Message, "BREAKING CHANGE"),
		}
	}

	p.commitKinds[commit.Hash] = kind

	return kind
}

// resolveTagToCommit resolves a tag reference to its commit, handling both annotated and lightweight tags.
// For annotated tags, it dereferences the tag object to get the commit.
// For lightweight tags, the reference points directly to the commit.
func resolveTagToCommit(repository *git.Repository, ref *plumbing.Reference) (*object.Commit, error) {
	// Try to get as annotated tag first
	tagObject, err := repository.TagObject(ref.Hash())
	if err == nil {
		// It's an annotated tag, get the commit it points to
		return tagObject.Commit()
	}

	// It's a lightweight tag, the reference points directly to a commit
	return repository.CommitObject(ref.Hash())
}

// resolveSemverTags lists every semver tag of the repository, resolved to its commit and parsed. Tags that do not
// point to a commit (e.g., trees or blobs) are skipped.
func resolveSemverTags(repository *git.Repository) ([]resolvedTag, error) {
	tags, err := repository.Tags()
	if err != nil {
		return nil, fmt.Errorf("fetching tags: %w", err)
	}

	var resolved []resolvedTag

	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()

		if !semver.Regex.MatchString(tagName) {
			return nil
		}

		// Resolve tag to commit (works for both annotated and lightweight tags)
		tagCommit, err := resolveTagToCommit(repository, ref)
		if err != nil {
			// Tag might point to a tree or blob; skip it
			return nil
		}

		version, err := semver.NewFromString(tagName)
		if err != nil {
			return fmt.Errorf("converting tag to semver: %w", err)
		}

		resolved = append(resolved, resolvedTag{name: tagName, commit: tagCommit, version: version})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("looping over tags: %w", err)
	}

	return resolved, nil
}

// candidateTags yields the tags belonging to the given project whose commit is reachable from the branch.
func candidateTags(tags []resolvedTag, project monorepo.Item, reachableHashSet map[plumbing.Hash]struct{}, yield func(t *resolvedTag)) {
	prefix := project.Name + "-"

	for i := range tags {
		t := &tags[i]

		if project.Name != "" && !strings.HasPrefix(t.name, prefix) {
			continue
		}

		if _, isReachable := reachableHashSet[t.commit.Hash]; !isReachable {
			continue
		}

		yield(t)
	}
}

// latestSemverTag returns the tag with the highest semver among the given project's tags reachable from the branch.
func latestSemverTag(tags []resolvedTag, project monorepo.Item, reachableHashSet map[plumbing.Hash]struct{}) *resolvedTag {
	var latest *resolvedTag

	candidateTags(tags, project, reachableHashSet, func(t *resolvedTag) {
		if latest == nil || semver.Compare(latest.version, t.version) == -1 {
			latest = t
		}
	})

	return latest
}

// latestStableTag returns a copy of the highest non-prerelease version among the given project's reachable tags.
func latestStableTag(tags []resolvedTag, project monorepo.Item, reachableHashSet map[plumbing.Hash]struct{}) *semver.Version {
	var latest *semver.Version

	candidateTags(tags, project, reachableHashSet, func(t *resolvedTag) {
		if t.version.HasPrerelease() {
			return
		}

		if latest == nil || semver.Compare(latest, t.version) == -1 {
			latest = t.version
		}
	})

	return copyVersion(latest)
}

// latestPrereleaseTag returns a copy of the highest prerelease version with the given label and core version among
// the given project's reachable tags.
func latestPrereleaseTag(tags []resolvedTag, project monorepo.Item, reachableHashSet map[plumbing.Hash]struct{}, coreVersion *semver.Version, label string) *semver.Version {
	var latest *semver.Version

	candidateTags(tags, project, reachableHashSet, func(t *resolvedTag) {
		// Must be a prerelease with matching label and the same core version (major.minor.patch)
		if !t.version.HasPrerelease() || t.version.PrereleaseLabel != label || !t.version.SameCoreVersion(coreVersion) {
			return
		}

		if latest == nil || semver.Compare(latest, t.version) == -1 {
			latest = t.version
		}
	})

	return copyVersion(latest)
}

func copyVersion(v *semver.Version) *semver.Version {
	if v == nil {
		return nil
	}

	c := *v

	return &c
}

// FetchLatestSemverTag parses a Git repository to fetch the tag corresponding to the highest semantic version number
// among all tags reachable from the given branch. The reachableHashSet should contain all commit hashes reachable
// from the branch reference. Supports both annotated and lightweight tags.
func (p *Parser) FetchLatestSemverTag(repository *git.Repository, project monorepo.Item, reachableHashSet map[plumbing.Hash]struct{}) (*TagInfo, error) {
	tags, err := resolveSemverTags(repository)
	if err != nil {
		return nil, err
	}

	latest := latestSemverTag(tags, project, reachableHashSet)
	if latest == nil {
		return nil, nil
	}

	return &TagInfo{Name: latest.name, Commit: latest.commit}, nil
}

// FetchLatestStableTag fetches the latest stable (non-prerelease) semver tag reachable from the branch.
// Supports both annotated and lightweight tags.
func (p *Parser) FetchLatestStableTag(repository *git.Repository, project monorepo.Item, reachableHashSet map[plumbing.Hash]struct{}) (*semver.Version, error) {
	tags, err := resolveSemverTags(repository)
	if err != nil {
		return nil, err
	}

	return latestStableTag(tags, project, reachableHashSet), nil
}

// FetchLatestPrereleaseTag fetches the latest prerelease tag for a given core version and prerelease label.
// For example, if coreVersion is 1.2.0 and label is "rc", it will find the highest rc.N tag for 1.2.0.
// Supports both annotated and lightweight tags.
func (p *Parser) FetchLatestPrereleaseTag(repository *git.Repository, project monorepo.Item, reachableHashSet map[plumbing.Hash]struct{}, coreVersion *semver.Version, label string) (*semver.Version, error) {
	tags, err := resolveSemverTags(repository)
	if err != nil {
		return nil, err
	}

	return latestPrereleaseTag(tags, project, reachableHashSet, coreVersion, label), nil
}

// ReachableCommits holds the results of walking reachable commits from a branch reference.
type ReachableCommits struct {
	HashSet map[plumbing.Hash]struct{}
	Commits []*object.Commit
}

// BuildReachableCommits walks all commits reachable from the given reference and returns
// both a hash set (for tag reachability checks) and the commit objects (for history processing).
func BuildReachableCommits(repository *git.Repository, fromRef *plumbing.Reference) (*ReachableCommits, error) {
	result := &ReachableCommits{
		HashSet: make(map[plumbing.Hash]struct{}),
	}

	logIter, err := repository.Log(&git.LogOptions{From: fromRef.Hash()})
	if err != nil {
		return nil, fmt.Errorf("creating log iterator: %w", err)
	}

	err = logIter.ForEach(func(c *object.Commit) error {
		result.HashSet[c.Hash] = struct{}{}
		result.Commits = append(result.Commits, c)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}

	return result, nil
}

// getBranchReference attempts to get a reference to a branch, trying remote-tracking branches first,
// then falling back to local branches if the remote reference is not found.
func (p *Parser) getBranchReference(repository *git.Repository, branchName string) (*plumbing.Reference, error) {
	// Try remote-tracking branch first (refs/remotes/origin/branchName)
	remoteBranchRef := plumbing.NewRemoteReferenceName(p.ctx.RemoteName, branchName)
	ref, err := repository.Reference(remoteBranchRef, true)
	if err == nil {
		p.ctx.Logger.Debug().
			Str("branch", branchName).
			Str("ref", remoteBranchRef.String()).
			Msg("using remote-tracking branch reference")
		return ref, nil
	}

	// If remote reference not found, try local branch (refs/heads/branchName)
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		localBranchRef := plumbing.NewBranchReferenceName(branchName)
		ref, err = repository.Reference(localBranchRef, true)
		if err == nil {
			p.ctx.Logger.Debug().
				Str("branch", branchName).
				Str("ref", localBranchRef.String()).
				Msg("using local branch reference")
			return ref, nil
		}

		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, fmt.Errorf("branch %q not found in remote or local references: %w", branchName, err)
		}
	}

	return nil, fmt.Errorf("getting reference for branch %q: %w", branchName, err)
}

// commitContainsProjectFiles checks if a given commit change contains at least one file whose path belongs to the
// given project's path.
//
// Instead of computing a full recursive tree diff, it relies on Git trees being Merkle trees: a commit changes a file
// under a project path if and only if the tree entry found at that path (hash and mode) differs between the commit's
// tree and its first parent's tree. This only requires resolving a handful of tree objects per commit, and those
// lookups are memoized per root tree hash, which every commit shares with its children.
func (p *Parser) commitContainsProjectFiles(s storer.EncodedObjectStorer, commit *object.Commit, project monorepo.Item) (bool, error) {
	paths := project.Paths
	if project.Path != "" {
		paths = append([]string{project.Path}, paths...)
	}

	if len(paths) == 0 {
		return false, nil
	}

	var parentTree plumbing.Hash
	if len(commit.ParentHashes) > 0 {
		parentHash := commit.ParentHashes[0]

		treeHash, ok := p.commitTrees[parentHash]
		if !ok {
			parent, err := object.GetCommit(s, parentHash)
			if err != nil {
				return false, fmt.Errorf("getting parent commit: %w", err)
			}
			treeHash = parent.TreeHash
			p.commitTrees[parentHash] = treeHash
		}

		parentTree = treeHash
	}

	for _, projectPath := range paths {
		current, err := p.entryAtPath(s, commit.TreeHash, projectPath)
		if err != nil {
			return false, fmt.Errorf("getting commit tree entry: %w", err)
		}

		var previous pathEntry
		if !parentTree.IsZero() {
			previous, err = p.entryAtPath(s, parentTree, projectPath)
			if err != nil {
				return false, fmt.Errorf("getting parent tree entry: %w", err)
			}
		}

		if current != previous {
			return true, nil
		}
	}

	return false, nil
}

// entryAtPath returns the entry found at the given path inside the given root tree, if any. Results are memoized.
func (p *Parser) entryAtPath(s storer.EncodedObjectStorer, rootTree plumbing.Hash, entryPath string) (pathEntry, error) {
	byPath, ok := p.pathEntries[rootTree]
	if ok {
		if entry, ok := byPath[entryPath]; ok {
			return entry, nil
		}
	} else {
		byPath = make(map[string]pathEntry, 1)
		p.pathEntries[rootTree] = byPath
	}

	entry, err := lookupEntry(s, rootTree, entryPath)
	if err != nil {
		return pathEntry{}, err
	}

	byPath[entryPath] = entry

	return entry, nil
}

// findTreeEntry returns the direct child entry of the given tree with the given name, or nil if there is none.
func findTreeEntry(tree *object.Tree, name string) *object.TreeEntry {
	for i := range tree.Entries {
		if tree.Entries[i].Name == name {
			return &tree.Entries[i]
		}
	}

	return nil
}

// lookupEntry walks down the tree, one path component at a time, to find the entry at the given path.
func lookupEntry(s storer.EncodedObjectStorer, rootTree plumbing.Hash, entryPath string) (pathEntry, error) {
	tree, err := object.GetTree(s, rootTree)
	if err != nil {
		return pathEntry{}, fmt.Errorf("getting tree %s: %w", rootTree, err)
	}

	for {
		name, rest, hasRest := strings.Cut(entryPath, "/")

		// Entries are scanned directly rather than through Tree.FindEntry, which rejects names that cannot exist in
		// a valid tree (e.g., "." or ".git") with an error. Such names are simply never found here, so a project
		// path made of them matches nothing.
		entry := findTreeEntry(tree, name)
		if entry == nil {
			return pathEntry{}, nil
		}

		if !hasRest {
			return pathEntry{hash: entry.Hash, mode: entry.Mode, found: true}, nil
		}

		// An intermediate component that is not a directory means nothing can exist at the given path
		if entry.Mode != filemode.Dir {
			return pathEntry{}, nil
		}

		tree, err = object.GetTree(s, entry.Hash)
		if err != nil {
			return pathEntry{}, fmt.Errorf("getting tree %s: %w", entry.Hash, err)
		}

		entryPath = rest
	}
}

func shortenMessage(message string) string {
	if len(message) > 50 {
		return message[0:47] + "..."
	}

	return message
}

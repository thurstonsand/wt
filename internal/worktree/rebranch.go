package worktree

import (
	"fmt"

	"github.com/thurstonsand/wt/internal/git"
)

// RebranchOptions configures the rebranch operation.
type RebranchOptions struct {
	NewBranch   string // Required: name of the new branch to re-seat onto.
	ForWorktree string // Worktree to target (optional if run inside one).
	Onto        string // Baseline to rebranch onto; defaults to origin/<default>.
	Remote      string // Remote to rebaseline from; defaults to "origin".
}

// RebranchResult describes the outcome of a rebranch.
type RebranchResult struct {
	WorktreeName  string // Folder name, preserved across the rebranch.
	WorktreePath  string
	OldBranch     string // The spent branch, left behind for prune.
	NewBranch     string
	Baseline      string // Branch the new branch was seated on (e.g. "main").
	StaleCommits  int    // Commits on the old branch not on the baseline.
	IndexRestored bool   // Whether the staged/unstaged split was preserved.
	Conflict      bool   // True when dirty-state restore hit a conflict.
}

// Rebranch re-seats a landed worktree onto a fresh baseline under a new branch,
// keeping the same directory and carrying uncommitted dirty-state forward.
// The spent branch is left behind for prune; committed work is never destroyed.
func (m *Manager) Rebranch(opts RebranchOptions) (*RebranchResult, error) {
	if opts.NewBranch == "" {
		return nil, ErrNewBranchRequired
	}
	remote := opts.Remote
	if remote == "" {
		remote = "origin"
	}

	wt, err := m.resolveWorktree(opts.ForWorktree)
	if err != nil {
		return nil, err
	}

	exists, err := m.git.LocalBranchExists(opts.NewBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to check local branch: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: %s", ErrWorktreeExists, opts.NewBranch)
	}

	wtGit := git.New(wt.WorktreePath)

	hasRemotes, err := m.git.HasRemotes()
	if err == nil && hasRemotes {
		if err = m.git.FetchAll(true); err != nil {
			return nil, fmt.Errorf("failed to fetch: %w", err)
		}
	}

	if err = m.checkLanded(wt.Branch); err != nil {
		return nil, err
	}

	baseline, startPoint, err := m.resolveBaseline(remote, opts.Onto)
	if err != nil {
		return nil, err
	}

	staleCommits := m.countStaleCommits(wtGit, startPoint)

	result := &RebranchResult{
		WorktreeName: wt.Name,
		WorktreePath: wt.WorktreePath,
		OldBranch:    wt.Branch,
		NewBranch:    opts.NewBranch,
		Baseline:     baseline,
		StaleCommits: staleCommits,
	}

	hasStash, err := wtGit.StashPushAll("wt-rebranch")
	if err != nil {
		return nil, fmt.Errorf("failed to stash changes: %w", err)
	}

	if err = wtGit.SwitchCreate(opts.NewBranch, startPoint); err != nil {
		if hasStash {
			_ = wtGit.StashPop()
		}
		return nil, fmt.Errorf("failed to create branch %q on %s: %w", opts.NewBranch, startPoint, err)
	}

	if err = m.git.SetBranchMeta(opts.NewBranch, git.BranchMeta{Parent: baseline}); err != nil {
		return nil, fmt.Errorf("failed to save branch metadata: %w", err)
	}

	if !hasStash {
		result.IndexRestored = true
		return result, nil
	}

	return m.restoreDirtyState(wtGit, result)
}

// restoreDirtyState pops the rebranch stash onto the new branch, preferring to
// preserve the staged/unstaged split. On conflict, the worktree is left for
// manual resolution. If --index cannot reapply cleanly, it falls back to a
// plain pop, collapsing the split into unstaged changes.
func (m *Manager) restoreDirtyState(wtGit *git.Git, result *RebranchResult) (*RebranchResult, error) {
	conflict := func() (*RebranchResult, error) {
		result.Conflict = true
		return result, fmt.Errorf("%w in %s: resolve conflicts, then commit or stash",
			ErrRebranchConflict, result.WorktreePath)
	}

	err := wtGit.StashPopIndex()
	if err == nil {
		result.IndexRestored = true
		return result, nil
	}
	if hasConflicts, cerr := wtGit.HasConflicts(); cerr == nil && hasConflicts {
		return conflict()
	}

	if err = wtGit.StashPop(); err != nil {
		if hasConflicts, cerr := wtGit.HasConflicts(); cerr == nil && hasConflicts {
			return conflict()
		}
		return result, fmt.Errorf("failed to restore changes: %w", err)
	}
	return result, nil
}

// resolveBaseline determines the branch the new branch is seated on and the
// git start-point ref to create it from. Without --onto, it rebaselines onto
// the remote's default branch (origin/<default>). With --onto, it accepts a
// local branch, a remote-tracking ref, or a bare branch name resolved against
// the remote. Returns the display name and the start-point ref.
func (m *Manager) resolveBaseline(remote, onto string) (baseline, startPoint string, err error) {
	if onto == "" {
		var def string
		def, err = m.git.DefaultRemoteBranch(remote)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve default branch on %s: %w", remote, err)
		}
		return def, remote + "/" + def, nil
	}

	exists, err := m.git.RefExists(onto)
	if err != nil {
		return "", "", fmt.Errorf("failed to check ref %q: %w", onto, err)
	}
	if exists {
		return onto, onto, nil
	}

	remoteRef := remote + "/" + onto
	exists, err = m.git.RefExists(remoteRef)
	if err != nil {
		return "", "", fmt.Errorf("failed to check ref %q: %w", remoteRef, err)
	}
	if exists {
		return onto, remoteRef, nil
	}

	return "", "", fmt.Errorf("baseline %q not found locally or on %s", onto, remote)
}

// checkLanded enforces that the branch has landed before rebranching, with a
// distinct error for the never-pushed case.
func (m *Manager) checkLanded(branch string) error {
	hasUpstream, err := m.git.HasUpstream(branch)
	if err != nil {
		return fmt.Errorf("failed to check upstream: %w", err)
	}
	if !hasUpstream {
		return fmt.Errorf("%w: %s", ErrNeverPushed, branch)
	}
	gone, err := m.git.UpstreamGone(branch)
	if err != nil {
		return fmt.Errorf("failed to check upstream status: %w", err)
	}
	if !gone {
		return fmt.Errorf("%w: upstream of %s still exists", ErrNotLanded, branch)
	}
	return nil
}

// countStaleCommits reports how many commits the spent branch holds beyond the
// new baseline, for surfacing in the result. Best-effort: 0 on error.
func (m *Manager) countStaleCommits(wtGit *git.Git, startPoint string) int {
	commits, err := wtGit.CommitsBetween(startPoint, "HEAD")
	if err != nil {
		return 0
	}
	return len(commits)
}

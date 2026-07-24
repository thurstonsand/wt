package worktree

import (
	"errors"
	"fmt"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
)

// ErrMergeConflict indicates a merge conflict occurred.
var ErrMergeConflict = errors.New("merge conflict")

// ErrProtectedBranch indicates an attempt to directly commit to a protected branch.
var ErrProtectedBranch = errors.New("cannot commit directly to protected branch")

// ErrTargetNotCheckedOut indicates the target branch is not checked out anywhere.
var ErrTargetNotCheckedOut = errors.New("target branch not checked out")

// ErrSourceDirty indicates the source worktree has uncommitted changes.
var ErrSourceDirty = errors.New("source worktree has uncommitted changes")

// MergeOptions configures worktree merge.
type MergeOptions struct {
	Name         string
	Mode         config.MergeMode
	Force        bool
	Base         string
	DeferRemoval bool
}

// MergeResult contains information about a merge (successful or conflicted).
type MergeResult struct {
	WorktreeName string
	WorktreePath string
	TargetBranch string
	TargetPath   string
	Mode         config.MergeMode
	Commits      []git.CommitInfo // For squash/rebase
	FileCount    int              // For staged mode
	ConflictPath string           // Set on conflict: directory where conflicts need resolution
}

// Merge merges a worktree's changes back to its parent branch.
// Finds where the target branch is checked out (could be main repo or any worktree)
// and performs the merge there.
// On success, the worktree is deleted. On conflict, worktree is preserved.
func (m *Manager) Merge(opts MergeOptions) (*MergeResult, error) {
	wt, err := m.resolveWorktree(opts.Name)
	if err != nil {
		return nil, err
	}

	targetBranch := opts.Base
	if targetBranch == "" {
		targetBranch = wt.Parent
	}
	if targetBranch == "" {
		return nil, fmt.Errorf("%w for %q\nUse --base to specify: wt merge %s --base <branch>",
			ErrParentUnknown, wt.Name, wt.Name)
	}

	targetWt, found, err := m.git.FindWorktreeByBranch(targetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to find target branch: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("%w: %s\nCheckout the target branch first",
			ErrTargetNotCheckedOut, targetBranch)
	}

	targetGit := git.New(targetWt.Path)

	if opts.Mode != config.MergeModeStaged && !opts.Force {
		var dirty bool
		dirty, err = wt.IsDirty()
		if err != nil {
			return nil, fmt.Errorf("failed to check source dirty state: %w", err)
		}
		if dirty {
			return nil, fmt.Errorf("%w: commit or stash changes, or use --force to discard them", ErrSourceDirty)
		}
	}

	if !opts.Force && isProtectedBranch(targetBranch) && opts.Mode != config.MergeModeStaged {
		return nil, fmt.Errorf("%w: %s (use --staged or -f to override)", ErrProtectedBranch, targetBranch)
	}

	result := &MergeResult{
		WorktreeName: wt.Name,
		WorktreePath: wt.WorktreePath,
		TargetBranch: targetBranch,
		TargetPath:   targetWt.Path,
		Mode:         opts.Mode,
	}

	// Get HEAD before merge for commit range calculation.
	headBefore, err := targetGit.RevParse("HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get target HEAD: %w", err)
	}

	targetStashed := false
	if opts.Mode != config.MergeModeStaged {
		targetStashed, err = targetGit.StashPushAll("wt merge target state")
		if err != nil {
			return nil, fmt.Errorf("failed to stash target worktree changes: %w", err)
		}
	}

	var mergeErr error
	switch opts.Mode {
	case config.MergeModeSquash:
		mergeErr = m.mergeSquash(wt, targetGit)
		if mergeErr == nil {
			if commit, err := targetGit.LastCommit(); err == nil && commit.Hash != "" {
				result.Commits = []git.CommitInfo{commit}
			}
		}
	case config.MergeModeRebase:
		mergeErr = m.mergeRebase(wt, targetBranch, targetGit, opts.Force)
		if mergeErr == nil && headBefore != "" {
			if commits, err := targetGit.CommitsBetween(headBefore, "HEAD"); err == nil {
				result.Commits = commits
			}
		}
	case config.MergeModeStaged:
		state, err := captureDirtyState(wt.git)
		if err != nil {
			return nil, fmt.Errorf("failed to capture worktree state: %w", err)
		}
		branchFiles, err := targetGit.DiffBranchFiles(targetBranch, wt.Branch)
		if err != nil {
			fmt.Printf("warning: failed to count diff files: %v\n", err)
		} else {
			result.FileCount = state.countWith(branchFiles)
		}
		mergeErr = m.mergeStaged(wt, targetGit, state)
	default:
		return nil, fmt.Errorf("unknown merge mode: %s", opts.Mode)
	}

	if targetStashed {
		if mergeErr != nil {
			if err := targetGit.ResetHard(headBefore); err != nil {
				return nil, fmt.Errorf("merge failed and target worktree could not be reset: %w", err)
			}
		}
		if err := targetGit.StashPopIndex(); err != nil {
			if mergeErr != nil {
				return nil, fmt.Errorf("merge failed and target worktree changes could not be restored: %w", err)
			}
			if resetErr := targetGit.ResetHard(headBefore); resetErr != nil {
				return nil, fmt.Errorf("target worktree changes conflict with the merge and it could not be rolled back: %w", resetErr)
			}
			if cleanErr := targetGit.CleanUntracked(); cleanErr != nil {
				return nil, fmt.Errorf("merge was rolled back but partially restored untracked files could not be removed: %w", cleanErr)
			}
			if restoreErr := targetGit.StashPopIndex(); restoreErr != nil {
				return nil, fmt.Errorf("merge was rolled back but target worktree changes could not be restored: %w", restoreErr)
			}
			mergeErr = fmt.Errorf("%w: target worktree changes overlap merged changes; merge was rolled back", ErrMergeConflict)
		}
	}

	if mergeErr != nil {
		if errors.Is(mergeErr, ErrMergeConflict) {
			switch opts.Mode {
			case config.MergeModeRebase:
				result.ConflictPath = wt.WorktreePath
			case config.MergeModeSquash, config.MergeModeStaged:
				result.ConflictPath = targetWt.Path
			}

			return result, mergeErr
		}
		return nil, mergeErr
	}

	if opts.DeferRemoval {
		return result, nil
	}

	if err := targetGit.WorktreeRemove(wt.WorktreePath, true); err != nil {
		return nil, fmt.Errorf("failed to remove worktree after merge: %w", err)
	}

	if err := removeDirAndCleanup(m.globalStore, wt.WorktreePath); err != nil {
		fmt.Printf("warning: failed to remove worktree directory: %v\n", err)
	}

	if !isProtectedBranch(wt.Branch) {
		if err := targetGit.DeleteBranch(wt.Branch, true); err != nil {
			fmt.Printf("warning: failed to delete branch %q: %v\n", wt.Branch, err)
		}
	}

	return result, nil
}

func (m *Manager) mergeSquash(wt *Worktree, targetGit *git.Git) error {
	if err := targetGit.MergeSquash(wt.Branch); err != nil {
		hasConflicts, checkErr := targetGit.HasConflicts()
		if checkErr == nil && hasConflicts {
			return fmt.Errorf("%w: %w", ErrMergeConflict, err)
		}
		return fmt.Errorf("failed to squash merge: %w", err)
	}

	dirty, err := targetGit.IsDirty()
	if err != nil {
		return fmt.Errorf("failed to check for changes: %w", err)
	}
	if !dirty {
		return nil
	}

	if err := targetGit.Commit(fmt.Sprintf("Merge worktree %s (squash)", wt.Name)); err != nil {
		return fmt.Errorf("failed to commit squash merge: %w", err)
	}

	return nil
}

func (m *Manager) mergeRebase(wt *Worktree, targetBranch string, targetGit *git.Git, force bool) error {
	wtGit := git.New(wt.WorktreePath)

	var err error
	if force {
		err = wtGit.RebaseAutostash(targetBranch)
	} else {
		err = wtGit.Rebase(targetBranch)
	}
	if err != nil {
		hasConflicts, checkErr := wtGit.HasConflicts()
		if checkErr == nil && hasConflicts {
			return fmt.Errorf("%w: %w", ErrMergeConflict, err)
		}
		return fmt.Errorf("failed to rebase: %w", err)
	}

	if err := targetGit.MergeFastForward(wt.Branch); err != nil {
		return fmt.Errorf("failed to fast-forward: %w", err)
	}

	return nil
}

func (m *Manager) mergeStaged(wt *Worktree, targetGit *git.Git, state dirtyState) error {
	if err := targetGit.MergeSquash(wt.Branch); err != nil {
		hasConflicts, checkErr := targetGit.HasConflicts()
		if checkErr == nil && hasConflicts {
			return fmt.Errorf("%w: %w", ErrMergeConflict, err)
		}
		return fmt.Errorf("failed to stage changes: %w", err)
	}
	if err := state.apply(wt.git, targetGit); err != nil {
		return fmt.Errorf("failed to transfer worktree state: %w", err)
	}
	return nil
}

func isProtectedBranch(branch string) bool {
	return branch == "main" || branch == "master"
}

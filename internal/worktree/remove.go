package worktree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
)

// RemoveOptions configures worktree removal.
type RemoveOptions struct {
	Name           string
	Force          bool
	PreserveBranch bool
}

// RemoveResult contains information about a successful removal.
type RemoveResult struct {
	WorktreeName string
	TargetPath   string
}

// Remove removes a worktree.
func (m *Manager) Remove(opts RemoveOptions) (*RemoveResult, error) {
	wt, err := m.resolveWorktree(opts.Name)
	if err != nil {
		return nil, err
	}

	mainWt, err := m.git.MainWorktree()
	if err != nil {
		return nil, fmt.Errorf("failed to find main worktree: %w", err)
	}
	mainGit := git.New(mainWt.Path)

	current, cwErr := m.CurrentWorktree()
	if cwErr != nil && !errors.Is(cwErr, ErrNotInWorktree) {
		fmt.Printf("warning: failed to detect current worktree: %v\n", cwErr)
	}
	insideCurrent := current != nil && current.Name == wt.Name

	if !opts.Force {
		dirty, err := wt.IsDirty()
		if err != nil {
			return nil, fmt.Errorf("failed to check dirty state: %w", err)
		}
		if dirty {
			return nil, fmt.Errorf("%w: %s (use -f to force)", ErrWorktreeDirty, wt.Name)
		}
	}

	if err := mainGit.WorktreeRemove(wt.WorktreePath, opts.Force); err != nil {
		return nil, fmt.Errorf("failed to remove worktree: %w", err)
	}

	if err := removeDirAndCleanup(m.globalStore, wt.WorktreePath); err != nil {
		fmt.Printf("warning: failed to remove worktree directory: %v\n", err)
	}

	if !opts.PreserveBranch && !isProtectedBranch(wt.Branch) {
		if err := mainGit.DeleteBranch(wt.Branch, true); err != nil {
			fmt.Printf("warning: failed to delete branch %q: %v\n", wt.Branch, err)
		}
	}

	result := &RemoveResult{WorktreeName: wt.Name}

	if insideCurrent {
		result.TargetPath = mainWt.Path
		if wt.Parent != "" {
			if parentWt, found, err := mainGit.FindWorktreeByBranch(wt.Parent); err == nil && found {
				result.TargetPath = parentWt.Path
			}
		}
	}

	return result, nil
}

func removeDirAndCleanup(store *config.GlobalStore, dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	parent := filepath.Dir(dir)
	if !store.IsManagedPath(parent) {
		return nil
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) > 0 {
		return nil //nolint:nilerr // non-empty or unreadable parent dir is fine to leave
	}
	_ = os.Remove(parent)
	return nil
}

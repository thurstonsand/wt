package worktree

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/thurstonsand/wt/internal/git"
)

// RebaseOptions configures the rebase operation.
type RebaseOptions struct {
	Name string // Worktree name (optional if in worktree)
	Onto string // Override parent branch (optional)
}

// Rebase updates a worktree by rebasing onto its parent branch.
// Returns the rebased worktree on success.
func (m *Manager) Rebase(opts RebaseOptions) (*Worktree, error) {
	wt, err := m.resolveWorktree(opts.Name)
	if err != nil {
		return nil, err
	}

	wtGit := git.New(wt.WorktreePath)

	inProgress, err := wtGit.RebaseInProgress()
	if err != nil {
		return nil, fmt.Errorf("failed to check rebase state: %w", err)
	}
	if inProgress {
		return nil, fmt.Errorf("%w: resolve with 'git rebase --continue' or '--abort' in %s",
			ErrRebaseInProgress, wt.WorktreePath)
	}

	oldParent := wt.Parent
	newParent := opts.Onto
	if newParent == "" {
		newParent = oldParent
	}

	exists, err := wtGit.RefExists(newParent)
	if err != nil {
		return nil, fmt.Errorf("failed to check branch: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("parent branch %q not found", newParent)
	}

	if ferr := wtGit.Fetch("origin", newParent); ferr != nil {
		if !isRemoteMissing(ferr) {
			log.Printf("fetch origin/%s: %v", newParent, ferr)
		}
	}

	updateParent := opts.Onto != "" && opts.Onto != oldParent
	if updateParent {
		err = wtGit.RebaseOnto(newParent, oldParent)
	} else {
		err = wtGit.RebaseAutostash(newParent)
	}

	if err != nil {
		hasConflicts, checkErr := wtGit.HasConflicts()
		if checkErr == nil && hasConflicts {
			return nil, fmt.Errorf("%w in %s\n\nResolve conflicts, then:\n  git rebase --continue\n\nOr abort:\n  git rebase --abort",
				ErrRebaseConflict, wt.Name)
		}
		return nil, fmt.Errorf("rebase failed: %w", err)
	}

	if updateParent {
		wt.Parent = opts.Onto
		if err := m.git.SetBranchMeta(wt.Branch, git.BranchMeta{Parent: opts.Onto}); err != nil {
			return nil, fmt.Errorf("failed to update branch metadata: %w", err)
		}
	}

	return wt, nil
}

// CurrentWorktree returns the worktree for the current directory, if any.
// Uses git worktree list to identify worktrees, then matches against wt metadata.
func (m *Manager) CurrentWorktree() (*Worktree, error) {
	toplevel, err := m.git.RevParse("--show-toplevel")
	if err != nil {
		return nil, ErrNotInWorktree
	}

	toplevel, err = evalPath(toplevel)
	if err != nil {
		return nil, ErrNotInWorktree
	}

	linkedWorktrees, err := m.git.LinkedWorktrees()
	if err != nil {
		return nil, err
	}

	for _, gw := range linkedWorktrees {
		gwPath, _ := evalPath(gw.Path)
		if gwPath == toplevel {
			meta, err := m.git.GetBranchMeta(gw.Branch)
			if err != nil {
				return nil, fmt.Errorf("failed to get branch metadata: %w", err)
			}
			return &Worktree{
				Name:         filepath.Base(gw.Path),
				WorktreePath: gw.Path,
				Branch:       gw.Branch,
				BranchMeta:   meta,
				git:          git.New(gw.Path),
			}, nil
		}
	}
	return nil, ErrNotInWorktree
}

func (m *Manager) resolveWorktree(name string) (*Worktree, error) {
	if name != "" {
		return m.Get(name)
	}
	return m.CurrentWorktree()
}

// evalPath returns the absolute, symlink-resolved path.
func evalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// isRemoteMissing returns true if the error indicates the remote doesn't exist.
func isRemoteMissing(err error) bool {
	var execErr *git.ExecError
	if errors.As(err, &execErr) {
		return strings.Contains(execErr.Stderr, "does not appear to be a git repository")
	}
	return false
}

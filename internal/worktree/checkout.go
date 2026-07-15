package worktree

import (
	"errors"
	"fmt"

	"github.com/thurstonsand/wt/internal/git"
)

// CheckoutOptions configures worktree checkout of an existing branch.
type CheckoutOptions struct {
	Branch string
	Parent string
	With   []string
}

// defaultParent resolves the parent to record when none is given explicitly.
// It prefers the remote's default branch (origin/<default>) so prune's tracking
// reflects the true trunk rather than a possibly-stale local ref, falling back
// to the local default branch when the remote default is unresolvable (e.g. no
// remote). Returns "" when no sensible parent exists or it would self-reference.
func (m *Manager) defaultParent(branch string) string {
	if def, err := m.git.DefaultRemoteBranch("origin"); err == nil && def != branch {
		return "origin/" + def
	}
	if def, err := m.git.DefaultBranch(); err == nil && def != branch {
		return def
	}
	return ""
}

// Checkout creates a managed worktree for an existing branch.
func (m *Manager) Checkout(opts CheckoutOptions) (*Worktree, error) {
	if opts.Branch == "" {
		return nil, errors.New("branch name is required")
	}

	if m.Exists(opts.Branch) {
		return nil, fmt.Errorf("%w: %s", ErrWorktreeExists, opts.Branch)
	}

	// Refresh remote-tracking refs so a remote-only branch resolves via git's
	// DWIM tracking-branch creation. Tolerate failure (offline / no remotes).
	if hasRemotes, err := m.git.HasRemotes(); err == nil && hasRemotes {
		_ = m.git.FetchAll(false)
	}

	wtPath := m.WorktreePath(opts.Branch)

	if err := m.git.WorktreeAdd(wtPath, opts.Branch, false, ""); err != nil {
		return nil, fmt.Errorf("failed to checkout branch: %w", err)
	}

	if err := m.copyIncludes(wtPath, opts.With); err != nil {
		_ = m.git.WorktreeRemove(wtPath, true)
		return nil, fmt.Errorf("failed to copy include files: %w", err)
	}

	parent := opts.Parent
	if parent == "" {
		parent = m.defaultParent(opts.Branch)
	}
	if parent != "" {
		if err := m.git.SetBranchMeta(opts.Branch, git.BranchMeta{Parent: parent}); err != nil {
			_ = m.git.WorktreeRemove(wtPath, true)
			return nil, fmt.Errorf("failed to save branch metadata: %w", err)
		}
	}

	meta, err := m.git.GetBranchMeta(opts.Branch)
	if err != nil {
		_ = m.git.WorktreeRemove(wtPath, true)
		return nil, fmt.Errorf("failed to read branch metadata: %w", err)
	}

	return &Worktree{
		RepoName:     m.repoName,
		Name:         opts.Branch,
		WorktreePath: wtPath,
		Branch:       opts.Branch,
		BranchMeta:   meta,
		git:          git.New(wtPath),
	}, nil
}

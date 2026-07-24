package worktree

import (
	"errors"
	"fmt"
	"os"

	"github.com/thurstonsand/wt/internal/config"
	"github.com/thurstonsand/wt/internal/git"
)

// ErrCleanWithBase is returned when --no-clean is used with --base.
var ErrCleanWithBase = errors.New("cannot use --no-clean with --base")

// ForkOptions configures worktree forking.
type ForkOptions struct {
	Name  string
	Base  string
	Clean *bool
	With  []string
}

// Fork creates a new worktree with the given options.
// If Base is empty, branches from HEAD. By default (clean=false),
// staged/unstaged changes and untracked files are copied to the new worktree.
// Include files are always copied regardless of clean setting.
func (m *Manager) Fork(opts ForkOptions) (*Worktree, error) {
	if opts.Name == "" {
		opts.Name = GenerateName()
	}

	if m.Exists(opts.Name) {
		return nil, fmt.Errorf("%w: %s", ErrWorktreeExists, opts.Name)
	}

	if err := m.guardBranchCollision(opts.Name); err != nil {
		return nil, err
	}

	cfg, _, err := m.globalStore.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	clean := m.resolveClean(cfg, opts.Base, opts.Clean)

	if opts.Base != "" && !clean {
		return nil, ErrCleanWithBase
	}

	base := opts.Base
	parentBranch := base
	if base == "" {
		parentBranch, err = m.git.CurrentBranch()
		if err != nil {
			return nil, fmt.Errorf("failed to determine current branch: %w", err)
		}
	}

	var state dirtyState
	if !clean {
		state, err = captureDirtyState(m.git)
		if err != nil {
			return nil, err
		}
	}

	wtPath := m.WorktreePath(opts.Name)

	if err = m.git.WorktreeAdd(wtPath, opts.Name, true, base); err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	wtGit := git.New(wtPath)

	cleanup := func() {
		_ = m.git.WorktreeRemove(wtPath, true)
		_ = m.git.DeleteBranch(opts.Name, true)
	}

	if !clean {
		if err = state.apply(m.git, wtGit); err != nil {
			cleanup()
			return nil, err
		}
	}

	if err = m.copyIncludes(wtPath, opts.With); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to copy include files: %w", err)
	}

	if err = m.git.SetBranchMeta(opts.Name, git.BranchMeta{Parent: parentBranch}); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to save branch metadata: %w", err)
	}

	return &Worktree{
		RepoName:     m.repoName,
		Name:         opts.Name,
		WorktreePath: wtPath,
		Branch:       opts.Name,
		BranchMeta:   git.BranchMeta{Parent: parentBranch},
		git:          wtGit,
	}, nil
}

// guardBranchCollision rejects fork names that already name a branch. Forking
// mints a new branch, so an existing local or remote branch of the same name
// should be checked out instead of silently shadowed.
func (m *Manager) guardBranchCollision(name string) error {
	localExists, err := m.git.LocalBranchExists(name)
	if err != nil {
		return fmt.Errorf("failed to check local branch: %w", err)
	}
	if localExists {
		return fmt.Errorf("%w: %s exists locally; use `wt checkout %s`", ErrBranchExists, name, name)
	}

	// Refresh remote-tracking refs so a branch that exists only on a remote and
	// hasn't been fetched yet is still detected. Tolerate failure (offline).
	if hasRemotes, rerr := m.git.HasRemotes(); rerr == nil && hasRemotes {
		_ = m.git.FetchAll(false)
	}

	remoteExists, qualified, err := m.git.RemoteBranchExists(name)
	if err != nil {
		return fmt.Errorf("failed to check remote branch: %w", err)
	}
	if remoteExists {
		return fmt.Errorf("%w: %s exists on remote; use `wt checkout %s` to track it", ErrBranchExists, qualified, name)
	}

	return nil
}

func (m *Manager) resolveClean(cfg config.GlobalConfig, base string, override *bool) bool {
	clean := cfg.Clean
	if base != "" {
		clean = true
	}
	if override != nil {
		clean = *override
	}
	return clean
}

func removeFiles(dir string, paths []string) error {
	for _, p := range paths {
		if err := os.Remove(dir + "/" + p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

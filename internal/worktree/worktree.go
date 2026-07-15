package worktree

import (
	"strings"

	"github.com/thurstonsand/wt/internal/git"
)

// Worktree represents a git worktree with its metadata and git operations.
type Worktree struct {
	RepoName     string
	Name         string
	WorktreePath string
	Branch       string
	git.BranchMeta
	git *git.Git
}

// IsDirty returns true if the worktree has uncommitted changes.
func (w *Worktree) IsDirty() (bool, error) {
	return w.git.IsDirty()
}

// IsLanded reports whether the worktree's branch has landed: it was pushed and
// its configured upstream is now gone. Detection is local; freshness depends on
// the last fetch.
func (w *Worktree) IsLanded() (bool, error) {
	return w.git.UpstreamGone(w.Branch)
}

// State returns human-readable status words for the worktree, joined by commas:
// "dirty", "landed", or "dirty,landed". Empty when neither applies.
func (w *Worktree) State() string {
	var parts []string
	if dirty, err := w.IsDirty(); err == nil && dirty {
		parts = append(parts, "dirty")
	}
	if landed, err := w.IsLanded(); err == nil && landed {
		parts = append(parts, "landed")
	}
	return strings.Join(parts, ",")
}

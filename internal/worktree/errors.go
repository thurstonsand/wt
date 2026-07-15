package worktree

import "errors"

// ErrWorktreeNotFound indicates the requested worktree does not exist.
var ErrWorktreeNotFound = errors.New("worktree not found")

// ErrWorktreeExists indicates a worktree with the given name already exists.
var ErrWorktreeExists = errors.New("worktree already exists")

// ErrWorktreeDirty indicates the worktree has uncommitted changes.
var ErrWorktreeDirty = errors.New("worktree has uncommitted changes")

// ErrNotInWorktree indicates the current directory is not inside a wt-managed worktree.
var ErrNotInWorktree = errors.New("not in a wt-managed worktree")

// ErrRebaseInProgress indicates a rebase is already in progress.
var ErrRebaseInProgress = errors.New("rebase already in progress")

// ErrRebaseConflict indicates a rebase conflict occurred.
var ErrRebaseConflict = errors.New("rebase conflict")

// ErrParentUnknown indicates the parent branch is not known for the worktree.
var ErrParentUnknown = errors.New("parent branch unknown")

// ErrBranchExists indicates fork was given a name that already exists as a
// local or remote branch. Such names should be checked out, not forked.
var ErrBranchExists = errors.New("branch already exists")

// ErrNotLanded indicates rebranch was run against a worktree whose branch has
// not landed: its upstream still exists on the remote.
var ErrNotLanded = errors.New("worktree branch has not landed")

// ErrNeverPushed indicates rebranch was run against a worktree whose branch was
// never pushed, so nothing could have landed.
var ErrNeverPushed = errors.New("worktree branch was never pushed")

// ErrRebranchConflict indicates restoring dirty-state during rebranch hit a
// conflict. The worktree is preserved on the new branch for manual resolution.
var ErrRebranchConflict = errors.New("rebranch conflict")

// ErrNewBranchRequired indicates rebranch was called without a new branch name.
var ErrNewBranchRequired = errors.New("new branch name is required")

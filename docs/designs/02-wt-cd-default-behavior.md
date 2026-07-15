# Design: `wt cd` Default Behavior

## Problem Statement

When working in a worktree, there's no quick way to navigate back to the main repository directory. Users must manually type the path or use shell history. Adding a default behavior to `wt cd` (no arguments) that navigates to the main worktree would streamline this common workflow.

## Design Decisions

### 1. `wt cd` with no arguments → main worktree

Running `wt cd` without a worktree name will navigate to the main repository directory (the directory containing the primary `.git` folder that all worktrees share).

Example:

```bash
# In a worktree: ~/.wt/worktrees/wt/my-feature/
$ wt cd
# Now in: ~/code/wt (main repo)
```

### 2. Implementation via `wt path` extension

The shell wrapper delegates `wt cd <name>` to `wt path <name>`. Extending `wt path` to support no arguments keeps the architecture consistent:

- `wt path <name>` → returns worktree path (existing)
- `wt path` → returns main worktree path (new)

### 3. Main worktree discovery via `git worktree list`

Git's `worktree list` command always returns the main worktree first. We'll use this to reliably identify the main repository path regardless of which worktree the user is currently in.

## Edge Cases

| Scenario                 | Behavior                                    |
| ------------------------ | ------------------------------------------- |
| Already in main worktree | cd to same directory (no-op, harmless)      |
| Not in a git repo        | Error: "not a git repository"               |
| In a bare repo           | Error from git worktree list (no worktrees) |

## Integration Points

### Files Modified

1. **`internal/git/errors.go`** - Add `ErrNoWorktrees` error
2. **`internal/git/worktree.go`** - Add `MainWorktree()` method
3. **`internal/cmd/path.go`** - Accept 0 args, call `MainWorktree()` when no name given
4. **`internal/cmd/cd.go`** - Update args constraint and help text
5. **`internal/shell/wrapper.go`** - No changes needed (already passes `"${@:2}"` which handles empty args)

### Data Flow

```text
wt cd           → shell wrapper → wt path        → git.MainWorktree() → path
wt cd <name>    → shell wrapper → wt path <name> → mgr.Get(name)      → path
```

## Implementation Plan

- [x] Add `MainWorktree() (WorktreeInfo, error)` method to `internal/git/worktree.go`
- [x] Add unit test for `MainWorktree()` in `internal/git/worktree_test.go`
- [x] Update `newPathCmd()` in `internal/cmd/path.go` to accept 0-1 args
- [x] Add integration test for `wt path` with no args in `internal/cmd/path_test.go`
- [x] Update `newCdCmd()` in `internal/cmd/cd.go` to accept 0-1 args and update help text
- [x] Update unit test in `internal/cmd/cd_test.go` to remove "requires arg" test

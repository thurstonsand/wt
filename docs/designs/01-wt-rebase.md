# Design: `wt rebase` - Update Worktree with Parent Changes

## Overview

Add `wt rebase <worktree-name>` command to incorporate commits made to the parent branch since the worktree was created. This keeps worktrees up-to-date with upstream changes without requiring a full merge-back cycle.

## Motivation

When working on a feature in a worktree, the parent branch may receive new commits (merged PRs, hotfixes, etc.). Currently, the only way to incorporate these changes is to:

1. `wt merge` back to parent
2. `wt fork` a new worktree from updated parent
3. Manually continue work

This is disruptive. `wt rebase` provides an in-place update: rebase the worktree branch onto the latest parent, preserving work-in-progress.

## Command Specification

```bash
wt rebase [name]           # Rebase worktree onto its parent branch
wt rebase                  # Rebase current worktree (if in one)
```

### Flags

| Flag              | Description                                                        |
| ----------------- | ------------------------------------------------------------------ |
| `--onto <branch>` | Rebase onto a different branch (updates parent-branch in metadata) |

### Behavior

1. **Resolve worktree**: Use `[name]` arg, or detect from current directory
2. **Load metadata**: Get `ParentBranch` from `wt-metadata/<name>.yaml`
3. **Validate state**:
   - Worktree exists
   - No rebase already in progress
4. **Fetch parent**: Ensure parent branch is up-to-date (if tracking remote)
5. **Rebase**: Execute rebase with `--autostash` (handles uncommitted changes)
6. **Handle conflicts**: If conflicts occur, preserve state for manual resolution
7. **Update metadata**: If `--onto` used, update `ParentBranch` in metadata

### Exit Codes

| Code | Meaning                                |
| ---- | -------------------------------------- |
| 0    | Rebase completed successfully          |
| 1    | Rebase failed (conflicts, dirty state) |

## Implementation

### New Files

```text
internal/cmd/rebase.go      # Cobra command
internal/worktree/rebase.go # Rebase operation
```

### Manager Method

```go
// RebaseOptions configures the rebase operation.
type RebaseOptions struct {
    Name  string // Worktree name (required if not in worktree)
    Onto  string // Override parent branch (optional)
    Force bool   // Allow with uncommitted changes
}

// Rebase updates a worktree by rebasing onto its parent branch.
func (m *Manager) Rebase(opts RebaseOptions) error
```

### Git Wrapper Additions

```go
// RebaseInProgress returns true if a rebase is in progress.
func (g *Git) RebaseInProgress() (bool, error)

// RebaseOnto rebases commits from oldBase..HEAD onto newBase.
// Uses --autostash to handle uncommitted changes automatically.
func (g *Git) RebaseOnto(newBase, oldBase string) error

// Fetch fetches updates from remote for a branch.
func (g *Git) Fetch(remote, branch string) error
```

### Algorithm

```text
Rebase(opts):
  1. wt = m.Get(opts.Name)  // or CurrentWorktree() if name empty
     - Error if worktree not found

  2. wtGit = git.New(wt.WorktreePath)

  3. if wtGit.RebaseInProgress():
     - Return error: "rebase already in progress, resolve with git rebase --continue or --abort"

  4. newParent = opts.Onto or wt.ParentBranch
     oldParent = wt.ParentBranch

  5. if remote tracking exists for newParent:
     - wtGit.Fetch("origin", newParent)  // Best-effort, don't fail

  6. if opts.Onto != "":
     - err = wtGit.RebaseOnto(newParent, oldParent)  // git rebase --onto <new> <old> --autostash
     else:
     - err = wtGit.Rebase(newParent, "--autostash")  // git rebase <parent> --autostash

  7. if err:
       if wtGit.HasConflicts():
         - Return ErrRebaseConflict with instructions
       - Return wrapped error

  8. if opts.Onto != "" && opts.Onto != wt.ParentBranch:
     - wt.ParentBranch = opts.Onto
     - m.metadataStore.SaveMetadata(wt.WorktreeMetadata)

  9. Return nil (success)
```

### --onto Semantics

When `--onto` is specified, we transplant only the worktree-specific commits:

```text
Before (forked from main, want to move to develop):
    A---B---C  (main, original parent)
         \
          D---E  (worktree branch)

    X---Y---Z  (develop, new parent)

Command: wt rebase --onto develop

Executes: git rebase --onto develop main worktree-branch --autostash

After:
    A---B---C  (main)

    X---Y---Z---D'---E'  (worktree branch, rebased)
            ^
         (develop)

Metadata updated: ParentBranch = "develop"
```

Without `--onto`, it's a simple rebase onto the existing parent (incorporating new commits).

### Error Handling

**Conflicts**: Unlike `wt merge`, we don't abort on conflict. The user should resolve in the worktree:

```text
wt: rebase conflict in <name>

Resolve conflicts in: ~/.wt/worktrees/repo/name/
  cd "$(wt path name)"
  git rebase --continue   # after resolving
  git rebase --abort      # to cancel
```

**Autostash**: Git's `--autostash` automatically stashes uncommitted changes before rebase and pops them after. If the stash pop conflicts, git leaves the stash intact for manual resolution.

### Current Directory Detection

Use `git rev-parse --show-toplevel` to get the worktree root, then match against known worktrees in metadata:

```go
func (m *Manager) CurrentWorktree() (*Worktree, error) {
    toplevel, err := m.git.RevParse("--show-toplevel")
    if err != nil {
        return nil, ErrNotInWorktree
    }

    for _, wt := range m.ListAll() {
        if wt.WorktreePath == toplevel {
            return wt, nil
        }
    }
    return nil, ErrNotInWorktree
}
```

## Command Implementation

```go
func newRebaseCmd() *cobra.Command {
    var opts struct {
        onto string
    }

    cmd := &cobra.Command{
        Use:   "rebase [name]",
        Short: "Update worktree by rebasing onto parent branch",
        Long: `Rebase the worktree branch onto the latest parent branch commits.

If run from within a worktree, the name argument is optional.

Uncommitted changes are automatically stashed and restored after rebase.

Examples:
  wt rebase feature-x              # Rebase feature-x onto its parent
  wt rebase                        # Rebase current worktree
  wt rebase feature-x --onto dev   # Rebase onto dev (changes parent)`,
        Args: cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            mgr, err := defaultManager()
            if err != nil {
                return err
            }

            name := ""
            if len(args) > 0 {
                name = args[0]
            }

            return mgr.Rebase(worktree.RebaseOptions{
                Name: name,
                Onto: opts.onto,
            })
        },
    }

    cmd.Flags().StringVar(&opts.onto, "onto", "", "rebase onto different branch (updates parent)")

    return cmd
}
```

## Edge Cases

| Scenario                     | Behavior                                      |
| ---------------------------- | --------------------------------------------- |
| No commits since fork        | No-op, rebase succeeds immediately            |
| Worktree has no commits      | Fast-forward to parent (like fresh fork)      |
| Parent branch deleted        | Error: "parent branch 'X' not found"          |
| `--onto` same as current     | No-op for metadata, still rebases             |
| Running rebase twice         | Second rebase continues from first            |
| Rebase in progress elsewhere | Error: "rebase already in progress"           |
| Uncommitted changes          | Autostashed, restored after rebase            |
| Stash pop conflicts          | Stash preserved, user resolves manually       |

## Testing

### Unit Tests

- `TestRebase_Success`: Basic rebase with diverged branches
- `TestRebase_NoChanges`: Parent unchanged, rebase is no-op
- `TestRebase_Conflicts`: Verify conflict handling, worktree preserved
- `TestRebase_Autostash`: Uncommitted changes stashed and restored
- `TestRebase_OntoFlag`: Transplants commits, updates metadata
- `TestRebase_OntoSameParent`: Still rebases, no metadata change
- `TestRebase_CurrentDirectory`: Detect worktree from cwd
- `TestRebase_NotInWorktree`: Error when cwd detection fails and no name given
- `TestRebase_RebaseInProgress`: Error if rebase already started

### Integration Tests

```go
func TestRebase_Integration(t *testing.T) {
    r := testutil.InitGitRepo(t)

    // Setup: create worktree, add commits to both
    r.CommitFile(t, "base.txt", "initial")
    // wt fork feature
    // Add commits to main after fork
    r.CommitFile(t, "main-update.txt", "update on main")
    // Add commits to worktree
    // wt rebase feature
    // Verify worktree branch is rebased onto main
}
```

## Future Considerations

### Interactive Rebase

Could support `wt rebase -i` to launch interactive rebase, though users can just `cd $(wt path x) && git rebase -i`.

### Rebase --continue/--abort Wrappers

Could add `wt rebase --continue` and `wt rebase --abort` as convenience wrappers, but users can use git directly. Low priority.

## Implementation Tasks

### Task 1: Git Wrapper Additions

- Add `RebaseInProgress()` method to check for `.git/rebase-merge` or `.git/rebase-apply`
- Add `RebaseOnto(newBase, oldBase)` method for `git rebase --onto` with `--autostash`
- Update existing `Rebase()` to accept `--autostash`
- Add `Fetch(remote, branch)` method for optional remote sync
- Tests for new methods
- **Commit:** "feat(git): add RebaseInProgress, RebaseOnto, and Fetch methods"

### Task 2: CurrentWorktree Detection

- Add `Manager.CurrentWorktree()` method
- Add `ErrNotInWorktree` sentinel error
- Tests for detection from various paths
- **Commit:** "feat(worktree): detect current worktree from cwd"

### Task 3: Rebase Operation

- `internal/worktree/rebase.go` with `Rebase()` method
- `ErrRebaseConflict` sentinel error with resolution instructions
- Unit tests for rebase scenarios
- **Commit:** "feat(worktree): add Rebase operation"

### Task 4: Rebase Command

- `internal/cmd/rebase.go` with command implementation
- Register in `root.go`
- Integration tests
- Update shell completions if needed
- **Commit:** "feat: wt rebase command"

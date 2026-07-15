# Design: Show main worktree in `wt ls`

## Problem Statement

`wt ls` only shows linked worktrees. The main (source) worktree is invisible, so users lack context about the repository root — what branch it's on, where it lives, and whether they're currently inside it.

## Design Decisions

**Unified table**: The main worktree is included as the first row in the worktree table with the name `(root)` and parent `-`. A `repo:` subtitle line below the table shows the home-shortened filesystem path.

**Active marker**: A `*` suffix on the name of the worktree you're currently inside (e.g. `(root) *`), consistent with `docker context ls`. Exactly one `*` appears in the output.

**Dirty indicator**: Linked worktrees with uncommitted changes show `~` in the DIRTY column (distinct from the `*` active marker).

**`--all` mode unchanged**: The cross-repo `wt ls -a` mode scans `~/.wt/worktrees/` via filesystem and doesn't have access to main worktree info. Left unchanged for now.

## Output Mockups

From main worktree, with linked worktrees:

```text
NAME        BRANCH    PARENT  DIRTY
(root) *    main      -
feat-foo    feat/foo  main    ~
feat-bar    feat/bar  main

repo: ~/code/wt
```

From a linked worktree:

```text
NAME          BRANCH    PARENT  DIRTY
(root)        main      -
feat-foo *    feat/foo  main    ~
feat-bar      feat/bar  main

repo: ~/code/wt
```

No linked worktrees, from main:

```text
NAME        BRANCH  PARENT  DIRTY
(root) *    main    -

repo: ~/code/wt
```

## Integration Points

1. **`worktree.ListResult`** — new struct bundling `git.WorktreeInfo` (main) + `[]*Worktree` (linked)
2. **`Manager.ListAll()`** — returns `*ListResult` (consolidated), uses `git.MainWorktree()` + `git.LinkedWorktrees()`
3. **`internal/cmd/list.go`** — `listCurrentRepo()` updated to print root line, active marker, and `~`-shortened paths

## Implementation Plan

- [x] **Consolidate `ListAll()` to return `*ListResult`** (`internal/worktree/list.go`)
  - `ListResult{Main git.WorktreeInfo, Worktrees []*Worktree}`
  - Single method fetches main + linked worktrees

- [x] **Update `listCurrentRepo()`** (`internal/cmd/list.go`)
  - Unified table with `(root)` as first row, `*` suffix for active worktree
  - `repo:` subtitle line with `shortenHome()` path
  - Active detection via `currentWorktreePath()` + `pathsEqual()`

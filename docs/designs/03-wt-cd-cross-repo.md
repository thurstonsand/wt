# Design: `wt cd` Cross-Repo Navigation

Status: **Deferred** — pending more usage experience with `wt list --all`.

## Problem Statement

`wt cd <name>` only works within the current repository context because completions and lookup use `defaultManager()`, which is scoped to the CWD repo. There's no way to jump to a worktree in a different repository without knowing its full path.

## Proposed Syntax

```bash
wt cd <repo> <wt>      # navigate to worktree in specific repo
wt cd <repo>/           # navigate to repo's main worktree
```

## Design Considerations

### Completion

Current completions use `defaultManager()` scoped to CWD. Cross-repo completion requires:

1. First argument: complete repo names from `~/.wt/worktrees/*/`
2. Second argument: complete worktree names from `~/.wt/worktrees/<repo>/*/`

This is a two-stage completion that Cobra supports via `RegisterFlagCompletionFunc` or custom `ValidArgsFunction` logic.

### Limitations

- Only repos with existing managed worktrees are discoverable — there's no universal repo registry.
- Repo names are derived from `filepath.Base(repoDir)`, so collisions are possible if two repos share a basename (e.g., two `backend` repos from different orgs).

### Possible Future Enhancements

- Config option `scan_dirs` listing directories to scan for git repos (e.g., `~/code`), enabling completion of repos even without existing worktrees.
- Alias support: `wt cd @myalias` → configured repo + worktree.

## Why Deferred

The immediate need is visibility (`wt list --all`). Cross-repo `cd` adds UX complexity (two-arg commands, ambiguous repo names) that should be informed by real usage patterns first.

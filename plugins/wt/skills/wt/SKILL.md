---
name: wt
description: Use when the user mentions worktrees or when switching between parallel tasks. Manages git worktrees for isolated development.
user-invocable: true
argument-hint: [fork|checkout|merge|list|prune]
---

# Git Worktree Helper (`wt`)

Use `wt` to manage isolated workspaces for parallel feature development.

## When to Use

- **Starting new feature work** that should be isolated from current changes
- **User mentions** worktrees, isolation, parallel work, or `wt` commands
- **Preserving dirty state** while switching to urgent work
- **Merging completed work** back to parent branch

## Core Workflows

### 1. Fork Current Work to New Worktree

When you need to isolate current work or start a new feature:

```bash
# Fork with current dirty state (staged/unstaged/untracked preserved)
wt fork <name>

# Fork clean from HEAD
wt fork --clean <name>

# Fork from different branch (always clean)
wt fork -b <branch> <name>

# Copy an extra include pattern this run (repeatable)
wt fork --with <pattern> <name>
```

After forking, your shell automatically changes to the new worktree (requires shell integration).

`fork` only creates new branches. If there's a preexisting local or remote branch, it errors and tells you to use `wt checkout <name>` instead.

### 2. Check Out an Existing Branch

When a branch already exists and you want to work on it in a managed worktree:

```bash
# Check out existing branch into a worktree
wt checkout <branch>

# Alias
wt co <branch>

# Set parent for merge/rebase tracking
wt checkout -p <parent> <branch>

# Copy an extra include pattern this run (repeatable)
wt checkout --with <pattern> <branch>
```

After checkout, your shell automatically changes to the worktree (requires shell integration).

### 3. Navigate Between Worktrees

```bash
# List all worktrees (shows root worktree, branch, parent, state, active marker)
wt list

# List worktrees across all repos
wt list --all

# Get path to worktree (for cd or scripts)
wt path <name>

# Get path to main repository
wt path

# Change directory to worktree
wt cd <name>

# Change directory to main repository
wt cd
```

### 4. Merge Work Back

When feature work is complete:

```bash
# Rebase and fast-forward - preserves commit history (default)
wt merge <name>

# Squash all commits into one
wt merge --squash <name>

# Apply as staged changes only (no commit)
wt merge --staged <name>

# Merge external worktree (not created by wt fork) to specified branch
wt merge --base <parent-branch> <name>
```

**Protected branches** (main/master): Default to `--staged` mode. Use `-f` to override and use an alternate mode.

On success, worktree and branch are automatically deleted. Shell auto-cd's to the parent directory.

### 5. Update Worktree with Parent Changes

When parent branch has new commits you want to incorporate:

```bash
# Rebase worktree onto updated parent
wt rebase <name>

# Rebase current worktree (when inside one)
wt rebase

# Move worktree to different parent branch
wt rebase --onto <new-parent> <name>
```

Uncommitted changes are auto-stashed and restored.

### 6. Continue After a Landed PR (Rebranch)

When a worktree's branch was pushed, merged via PR/MR, and deleted on the remote — but you still have uncommitted work to continue — the worktree is **landed**. `wt list` shows `landed` in its STATE column.

```bash
# Re-seat the current worktree onto fresh origin/<default> under a new branch
wt rebranch <new-branch>

# Target another worktree by name
wt rebranch <new-branch> -w <worktree>

# Rebranch onto a different baseline
wt rebranch <new-branch> --onto <branch>
```

The directory name is preserved (so the pre-existing agent session keeps working from the same path); only the branch changes. Uncommitted changes (staged, unstaged, untracked) are carried forward. The spent branch is left behind — drop it later with `wt prune --all`. Committed work is never destroyed.

If restoring changes conflicts, the worktree is preserved on the new branch with conflict markers for manual resolution; the old branch stays intact.

### 7. Remove Worktree

```bash
# Remove worktree and delete branch
wt rm <name>

# Force remove dirty worktree (also force-deletes unmerged branch)
wt rm -f <name>

# Remove worktree but keep the branch
wt rm --preserve-branch <name>
```

### 8. Prune Stale Worktrees

When a source repo is deleted, its worktree directories under `~/.wt/worktrees/` become orphaned. Prune scans globally and removes them:

```bash
# Scan and pick items to remove (stale dirs pre-selected)
wt prune

# Pre-select wt-managed branches across all tracked repos too
wt prune --all

# Report only (no deletion)
wt prune --dry-run

# Remove without prompting (--all to also delete branches)
wt prune --force
wt prune --all --force
```

## Configuration

View/modify settings:

```bash
wt config --help  # Show options
```

**Include files**: gitignored files needed in every worktree (e.g. `.env`) are copied via `.gitignore`-style patterns in `.worktreeinclude` (repo root) and `~/.wt/worktreeinclude` (user). Copied on every fork/checkout, even with `--clean`.

## Decision Guide

| Situation                            | Command                                        |
| ------------------------------------ | ---------------------------------------------- |
| Start isolated feature work          | `wt fork feature-name`                         |
| Work on existing branch              | `wt checkout branch-name`                      |
| Urgent fix, preserve current changes | `wt fork urgent-fix` then work in new worktree |
| See all active worktrees             | `wt list` (or `wt list -a` for all repos)      |
| Incorporate upstream changes         | `wt rebase feature-name`                       |
| Feature complete, merge back         | `wt merge feature-name`                        |
| PR landed, continue with new work    | `wt rebranch next-feature`                     |
| Abandoned work, clean up             | `wt rm -f old-feature`                         |
| Orphaned worktrees after repo delete | `wt prune`                                     |
| Orphaned branches from old worktrees | `wt prune --all`                               |

## Shell Integration Setup

For `wt cd`, auto-cd on `wt fork/checkout/merge/rm` to work, user needs shell integration:

```bash
# Add to ~/.zshrc or ~/.bashrc
eval "$(wt shell zsh)"  # or bash
```

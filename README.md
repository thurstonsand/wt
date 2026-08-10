# wt

A CLI that wraps `git worktree` with dirty-state forking, merge-back modes, and shell integration.

## Install

```bash
brew install thurstonsand/tap/wt
```

Or with Go:

```bash
go install github.com/thurstonsand/wt@latest
```

Or from source with `mise`:

```bash
git clone git@github.com:thurstonsand/wt.git
cd wt
mise trust && mise bootstrap
mise run install
```

If you have mise's shell activation, `mise bootstrap` will run automatically after you trust the repo.

## Shell integration

Add one line to your rc file:

```bash
# Bash
eval "$(wt shell bash)"

# Zsh
eval "$(wt shell zsh)"
```

This gives you `wt cd` (change to a worktree directory), auto-cd after `wt fork`, `wt checkout`, `wt merge`, and `wt rm`, and tab completion.

## Workflows

### Fork and merge back

You're mid-work on main and want to spin something off to implement, then land it back.

```bash
wt fork my-feature

# auto-cd's into the new worktree.
# staged, unstaged, and untracked files came with you.

# ... work, commit, push ...

wt merge               # from inside the worktree
wt merge my-feature    # or by name, from parent
```

`wt merge` defaults to rebase mode (fast-forwards main to your rebased commits). You can also `--squash` into one commit or `--staged` to land changes without committing. See [wt merge](#wt-merge) for all flags.

### Parallel branches

You have two (or more) independent lines of work, each getting its own MR. Use `wt fork` for new branches and `wt co`/`wt checkout` for ones that already exist.

```bash
# from main
wt fork auth-refactor
# ... start work, commit ...

wt cd                      # back to main
wt co logging-cleanup      # existing remote branch
# ... work, commit ...

wt cd auth-refactor        # jump between worktrees
wt cd logging-cleanup

# push each branch normally, open MRs, clean up when merged upstream
wt rm auth-refactor
wt rm logging-cleanup
```

### Other useful commands

```bash
wt list/ls                # show worktrees for current repo (-a for all repos)
wt cd my-feature          # cd into a worktree (shell integration required)
wt cd                     # cd back to the main repo (same as: `wt cd root`)
wt rm my-feature          # remove worktree and delete branch
wt rebase my-feature      # pull in upstream changes from parent
wt rebranch next-feature  # continue in a landed worktree under a new branch
wt prune                  # clean up stale worktree dirs (and pick prunable branches)
wt prune --all            # also pre-select wt-managed branches across all tracked repos
```

## Configuration

Stored at `~/.wt/config.yaml`. Edit directly or use `wt config`:

```yaml
# Skip dirty-state transfer on fork (default: false)
clean: false

# Default merge mode: rebase, squash, or staged
merge: rebase

# Run 'direnv allow' in new worktrees when .envrc exists (default: false)
direnv: false

# Shell commands to run in new worktrees after creation
# Each command runs via sh -c with cwd set to the worktree directory.
# Runs for every repo, so must be universal.
# Failures are non-fatal (warnings only).
post_create:
  - [ -f package.json ] && npm install
  - [ -f .tool-versions ] && mise trust

# Repositories tracked by wt (auto-populated on first use per repo)
# Used by `wt prune` to scan for prunable branches globally.
repos:
  - /Users/you/code/project-a
  - /Users/you/code/project-b
```

```bash
wt config show                       # dump current settings
wt config set merge squash           # change a value
wt config edit                       # open in $EDITOR
```

### Include files

Some files are deliberately ignored by git but still needed in every worktree —
`.env`, `docker-compose.override.yml`, local secrets. Dirty-state transfer skips
them (it respects `.gitignore`), so `wt` copies them from a `.worktreeinclude`
file using `.gitignore` pattern syntax. Matches are copied on every fork and
checkout, regardless of `--clean`, from your working tree — so a locally
modified tracked file carries its current contents.

Two layers, merged in ascending precedence (a later `!pattern` negation wins):

- **Project**: `.worktreeinclude` at the repo root. Commit it to share with the
  team.
- **User**: `worktreeinclude` in `~/.wt` (or `$WT_HOME`). Personal, cross-repo.

Neither is created for you; add one when you need it.

```gitignore
# .worktreeinclude
.env*
docker-compose.override.yml
!.env.ci
```

## Data locations

- Config: `~/.wt/config.yaml`
- Include patterns: `.worktreeinclude` (repo root) and `~/.wt/worktreeinclude`
- Worktrees: `~/.wt/worktrees/<repo-name>/<wt-name>/`
- Branch metadata: `git config branch.<name>.wt-parent`

## Claude Code plugin

```bash
/plugin marketplace add https://github.com/thurstonsand/wt.git
/plugin install wt@wt
```

Invoke with `/wt` or let Claude auto-invoke based on context.

The plugin also registers `WorktreeCreate` / `WorktreeRemove` hooks so Claude Code's `EnterWorktree` tool creates and tears down worktrees via `wt fork` and `wt rm`. The branches are preserved. If the requested name matches an existing branch — local or on `origin`, it is checked out with upstream tracking. Otherwise a new worktree is forked clean from `origin`'s default branch (resolved via `origin/HEAD`, falling back to `main` then `master`).

## pi extension

```bash
pi install npm:@thurstonsand/pi-wt
```

The extension provides `/mv <dir>` for moving a live session to any directory. When `wt` is installed, it also provides `/wt fork`, `checkout`/`co`, `rm`, `merge`, and `rebranch` for session-aware worktree operations. Moves preserve the conversation and session ID without restarting pi. It also includes the `wt` skill, which teaches the agent how to use the tool. See [`plugins/pi/README.md`](plugins/pi/README.md) for details.

---

## Command reference

### wt fork

```bash
wt fork feature-x            # fork HEAD, transfer dirty state
wt fork --clean feature      # fork HEAD without dirty state
wt fork -b develop fix       # fork from 'develop' (implies --clean)
wt fork --with file.txt fix  # copy an extra include pattern this fork
```

| Flag                  |                                                  |
| --------------------- | ------------------------------------------------ |
| `--clean`             | Don't transfer staged/unstaged/untracked changes |
| `--no-clean`          | Force dirty-state transfer (overrides config)    |
| `-b, --base <branch>` | Branch from a different base (implies `--clean`) |
| `--with <pattern>`    | Extra include pattern for this run (repeatable)  |

Dirty-state transfer moves staged changes (including partial hunks), unstaged changes, and untracked files to the new worktree.

`fork` always mints a _new_ branch. If the branch already exists (local or remote), it refuses, pointing you at `wt checkout`.

### wt checkout

```bash
wt checkout feature-x              # check out existing branch
wt co feature-x                    # alias
wt checkout -p main feature-x      # override the tracked parent
wt checkout --with file.txt feat   # copy an extra include pattern this run
```

| Flag                    |                                                                 |
| ----------------------- | --------------------------------------------------------------- |
| `-p, --parent <branch>` | Parent for merge/rebase tracking (defaults to origin's default) |
| `--with <pattern>`      | Extra include pattern for this run (repeatable)                 |

Unlike `fork`, this doesn't create a new branch. It checks out an existing one into a managed worktree, or returns the existing worktree if that branch is already checked out.

### wt merge

```bash
wt merge                     # from the worktree (default rebase)
wt merge feature-x           # from the parent (default rebase)
wt merge --squash            # squash all commits into one
wt merge --rebase            # fast-forward to rebased commits
wt merge --staged            # apply without committing, preserving dirty-state staging
wt merge --base main ext-wt  # merge into a specific branch
```

| Flag              |                                                                          |
| ----------------- | ------------------------------------------------------------------------ |
| `--squash`        | Squash all commits into one on parent                                    |
| `--rebase`        | Fast-forward parent to rebased commits (default)                         |
| `--staged`        | Apply without committing; preserve staged, unstaged, and untracked state |
| `-f, --force`     | Discard source dirt and allow protected branches                         |
| `--base <branch>` | Target branch to merge into (for external worktrees)                     |

Squash and rebase refuse a dirty source worktree unless `--force` is given. Dirty target state is stashed and restored automatically. Protected branches (main/master) default to `--staged` mode. On success, the worktree and branch are deleted. On conflict, both are preserved so you can resolve manually.

### wt rebase

```bash
wt rebase feature-x              # rebase onto parent branch
wt rebase                        # rebase current worktree
wt rebase --onto release feature # rebase onto a different branch
```

| Flag              |                                                        |
| ----------------- | ------------------------------------------------------ |
| `--onto <branch>` | Rebase onto a different branch (updates stored parent) |

Uncommitted changes are stashed and restored automatically. On conflict, use `git rebase --continue` or `--abort` in the worktree.

### wt rebranch

```bash
wt rebranch feature-2                # from inside a landed worktree
wt rebranch feature-2 -w wt1         # target another worktree by name
wt rebranch feature-2 --onto develop # re-seat onto a different baseline
wt rebranch feature-2 --move         # also rename the worktree directory
```

| Flag                        |                                                                 |
| --------------------------- | --------------------------------------------------------------- |
| `-w, --for-worktree <name>` | Worktree to rebranch (defaults to current)                      |
| `--onto <branch>`           | Baseline to rebranch onto (defaults to origin's default branch) |
| `--move`                    | Rename the worktree directory to match the new branch           |

When a worktree's branch has _landed_ — pushed, merged via PR/MR, and deleted on the remote — `wt rebranch` re-seats it onto a fresh baseline under a new branch, keeping the same directory by default and carrying uncommitted changes forward. With `--move`, git relocates the worktree to the new branch's sanitized directory name before dirty state is restored. The merged branch is left behind so that committed work is never destroyed; drop it later with `wt prune`.

### wt rm

```bash
wt rm feature-x              # remove worktree and delete branch
wt rm                        # remove current worktree (from inside it)
wt rm -f feature-x           # force-remove dirty worktree
wt rm --preserve-branch foo  # remove worktree, keep branch
```

| Flag                |                                                                  |
| ------------------- | ---------------------------------------------------------------- |
| `-f, --force`       | Force removal even if dirty (also force-deletes unmerged branch) |
| `--preserve-branch` | Keep the branch after removing the worktree                      |

When run from inside the worktree being removed, auto-cd's to the parent worktree (or main repo if no parent).

### wt prune

```bash
wt prune                 # scan; pick items to remove (stale dirs + landed branches pre-selected)
wt prune --all           # pre-select wt-managed branches too
wt prune --dry-run       # report stale dirs and prunable branches, remove nothing
wt prune --force         # remove stale dirs and landed branches without prompting
wt prune --all --force   # also remove wt-managed branches without prompting
```

| Flag            |                                                                 |
| --------------- | --------------------------------------------------------------- |
| `-n, --dry-run` | Report findings without removing                                |
| `-f, --force`   | Remove without prompting                                        |
| `--all`         | Pre-select wt-managed branches (otherwise listed but unchecked) |

Scans `~/.wt/worktrees/` for stale directories (broken git pointer, deleted source repo) and all tracked repositories for prunable branches not checked out in any worktree. A branch is prunable when it has _landed_ (its upstream was deleted on the remote) or carries `wt-parent` metadata. Repos are tracked automatically when you run any `wt` command, and the current repo is always included.

### wt list/ls

```bash
wt list       # worktrees for current repo
wt list -a    # worktrees across all repos
```

### wt path

```bash
wt path my-feature   # print worktree path
wt path              # print main repo path
wt path root         # same as bare `wt path`
```

### wt config

```bash
wt config show                       # show all settings
wt config get clean                  # get a value
wt config set merge squash           # set a value
wt config add post_create 'npm install'   # add to a list
wt config remove post_create 'npm install' # remove from a list (exact match)
wt config remove post_create         # show numbered entries
wt config remove post_create 0       # remove by index
wt config edit                       # open in $EDITOR
```

### wt shell

```bash
wt shell bash    # output bash integration script
wt shell zsh     # output zsh integration script
```

Eval the output in your rc file. See [shell integration](#shell-integration).

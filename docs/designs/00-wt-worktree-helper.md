# Plan: `wt` - Git Worktree Helper CLI

## Overview

Go CLI tool for simplified git worktree management with dirty-state forking, merge-back modes, and shell integration.

- **TODO**: import existing wt's into the tool somehow? or make it transparent somehow with no config?

## Directory Structure

```text
~/.wt/
  config.yaml                           # global config
  worktrees/<repo-basename>/
    <wt-name>/                          # worktree directory

<repo>/.git/
  wt-metadata/
    <wt-name>.yaml                      # metadata (inside git dir, discoverable)
```

**Repo discovery:** Uses `git rev-parse --git-common-dir` to find the shared git
directory, which works from the main repo or any of its worktrees. Metadata is
stored there, making it discoverable regardless of current directory.

**Worktree directories:** Stored under `~/.wt/worktrees/<repo-basename>/` where
`<repo-basename>` is derived from the git root directory name. This keeps
worktrees organized per-repo while allowing user-defined worktree names.

Note: If two repos have the same basename, worktree names must be unique across
both. This is an acceptable tradeoff for readable paths.

## Commands

| Command            | Description                                                    |
| ------------------ | -------------------------------------------------------------- |
| `wt fork [name]`   | Fork current work including staged/unstaged/partial-hunk state |
| `wt merge <name>`  | Merge back to parent (--squash/--rebase/--staged)              |
| `wt rm <name>`     | Remove worktree (fails if dirty, -f to force)                  |
| `wt list`          | List worktrees with branch, parent, dirty status               |
| `wt path <name>`   | Print worktree path                                            |
| `wt cd <name>`     | Change directory to worktree (requires shell integration)      |
| `wt rename <name>` | Rename worktree and/or its branch                              |
| `wt shell <shell>` | Output shell integration (bash, zsh)                           |
| `wt config`        | Manage config (list/get/set)                                   |

### wt fork behavior

- **Default:** Branch from HEAD, transfer dirty state (staged/unstaged/untracked)
- **With `--base <branch>`:** Branch from specified branch, implies `--clean`
- **`--clean`:** Create clean worktree without transferring dirty state
- **`--no-clean`:** Force dirty state transfer (error if used with `--base`)

Allowlist files (e.g., `.env*`) are always copied regardless of `--clean`.

## Project Structure

```text
wt/                                    # git root
├── cmd/wt/main.go
├── internal/
│   ├── cmd/           # cobra commands (root, fork, merge, rm, list, path, shell)
│   ├── config/        # GlobalConfig, WorktreeMetadata
│   ├── worktree/      # Manager, fork/merge/remove operations
│   ├── git/           # git command wrappers
│   ├── files/         # file copy with allowlist support
│   └── shell/         # shell integration scripts (bash, zsh)
├── go.mod
├── justfile
├── .golangci.yml
└── .gitignore
```

## Key Implementation Details

### Fork with Dirty State (Patch Approach)

```go
// Preserves staged vs unstaged, including partial hunks
stagedPatch := git.DiffCached()    // HEAD → index
unstagedPatch := git.Diff()        // index → working tree

// In new worktree (starting at same commit):
git.ApplyPatch(stagedPatch, cached=true)   // apply + stage
git.ApplyPatch(unstagedPatch, cached=false) // apply unstaged
```

Source repo never modified - only read patches. Safe.

### Merge Modes

- `--squash`: Squash all commits into one on parent
- `--rebase`: Rebase commits onto parent
- `--staged`: `git diff` from worktree, `git apply` to parent, no commit

**Protected branch handling:** When merging into main/master, default to `--staged`
mode to prevent direct commits to protected branches. Use `-f/--force` to override
and allow squash/rebase into protected branches.

**Conflict handling:** On conflict, behave like squash - leave worktree intact
for manual deletion via `wt rm`. Parent branch left in conflicted state for
manual resolution.

**Success detection:** After `git merge --squash` or `git rebase`, check exit
code. Exit 0 = success, delete worktree. Non-zero = conflict, keep worktree.

### File Copy Logic

```yaml
# ~/.wt/config.yaml (created with defaults on first use)
clean: false
allowlist:
  - .env*
  - docker-compose.override.yml
```

**Copy behavior:**

1. Load config from `~/.wt/config.yaml` (creates with defaults if missing)
2. Determine effective `clean` value:
   - Start with `config.clean` (default: false)
   - If `--base` specified: force `clean = true`
   - If `--clean` flag: force `clean = true`
   - If `--no-clean` flag: force `clean = false` (errors with `--base`)
3. Build file list:
   - Allowlist files: always included (filesystem walk matching patterns)
   - If `clean = false`: add untracked files from `git ls-files --others --exclude-standard`
   - Deduplicate
4. If `clean = false`: transfer dirty state (staged/unstaged patches)
5. Copy all files in single pass

**Logic summary:**

- `clean: false` (default): Copy staged changes, unstaged changes, AND untracked files
- `clean: true`: Copy nothing (clean worktree from commit)
- Allowlist files: ALWAYS copied regardless of `clean` setting
- `--base` flag: implies `--clean` (can't transfer dirty state to different base)

**Note:** Users can customize `allowlist` in config. Empty list `[]` disables
allowlist copying entirely.

### Shell Integration

```zsh
# eval "$(wt shell zsh)"
wt() {
  if [[ "$1" == "cd" ]]; then
    local dir="$(command wt path "${@:2}" 2>/dev/null)"
    [[ -n "$dir" && -d "$dir" ]] && cd "$dir" || { echo "wt: not found" >&2; return 1; }
  else
    command wt "$@"
  fi
}
# + _wt completion function with dynamic worktree/branch completion
```

### Metadata

Stored at `<git-common-dir>/wt-metadata/<wt-name>.yaml`:

```yaml
name: my-feature
worktree-path: ~/.wt/worktrees/myrepo/my-feature
branch: my-feature
parent-branch: main
created-at: 2024-01-26T10:30:00Z
```

Note: `worktree-path` is stored explicitly even though git tracks worktree
locations. This provides a single source of truth for wt and avoids parsing
git's internal structures.

Note: `source-worktree` field removed - not needed. Merge always targets
`parent-branch`, regardless of whether forked from main repo or another
worktree.

## Dependencies

All verified in use internally:

- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - config management
- `github.com/google/uuid` - name generation
- `github.com/bmatcuk/doublestar/v4` - glob matching (used in golangci-lint deps)
- `gopkg.in/yaml.v3` - YAML parsing

## Implementation Tasks

Each task is a committable unit with passing lint/tests.

### Task 1: Project Scaffold ✅

- Create `wt/` directory as git root
- `go mod init`
- `justfile` with recipes: `build`, `test`, `lint`, `coverage`
- `.golangci.yml` with comprehensive linters
- `.gitignore`
- `cmd/wt/main.go` with minimal root command (just `--version`)
- Basic test for root command
- **Commit:** "feat: initial project scaffold with lint/test setup"

### Task 2: Config Package ✅

- `internal/config/config.go` - GlobalConfig struct, Load/Save, defaults
- `internal/config/metadata.go` - WorktreeMetadata struct, Load/Save/Delete
- `internal/config/config_test.go` - unit tests for config operations
- `internal/config/metadata_test.go` - unit tests for metadata operations
- **Commit:** "feat: config and metadata management"

### Task 3: Git Wrapper (Basic) ✅

- `internal/git/git.go` - Git struct, run() helper
- `internal/git/branch.go` - CurrentBranch, DefaultBranch, IsDirty
- `internal/git/worktree.go` - WorktreeAdd, WorktreeRemove, WorktreeList
- `internal/git/git_test.go` - tests with temp git repos
- **Commit:** "feat: git command wrapper"

### Task 4: Worktree Manager Core ✅

- `internal/worktree/manager.go` - Manager struct, NewManager, path helpers
- `internal/worktree/list.go` - ListAll (scan metadata files)
- `internal/worktree/manager_test.go` - unit tests
- **Commit:** "feat: worktree manager core"

### Task 5: wt fork ✅

- `internal/cmd/fork.go` - fork command with --base, --clean flags
- `internal/worktree/fork.go` - Fork operation with patch-based state transfer
- `internal/git/branch.go` - added Checkout method
- `internal/git/diff.go` - DiffCached, Diff, ApplyPatch, CheckoutIndex
- Integration tests: fork with staged, unstaged, mixed scenarios
- **Commit:** "feat: wt fork command with dirty state preservation"

### Task 6: wt list, wt path ✅

- `internal/cmd/list.go` - list command with tabular output
- `internal/cmd/path.go` - path command
- Tests for both commands
- **Commit:** "feat: wt list and wt path commands"

### Task 7: wt rm ✅

- `internal/cmd/rm.go` - rm command with -f/--force
- `internal/worktree/remove.go` - Remove operation with dirty check
- Tests including dirty worktree rejection
- **Commit:** "feat: wt rm command"

### Task 8: File Copy with Allowlist ✅

- `internal/files/copy.go` - BuildFileList, CopyFiles
- `internal/files/copy_test.go` - tests for list building and copy logic
- `internal/worktree/files.go` - integration with fork operation
- `internal/git/branch.go` - add UntrackedFiles method
- **Commit:** "feat: file copy with allowlist support"

### Task 9: wt merge ✅

- `internal/cmd/merge.go` - merge command with --squash/--rebase/--staged
- `internal/worktree/merge.go` - Merge operation with conflict detection
- `internal/git/merge.go` - MergeSquash, Rebase, ApplyDiff helpers
- Tests for each mode + conflict handling
- **Commit:** "feat: wt merge command with squash/rebase/staged modes"

### Task 10: wt shell ✅

- `internal/cmd/shell.go` - shell command with bash/zsh support
- `internal/shell/wrapper.go` - Shell wrapper functions + Cobra completions
- Tests for shell output and arg validation
- **Commit:** "feat: wt shell command with bash/zsh support"

### Task 11: Polish & Documentation ✅

- Error message improvements
- Edge case handling (run from worktree, missing config, etc.)
- README.md with usage examples
- Increase test coverage to >80%
- Add Claude Code plugin for using the tool effectively
- **Commit:** "docs: README and polish"

### Task 12: wt config Command ✅

- `internal/cmd/config.go` - config command with subcommands
- `wt config show` - show all config values with descriptions
- `wt config get <key>` - get specific value with description and valid options
- `wt config set <key> <value>` - set config value in `~/.wt/config.yaml`
- `wt config edit` - open config in `$EDITOR` (or `vi`) and wait for close
- Supported keys: `clean` (bool), `merge` (squash/rebase/staged), `allowlist`
- Output includes hints: what the value means, valid options
- Use yaml.v3 Node API for `set` to preserve user comments
- Default config file includes comments describing each setting
- Tests for all subcommands
- **Commit:** "feat: wt config command with show/get/set/edit"

### Task 13: Per-Repo Allowlist Config

- Support `.wt.yaml` in repo root for repo-specific configuration
- Merge repo allowlist with global allowlist (add-only)
- Future: support `!<glob>` patterns to exclude from global allowlist
- **Commit:** "feat: per-repo .wt.yaml config"

## Verification

1. **Basic flow**: `wt fork foo` → `wt list` → `wt path foo` → `wt rm foo`
2. **Fork dirty state**: Make staged + unstaged changes → `wt fork` → verify
   state preserved in new worktree
3. **Partial hunks**: Stage only some hunks → `wt fork` → verify partial
   staging preserved
4. **Fork clean**: `wt fork --clean` → verify no dirty state transferred
5. **Fork with base**: `wt fork --base develop` → verify clean worktree from develop
6. **Merge back**: Make commits in worktree → `wt merge --squash` → verify
   changes on parent, worktree deleted
7. **Merge --staged**: `wt merge --staged` → verify changes appear staged on
   parent, no commit
8. **Conflict handling**: Create conflicting changes → `wt merge` → verify
   worktree preserved, parent conflicted
9. **Shell integration**: `eval "$(wt shell zsh)"` → `wt cd foo` → verify
   directory change + completions work
10. **Allowlist**: Create `.env` file → `wt fork --clean` → verify `.env` copied
    to new worktree despite --clean

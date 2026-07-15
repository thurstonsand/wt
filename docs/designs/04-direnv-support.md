# Design: Post-create hooks (direnv + custom)

## Problem Statement

When `wt fork` or `wt checkout` creates a new worktree directory, environment tools like direnv block until manually allowed. Other tools (mise, asdf) have similar trust/setup steps. Users want worktrees ready to use immediately without manual post-creation rituals.

## Design Decisions

Two hook layers: built-in + custom:

1. **Built-in hooks** — named config flags with smart behavior (precondition checks, friendly warnings). First built-in: `direnv`.
2. **Custom hooks** — `post_create` list of arbitrary shell commands, run in worktree dir via `sh -c`. User's responsibility to guard with preconditions.

Config example:

```yaml
direnv: true
post_create:
  - |
    [ -f package.json ] && npm install
  - |
    [ -f .tool-versions ] && mise trust
  - |
    [ -f pyproject.toml ] && pip install -e .
```

Multi-line scripts use YAML's literal block scalar (`|`). Each list entry is a plain `string` — the `|` is YAML syntax, not our concern. `sh -c` handles multi-line strings natively as scripts.

**Flat config, global scope**: No repo-specific config files. No nested `hooks:` section. Matches the existing flat `GlobalConfig` pattern (`clean`, `merge`, `allowlist`). Scripts self-guard using shell conditionals — same pattern as the built-in direnv hook (check `.envrc` → skip if absent).

**Non-fatal execution**: All hooks warn on failure, never block worktree creation. The worktree is valid regardless of hook outcome.

**Execution order**: Built-in hooks run first (direnv), then custom `post_create` commands in list order. Each command is independent — a failure in one does not skip subsequent hooks.

**No per-invocation flags**: These are set-and-forget config. No `--direnv` or `--no-hooks` CLI flags. Avoids flag bloat.

**Hook placement**: `Manager.RunPostCreateHooks()` is a public method called from the cmd layer after `Fork()`/`Checkout()` succeed. This keeps the worktree operations pure (no I/O side effects beyond git) and gives the cmd layer control over output.

### Built-in: direnv

- Check `cfg.Direnv` → if false, skip
- Check `.envrc` exists in worktree dir → if not, skip silently
- Run `direnv allow` in worktree dir
- On error: warn with context ("direnv not found" vs "direnv allow failed")

### Custom: post_create

- Iterate `cfg.PostCreate` list
- For each entry, run `sh -c "<command>"` with cwd set to worktree dir
- On error: warn with the command and error message, continue to next

## Edge Cases

| Scenario                                       | Behavior                                                      |
| ---------------------------------------------- | ------------------------------------------------------------- |
| `.envrc` not present in worktree               | direnv hook skips silently                                    |
| `direnv` binary not installed                  | Warn: "direnv not found in PATH, skipping"                    |
| `direnv allow` fails                           | Warn with error, continue                                     |
| Config `direnv: false` (default)               | Skip entirely                                                 |
| `post_create` command fails                    | Warn, continue to next command                                |
| `post_create` is empty (default)               | Skip entirely                                                 |
| `post_create` command references relative path | Runs with cwd = worktree dir, so relative paths resolve there |
| `post_create` guard fails (e.g. `[ -f x ]`)    | Silent — shell exits 1, we warn, move on                      |
| `sh` not available                             | Extremely unlikely; if so, warn and skip                      |

## Integration Points

1. **`config.GlobalConfig`** — new `Direnv bool` + `PostCreate []string` fields, `ConfigFields` entries
2. **`worktree.Manager`** — new `CommandRunner` interface, `RunPostCreateHooks()` method
3. **`internal/cmd/fork.go`** — call `RunPostCreateHooks()` after fork
4. **`internal/cmd/checkout.go`** — call `RunPostCreateHooks()` after checkout

## Implementation Plan

- [ ] **Add config fields** (`internal/config/config.go`)
  - Add `Direnv bool \`yaml:"direnv"\``to`GlobalConfig`
  - Add `PostCreate []string \`yaml:"post_create,omitempty"\``to`GlobalConfig`
  - Add `FieldMeta` entries for both to `ConfigFields`
  - `direnv`: key `"direnv"`, `parseBool`, default `false`
  - `post_create`: key `"post_create"`, `IsList: true`, `parseStringList`, default `[]`

- [ ] **Add `CommandRunner` to Manager** (`internal/worktree/manager.go`)
  - `CommandRunner` interface: `Run(dir, name string, args ...string) error`
  - Default impl: `execRunner` using `os/exec.Command` with `Dir` set
  - `WithCommandRunner()` functional option
  - `cmdRunner` field on `Manager`, defaulted in `NewManager()`

- [ ] **Add `RunPostCreateHooks()`** (`internal/worktree/hooks.go` — new file)
  - `func (m *Manager) RunPostCreateHooks(cfg config.GlobalConfig, wtPath string, w io.Writer)`
  - Direnv hook: check flag → check `.envrc` → run `direnv allow`
  - Custom hooks: iterate `cfg.PostCreate` → run each via `sh -c`
  - All errors → `fmt.Fprintf(w, "warning: ...")`, never return error

- [ ] **Wire up in cmd layer** (`internal/cmd/fork.go`, `internal/cmd/checkout.go`)
  - After successful `Fork()`/`Checkout()`, load config and call `mgr.RunPostCreateHooks(cfg, wt.WorktreePath, cmd.ErrOrStderr())`

- [ ] **Tests: config** (`internal/config/config_test.go`)
  - `Direnv` defaults to `false`
  - `PostCreate` defaults to empty
  - Both load correctly from YAML

- [ ] **Tests: hooks** (`internal/worktree/hooks_test.go` — new file)
  - Mock `CommandRunner` that records calls
  - direnv enabled + `.envrc` present → `direnv allow` called
  - direnv enabled + no `.envrc` → not called
  - direnv disabled → not called
  - `post_create` commands → each run via `sh -c`
  - Runner error → warning emitted, no error returned
  - Mixed: direnv + post_create → correct order (direnv first)

- [ ] **Document best practices** (README and/or `wt config --help`)
  - Explain that `post_create` scripts run in the new worktree directory
  - Recommend guarding scripts with file-existence checks for repo-specific behavior:

    ```yaml
    post_create:
      - |
        [ -f package.json ] && npm install
      - |
        [ -f .tool-versions ] && mise trust
      - |
        [ -f pyproject.toml ] && pip install -e .
    ```

  - Note that failures are non-fatal (warnings only)
  - Note that `direnv` has its own config flag with smart built-in behavior — no need to add `direnv allow` to `post_create`

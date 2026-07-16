# Pi extension: migrate the session with the worktree

## Status

Accepted

## Decision Summary

Ship a pi extension in `plugins/pi/` exposing a single `/wt` command that wraps the `wt` binary and adds the one thing `wt` cannot do from outside: move the _live pi session_ along with the work. Migration is a **move, not a fork** — the session file relocates to the destination's session store with its identity intact, the running pi process rebinds in-place via pi's `switchSession` runtime swap, and nothing is left behind. `wt` remains the sole authority for all git and filesystem operations; the extension never reimplements a worktree operation. One `wt` enhancement rides along: `wt rebranch --move`, which also renames the worktree directory to the new branch — a behavior that only becomes safe once sessions are mobile.

## Problem Statement / Background

A common flow: start an exploratory pi session in the main checkout, decide mid-session the work is real, `wt fork` it. The files move (dirty-state transfer), but the session does not — it lives in `~/.pi/agent/sessions/--<encoded-main-cwd>--/` and its runtime cwd is the main checkout. The result is split-brain: conversation history and agent context in one directory, work in another. Naive fixes (quit pi, re-launch in the worktree, fork the session across) lose the live process, strand session files, or grow artificial lineage.

The landing side has the same problem in reverse. The primary landing flow is: push a branch, PR gets squash-merged on the remote, `wt rm -f` locally. If the session lives in the removed worktree, it now points at a dead cwd and is invisible to `/resume` from the main checkout — orphaned. Worse, `wt` reuses worktree paths per branch name, so a future fork of the same name inherits the dead session store.

### Pi mechanics this design relies on (verified against pi v0.80.6)

- **Session store is keyed by cwd.** Sessions live in `~/.pi/agent/sessions/--<encoded-cwd>--/` (leading slash stripped, `/\:` → `-`). Listing (`/resume`) only sees the store for the current cwd.
- **cwd is runtime state, not process state.** pi never `chdir`s. Tools, file resolution, project resources (AGENTS.md, extensions, skills), and the footer all derive from `sessionManager.getCwd()`, which comes from the session file header.
- **`ctx.switchSession(sessionPath)`** (on `ExtensionCommandContext`) tears down the current runtime and rebuilds it in-process against the target session file — cwd, system prompt, and project resources rebind to the new header cwd. The TUI never exits. Post-switch work goes in the `withSession` callback; the pre-switch ctx is deliberately invalidated.
- **Appends are path-based.** pi writes session entries with `appendFileSync` against the stored path — no held file descriptor. A write after the file moves would resurrect a stub at the old path.
- **The system prompt embeds `Current working directory: <cwd>`** and is rebuilt on every runtime swap. Ground truth about location survives any rewind; no session entry is load-bearing for it.
- **Project trust walks parent directories** and the trust prompt offers "Trust parent folder". One approval of `WT_HOME/worktrees/<repo>` covers all future worktrees of that repo.
- **External session indexes key by session id.** The `pi-sessions` package indexes every session into SQLite with `session_id` as primary key and `session_path`/`cwd` as mutable columns, syncing on `session_start`/`turn_end`/`session_shutdown` via an id-keyed upsert. A moved file with a preserved id self-heals in the index one hook cycle after the swap; a re-minted id would strand a dead row per migration until a manual full reindex.

## Goals

- `/wt fork [name]` and `/wt checkout`/`co <branch>` move the live session into the new worktree in-process; the user continues as if the session had always been there.
- `/wt rm [-f] [name]` and `/wt merge [name] [flags]` bring the session home to the main checkout before the worktree disappears. `rm` is the primary landing path (remote squash-merge flow); `merge` is equally supported.
- `/wt rebranch <new-branch>` re-seats a landed worktree _and_ renames its directory to match, migrating the session to the renamed path — via a new `wt rebranch --move` flag.
- Migration preserves session identity: same session id, same filename, no synthetic lineage, nothing stranded in the source store.
- The model is told about the move through a custom message; its renderer gives the human a concise, dimmed switch result instead of exposing the full agent-facing notice.
- `/wt` argument completion mirrors the CLI's completions.

## Non-Goals

- Reimplementing any `wt` behavior in TypeScript. Dirty-state transfer, `.worktreeinclude`, merge strategies, branch guards, direnv — all stay in the binary. The extension shells out and surfaces stderr.
- Changing `wt fork`'s copy semantics. Fork copies dirty state and leaves the source dirty (WYSIWYG); the extension migrates the session, not git state.
- Migrating process environment. pi's env was captured at launch; a worktree with a different `.envrc` keeps the stale env until pi restarts. Known limitation, stated, not solved.
- Rescuing sessions orphaned by running `wt rm` outside pi. pi's own "stored cwd does not exist, continue here?" fallback is the recovery path.
- Wrapping the rest of the `wt` surface. `list`, `cd`, `prune`, `config` and friends gain nothing from pi integration; a terminal is one keystroke away.
- Supporting non-pi agents. Claude Code integration already exists in `plugins/wt`.

## Exposed Shape

### Command surface

One pi command, mirroring the CLI:

```pi
/wt fork [name] [flags]       # wt fork, then migrate session → worktree
/wt checkout <branch> [flags] # wt checkout, then migrate session → worktree
/wt co <branch> [flags]       # alias
/wt rm [-f] [name]            # migrate session → main checkout, then wt rm
/wt merge [name] [flags]      # wt merge, migrate session → main, then wt rm
/wt rebranch <branch> [flags] # wt rebranch --move, then migrate session → renamed dir
```

Flags are forwarded verbatim. Any other subcommand is an error: `/wt` carries exactly the commands that benefit from session integration, and everything else belongs in a terminal.

Argument completion delegates to cobra's hidden `wt __complete` protocol, so branch names, worktree names, and flags complete identically to the shell.

### Migration semantics (the MV)

On every migration, in order:

1. `waitForIdle()` — never touch the file mid-stream.
2. Run the `wt` operation and read its fd-3 cd directive, the same machine channel used by shell integration, to obtain the authoritative destination path.
3. Read the session file, rewrite the header `cwd` to the destination, write it into the destination's session store **under the same filename** — timestamp and session id survive; this is relocation, not lineage.
4. `ctx.switchSession(newPath, { withSession })` — in-process runtime swap.
5. Inside `withSession`: delete the old file, then inject a custom message: _"Session migrated to `<new>`; paths under `<old>` now resolve under `<new>`."_ Its renderer shows the human only a concise switch result. On landing commands the notice also states that the old worktree directory has been deleted. Destructive steps (`wt rm`) also run here, after the swap.

For `rm`/`merge`, migration home happens **before** worktree removal, and only when the session actually lives in the source worktree reported by `wt` (compared after canonicalization). `/wt rm other-branch` from the main checkout skips migration and just runs the binary.

### `wt rebranch --move` (new CLI capability)

`wt rebranch` keeps its directory today _because_ agent sessions had to stay next to the code. `--move` lifts that constraint: after re-seating the branch, the worktree directory is renamed to the new branch's sanitized name via `git worktree move` (wrap git, don't reimplement). Default behavior is unchanged — shell users opt in; a surprise cwd rename under an interactive shell would be hostile. The existing post-rebranch cd directive (`shell.PrintWithCD`) reports the renamed path, so shell integration follows automatically.

The rename happens **before** the rebase, so a conflict state is resolved at the final path — the cd directive and the pi session migration are correct immediately, once, regardless of outcome. A later abort does not un-rename, consistent with rebranch's existing stance that the spent branch is left for `wt prune`.

### Package layout

`plugins/pi/` is a pi package (`package.json` with the `pi` key), sitting parallel to the Claude Code plugin in `plugins/wt/`, published to npm:

```bash
pi install npm:pi-wt
```

## Design Decisions

### 1. Move, not fork

pi offers `SessionManager.forkFrom(source, targetCwd)`, which copies history into the target store — but it mints a new session id, records `parentSession`, and leaves the source file behind. N round trips would strand N files and grow an artificial lineage chain. The MV keeps identity: same id, same filename, untouched `parentSession` (whatever real lineage the session had survives, as parent paths are absolute). pi's session selector renders a session whose parent path is absent from the current store as a root — dangling references degrade gracefully. Ten round trips produce zero new tree nodes and zero leftovers.

Identity is also what external tooling keys on. `pi-sessions` upserts by session id, so an MV converges in its index automatically (path and cwd columns update in place, chunks and lineage stay attached); id churn would orphan a database row per migration. Anything else the user has holding a session uuid — search history, cross-session ask/messaging, sideshow references — survives an MV and dies with a fork.

The cost is one departure from exported API: serializing the (exported-API-parsed) entries back to JSONL when writing the relocated file. That format is versioned, documented in pi's `docs/session-format.md`, and migrated on load by pi itself — a documented contract, not hidden internals. Accepted.

### 2. Copy → switch → delete, in that exact order

Because appends are path-based, deleting or moving the old file _before_ the runtime swap is a race: any entry written during teardown (extensions persist state on `session_shutdown`) resurrects a stub at the old path. Writing the copy first, swapping, and deleting the original inside `withSession` closes the window — late appends land in a file that is about to be deleted, and the new runtime only ever appends to the new path.

### 3. Wrap wt, don't reimplement (tenet, extended)

The extension's git surface is exactly a no-shell `spawn("wt", ...)`, with fd 3 piped so it can consume the same cd directive as shell integration. Preflight — dirty targets, branch collisions, missing repos — is `wt`'s job; the extension surfaces the binary's stderr verbatim and aborts the migration if the exit code is non-zero. Two hidden integration flags preserve that authority while allowing the required ordering: `rm --validate-only` runs removal guards before migration, and `merge --defer-removal` lands changes without deleting the source until after the session swap. The only pi-side guards are: a persisted session must exist (`--no-session` runs have nothing to migrate), the agent must be idle, and the `wt` binary must be on PATH.

### 4. `rm -f` means `-f`

`/wt rm -f` on a worktree with stray local edits migrates the session home and removes the worktree, edits and all. The extension adds no confirmation layer: `/wt rm -f` must behave exactly like `wt rm -f` (least surprise; the CLI already owns the never-lie-on-destructive-actions contract). Without `-f`, `wt`'s own refusal propagates and the session stays put.

### 5. Migration notice is courtesy, not ground truth

The context is full of old absolute paths after a move (tool outputs embed them constantly), so a custom message telling the model paths have moved is cheap insurance against edits landing in the still-existing source checkout. But correctness does not depend on it: the system prompt carries the current cwd and is rebuilt on swap, so rewinding past the migration point loses only the courtesy notice, never the location.

### 6. Completions shell out to `wt __complete`

`getArgumentCompletions(prefix)` splits the prefix into words and invokes cobra's `wt __complete <words>... <partial>`, mapping the reply (candidate lines plus `:directive` suffix) to pi autocomplete items. One source of truth; new `wt` flags and subcommands complete in pi with no extension change.

### 7. Ordering on the landing path

`merge` must run before the session leaves (it operates from inside the worktree), and `rm` must run after the swap (it deletes the directory the old runtime was standing in). Hence: `wt merge` → migrate → swap → `wt rm` inside `withSession`. For plain `/wt rm`, the same tail applies without the merge step. Between the copy and the swap the destination store briefly holds a duplicate id; the swap and delete make it unobservable in practice.

### 8. `/wt rebranch` implies `--move`

A rebranch that keeps its directory has zero session consequence — there would be nothing for `/wt rebranch` to do, and `/wt` only carries commands that benefit from integration. So `/wt rebranch` always passes `--move`: `wt rebranch --move` → dir renamed → relocate the session store → swap. The parity argument from Decision 4 does not apply: `--move` is additive convenience, not an alteration of destructive semantics, and the plain behavior remains one terminal away.

## Edge Cases & Failure Modes

- **`wt` fails (fork collision, dirty target, not a repo):** no session mutation has happened yet; stderr is shown; session stays where it was.
- **Migration copy fails after a successful fork:** the worktree exists but the session did not move. Recovery is `/wt co <name>` — checkout of an existing worktree uses the same migration tail.
- **Swap cancelled** (another extension vetoes `session_before_switch`): the copied file is deleted, original remains authoritative; report and abort.
- **In-memory session (`--no-session`):** nothing to migrate; run the `wt` operation and tell the user to `wt cd` in a terminal if they want to follow.
- **Session forked inside pi while in a worktree, parent later migrates home:** the child stays in the worktree store and renders as a root there. Cosmetic; the child's history is intact.
- **`wt rm` run from a terminal instead of `/wt rm`:** the session strands in the dead store — pre-existing behavior, recoverable via pi's missing-cwd fallback. The extension cannot intercept out-of-band removals.
- **Another live pi session inside the worktree being removed:** its runtime survives (POSIX; cwd is an inode reference) but its next file operation fails. Same exposure as running `wt rm` today; not the extension's to solve.
- **Rebranch ends in conflict:** the dir was renamed and the session migrated before the rebase ran, so resolution proceeds in pi at the final path. An abort leaves the rename in place — honest, and consistent with rebranch never destroying committed work.
- **First fork of a repo:** pi prompts to trust the new directory once; "Trust parent folder (`WT_HOME/worktrees/<repo>`)" silences it for all future worktrees of that repo.
- **Stale process env:** a branch that changes `.envrc`/secrets does not re-evaluate into the live pi process. Documented limitation (Non-Goals).

## Rejected Alternatives

### Fork-based migration (`SessionManager.forkFrom` + delete)

Strands nothing only if the delete always succeeds, but still mints a new session id and a `parentSession` hop per migration — round trips accumulate lineage that means nothing, and every id-keyed external index (`pi-sessions`) accretes a dead row per migration. Rejected for identity loss (Design Decision 1).

### Rewrite the header in place, `switchSession` to the same file

Moves nothing: the file stays in the _source_ store (pi derives the store from the file's parent directory), so `/resume` from the worktree would never list it. The file must physically relocate.

### Relaunch pi in the worktree (`pi --resume` wrapper)

Loses the live process — the explicit constraint this design exists to satisfy.

### Upstream a `pi session move` feature

Cleanest long-term home for the MV primitive, but the extension works today on stable exported API (`SessionManager`, `switchSession`). Revisit if pi grows native support.

## Integration Points

- **`wt` CLI:** invoked as a subprocess; `fork`, `checkout`, `merge`, `rm`, `rebranch --move`, `__complete`. Destination paths come from the existing fd-3 cd directive. Go-side support includes the user-facing `--move` flag on rebranch, idempotent checkout, and hidden `rm --validate-only` and `merge --defer-removal` integration seams used to keep validation before migration and deletion after the swap. In deferred mode, `merge` reports its source worktree path on stdout and its landing path on fd 3; validation does the same for `rm`.
- **pi extension API (v0.80.x):** `registerCommand` with `getArgumentCompletions`, `ExtensionCommandContext.switchSession` + `withSession`, `waitForIdle`, `sendMessage`; `SessionManager` exported from the pi package for store-path derivation and entry parsing.
- **`pi-sessions` index:** no changes required; its id-keyed upsert absorbs migrations on the next hook sync.
- **`plugins/` precedent:** sits beside `plugins/wt` (Claude Code). The skill and hooks there are unaffected.

## Implementation Plan

1. **`wt rebranch --move` (Go):** rename via `git worktree move` before the rebase, updated `PrintWithCD`, completion, tests. Independently landable; useful from the shell without pi.
2. **Package scaffold:** `plugins/pi/` with `package.json` (`pi` key), TypeScript extension entry, lint/typecheck wiring consistent with repo tooling.
3. **Command core:** `/wt` command, arg splitting, subprocess execution, stderr surfacing, unsupported-subcommand errors, `__complete`-backed completions.
4. **Migration primitive:** store-path derivation, header rewrite, copy → `switchSession` → delete, migration notice. Covered by tests against fixture session files.
5. **Outbound commands:** `fork` and `co` wired to the primitive.
6. **Landing commands:** `rm` (primary) and `merge`, including the session-in-target detection, post-swap `wt rm`, and the directory-deleted notice.
7. **Rebranch command:** `/wt rebranch` → `wt rebranch --move` → migrate to renamed path.
8. **Smoke evidence:** scripted end-to-end run — fork from a real repo, confirm `/resume` visibility moves stores, confirm the `pi-sessions` row converges to the new path, land with `rm -f`, confirm the session sits in the main store with identical id and no leftovers.
9. **Publish:** npm release of the pi package.

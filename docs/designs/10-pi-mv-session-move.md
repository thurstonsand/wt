# Pi extension: `/mv` — move the session to any directory

## Status

Accepted

## Decision Summary

Add a second top-level command, `/mv <dir>`, to the existing pi plugin (`plugins/pi/`, published as `@thurstonsand/pi-wt`). It migrates the live session to an arbitrary directory's session store — no git, no `wt` binary involved. This reframes the plugin's domain as **session mobility**: `/wt` subcommands are the wt-aware movers, `/mv` is the manual one. The package keeps its name; its description and README shift to match.

The motivating workflow: start chasing a bug in repo A, discover the fix belongs in repo B, `/mv ~/Develop/repo-b` and keep working with full conversation context.

## Why not elsewhere

- **`/wt move`**: rejected. Every `/wt` subcommand mirrors a real `wt` CLI command and gets completions from `wt __complete`. A subcommand only the extension knows would break that invariant.
- **Separate package**: rejected. The ~150 lines of migration machinery (`migration.ts`, `session-file.ts`) would need duplication or a third package. Not worth it for one command.

## Mechanics (verified against pi v0.80.10 source and the existing extension)

Everything downstream of destination discovery already exists and is git-agnostic:

- `migrateSession(ctx, destination)` in `extensions/wt/migration.ts` snapshots the session, rewrites the header cwd, writes into the destination's session store, and calls `ctx.switchSession`. `/mv` calls it with no `MigrationOptions`.
- Pi's `switchSession` rebuilds the runtime with the new cwd — destination repo's AGENTS.md, project extensions, and trust flow engage normally. Cross-repo needs no extra work.
- The migration notice (`wt-session-migration` customType + registered renderer) is already git-agnostic text; reuse as-is.
- `writeMigration` already errors when source and destination stores are identical ("already stored for") — covers `/mv .`.

## Design decisions

1. **Command**: `/mv <dir>`, single required argument, registered unconditionally in `extensions/wt.ts`. `/wt` is registered only when the `wt` binary is installed. Pi surfaces a conflict if another extension claims `/mv`; acceptable.
2. **Destination validation**: must exist and be a directory. Nothing else — no git requirement. Fail fast with a clear error otherwise.
3. **Path expansion — the one real seam.** `canonicalPath` (`paths.ts`) resolves relative paths against `process.cwd()`, which diverges from the session cwd after any migration, and Node never expands `~`. `/mv` needs an expansion helper: `~`/`~/...` → `os.homedir()`, relative paths → resolved against `ctx.cwd`. Expansion happens **before** `migrateSession`. Put it in `paths.ts`.
4. **Autocomplete**: filesystem directory completion for `/absolute`, `~/home`, and `../relative` prefixes. Directories only. Must share the expansion helper with the execution path so suggestions and execution cannot disagree. Wire via `registerCommand`'s `getArgumentCompletions`; the cwd comes from the `session_start` capture already in `wt.ts` (note: that capture must stay current across migrations — verify `session_start` refires on switch, which it does via the `resume` session-start event).
5. **In-memory sessions**: guard with `isPersisted(ctx)` and warn, mirroring `outbound.ts`. Call `ctx.waitForIdle()` before executing, mirroring `command.ts`.

## Known limitations (document, don't solve)

- **Environment staleness**: pi's process env was captured at launch. Cross-repo this is heavier than cross-worktree (`.envrc`, PATH, secrets differ more). README's existing limitation section should call out `/mv` explicitly. _(Superseded: solved rather than documented — one probe shell fires the destination's directory hooks, and the captured delta is applied to `process.env`. The README section referenced above now describes the behavior instead of the limitation. See `plugins/pi/extensions/wt/env-probe.ts`.)_

## Test seams

Match existing patterns in `plugins/pi/test/` (vitest, pure-function level):

- path expansion helper: `~`, `~/x`, absolute, `../relative` against an injected cwd/home
- directory completion listing: prefix → candidates (temp-dir fixtures)
- `migrateSession` internals are already covered via existing command tests; do not re-test through `/mv`

Evidence on completion: live smoke test — run pi in repo A, `/mv` to repo B, show the session file relocated to repo B's store, old file gone, session id unchanged, and the migration notice rendered.

## Implementation Plan

### Phase 1 — `/mv` end-to-end

`extensions/wt/commands/move.ts` (validate → expand → `migrateSession`), expansion helper in `paths.ts`, registration in `wt.ts`, unit tests for expansion. Demonstrable: live cross-repo move.

### Phase 2 — directory completion

Completion function sharing the expansion helper, wired to the `/mv` registration, unit tests for candidate listing. Demonstrable: tab-completing `~/Dev`, `../`, `/Users/`.

### Phase 3 — reframe docs

`package.json` description → session mobility framing; README: document `/mv`, extend the environment limitation note. `mise run lint` for markdownlint.

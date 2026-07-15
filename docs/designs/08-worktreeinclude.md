# Adopt `.worktreeinclude` as the file-copy source

## Status

Accepted

## Decision Summary

Replace `wt`'s proprietary global `allowlist` config with the emerging
`.worktreeinclude` convention: gitignore-style pattern files, layered
project-then-user, resolved through git itself. The tradeoff is a deliberate
behavior shift — pattern matching moves from doublestar (root-anchored) to
gitignore semantics (unanchored, depth-matching) — accepted in exchange for a
shared, portable format and free reuse of git's ignore engine.

## Problem Statement / Background

`wt` copies certain files into every new worktree regardless of `--clean` — the
gitignored setup files that dirty-state transfer deliberately skips (`.env`,
`docker-compose.override.yml`, secrets). Today this is a global `allowlist` of
doublestar globs in `~/.wt/config.yaml`: one list for every repo, a
`wt`-proprietary format, matched against `os.DirFS(cwd)`.

Three problems:

1. **Proprietary and global-only.** The `allowlist` is invented here and cannot
   be committed alongside a repo or shared with a team. Codex and other tools
   have converged on `.worktreeinclude` — a repo-root file of gitignore-style
   patterns — for exactly this purpose. Sharing the file is worth more than
   owning the format.
2. **cwd-sensitivity (latent bug).** `wt` builds its git instance from
   `os.Getwd()` (`helpers.go:11`), so both the allowlist glob (`os.DirFS(cwd)`)
   and dirty-state transfer (`git ls-files` returning cwd-relative paths)
   silently assume you invoke `wt` from the repo root. Fork from a subdirectory
   and the wrong tree is matched.
3. **Reimplemented matching.** Allowlist matching uses the `doublestar` library
   and a hand-rolled directory walk, when git's own `ls-files` already resolves
   ignore-style patterns against the exact working tree — tenet: wrap git, don't
   reimplement.

### Why the include set must always run

Dirty-state transfer lists untracked files via `git ls-files --others
--exclude-standard` (`branch.go:163`), and `--exclude-standard` drops gitignored
files (proven by `TestUntrackedFilesRespectsGitignore`). So on a normal
(non-clean) fork, `.env` is **never** carried by dirty-state — the include set
is the only thing that brings it across. This is the crux: the include set is
not a clean-fork nicety, it carries the gitignored slice dirty-state skips, on
every create.

## Goals

- Copy files into new worktrees from committed project-level and personal
  user-level `.worktreeinclude` files, using gitignore syntax.
- Make the include set carry gitignored files on every create — clean fork,
  non-clean fork, and checkout.
- Support carrying a locally-modified tracked file into every worktree (a known
  anti-pattern the author must support).
- Make the whole fork operation cwd-independent.

## Non-Goals

- Backfilling or migrating existing `allowlist` config. Breaking change; the
  author migrates two patterns by hand.
- A CLI for editing include files. They are plain text at known locations.
- Auto-seeding a default user-level file.
- Auto-copying `AGENTS.override.md` (a Codex-specific behavior).
- Matching Codex's exact copy semantics (skip-symlinks, no-overwrite). We share
  the _file_, not the behavior.

## Exposed Shape

### File format (end-user surface)

Two layers, both gitignore syntax — one pattern per line, `#` comments, `!`
negations, blank lines ignored:

- **Project-level**: `<repo-toplevel>/.worktreeinclude`. Committed, shared.
- **User-level**: `<WT_HOME>/worktreeinclude` (no dot; it lives in `wt`'s own
  directory). Personal, cross-repo.

Neither is auto-created. Absent files contribute nothing.

### Matching contract

The set of copied files is:

```sh
git ls-files --cached --others -i --exclude-from=<merged> --full-name
```

run at the repo toplevel. `--cached --others` unions tracked + untracked; `-i
--exclude-from` filters that union to pattern matches; files are copied from the
**working tree**, so local modifications to tracked files ride along. Matches
are copied regardless of `--clean`.

### Precedence

Patterns are concatenated into one temp exclude file in ascending priority so
last-match-wins resolves correctly:

```diagram
project .worktreeinclude  →  user worktreeinclude  →  --with patterns
```

A user `!pattern` overrides a project include; `--with` (per-invocation)
overrides both.

### CLI

- `wt fork --with <pattern>` and `wt checkout --with <pattern>`: `<pattern>` is
  now a gitignore pattern (was a doublestar glob), highest precedence.
- `wt config`: the `allowlist` key is removed. No replacement.

### Internal boundary

- `internal/files`: doublestar glob + directory walk (`BuildFileList`) deleted;
  `CopyFiles` retained unchanged.
- `internal/git`: new method resolving the include set via `ls-files`.
- `internal/worktree`: `copyFiles` drops its allowlist argument; fork and
  checkout call the git-backed resolver. Dirty-state transfer reseated on the
  toplevel path.

## Design Decisions

### 1. Resolve everything at the repo toplevel

`git rev-parse --show-toplevel` locates the project `.worktreeinclude` and
anchors `ls-files --full-name` to root-relative paths. Dirty-state transfer is
reseated on the same toplevel path so the entire fork operation is
cwd-independent. This folds the latent cwd bug fix into the change rather than
leaving two path-resolution regimes. Orthogonal to `.worktreeinclude` strictly
speaking, but cheap to fix here and expensive to leave half-fixed.

### 2. `git ls-files -c -o -i` over a pattern library

Using git's own ignore engine gives gitignore syntax and the tracked+untracked
union for free, and keeps matching identical to how a developer would reason
about ignore patterns. Rejected adding a `go-gitignore` dependency: it would
reimplement what git already does and wouldn't resolve tracked vs untracked.

### 3. `--cached` is load-bearing

`--others` alone would miss tracked files. The author's supported anti-pattern —
a local, uncommitted modification to a tracked file that must appear in every
worktree — requires matching tracked files and copying their working-tree
content. `--cached` supplies the tracked half of the union; copying from the
working tree supplies the local mod. On a clean fork (no dirty-state transfer)
this is the _only_ path that carries such a file.

### 4. Single concatenated temp exclude file

One temp file with project → user → `--with` gives deterministic last-wins
precedence under our control, rather than depending on git's ordering across
multiple `--exclude-from` flags. `--with` strings (CLI, not files) also need a
file to live in; concatenation handles them uniformly.

### 5. Overwrite, don't skip

`CopyFiles` keeps `O_TRUNC`. The author wants an include-matched file to
overwrite whatever the checkout produced (the local-mod workflow depends on it).
Symlinks are skipped incidentally by `CopyFiles`' existing `IsRegular` guard; no
dedicated handling. This diverges from Codex's skip-existing rule by design.

### 6. `.gitignore` is orthogonal to the matcher

The matcher never passes `--exclude-standard`, so git does not consult
`.gitignore` (or `.git/info/exclude`, or global excludes) when resolving the
include set — the merged `--exclude-from` file is the sole authority for `-i`. A
gitignored file is copied iff it matches an include pattern; a gitignored file
that matches nothing is left behind. Mechanically independent, but complementary
by design: `.gitignore` governs what dirty-state transfer drops (via
`--exclude-standard`), and the include set exists to recover exactly that
dropped slice.

### 7. Remove the `allowlist` config entirely, no CLI replacement

`.worktreeinclude` is a standard format at known locations, editable directly by
human or agent. The only prior justification for `wt config add allowlist` was
that the format was proprietary and internally managed; that justification is
gone. Removing it deletes the global-config coupling rather than porting it.

## Edge Cases & Failure Modes

- **Staged deletion of an include-matched file:** `--cached` omits it (removed
  from index), working tree lacks it — not copied, no resurrection.
- **Tracked file deleted in working tree (unstaged):** `--cached` lists it, but
  the working-tree copy misses; `CopyFiles`' `os.Stat` guard skips it.
- **Non-clean fork overlap:** untracked-not-ignored files are copied by both
  dirty-state and the include set — harmless overwrite, not guarded.
- **Neither include file present, no `--with`:** empty exclude input, include
  set empty, copy is a no-op.
- **Depth-matching behavior change:** `.env*` was root-anchored under doublestar;
  under gitignore it matches at any depth. New nested `.env` files will now be
  copied. Write `/pattern` to re-anchor to root.

## Alternatives

### Keep the allowlist, add `.worktreeinclude` as a supplement (union)

- **Status:** Rejected.
- **Decision:** Lowest-friction, but leaves two formats and the proprietary
  global config in place. The author wants the old mechanism gone entirely
  ("boil the ocean"), so a superset that preserves the allowlist loses.

### `go-gitignore` matching library

- **Status:** Rejected.
- **Decision:** Reimplements git's ignore engine and still cannot distinguish
  tracked from untracked. `ls-files` does both natively.

### `wt config add/remove [--global] include` command

- **Status:** Rejected.
- **Retained discussion:** Considered a virtual config field (or dedicated `wt
include` verb) for deterministic, comment-safe mutation and agent operability.
  Rejected because the format is now standard and file-based; a command would
  re-introduce the coupling this design removes. Direct file editing suffices.

## Implementation Plan

Three phases. Phase 1 is a standalone prefactor; Phase 2 is additive and
tested-but-unwired; Phase 3 is the atomic swap that flips the mechanism and
removes the old one in a single reviewable unit. `.gitignore`-orthogonality,
precedence, and the tracked-file case are all pinned by tests.

- [ ] **Phase 1: Reseat the source repo on its toplevel (cwd bug fix)**
  - Goal: Make all source-repo git operations (dirty-state transfer today,
    include resolution later) resolve against the repo toplevel instead of the
    invocation cwd, so `wt fork` from a subdirectory behaves identically to
    forking from root.
  - Files: `internal/git/git.go` (new `Toplevel()`), `internal/worktree/manager.go`
    (reseat default `m.git`), `internal/git/git_test.go`,
    `internal/worktree/fork_test.go`.
  - Work:
    - Add `func (g *Git) Toplevel() (string, error)` — `rev-parse --show-toplevel`.
    - In `NewManager`, when `m.git` is not injected, resolve the toplevel of
      `repoDir` and build `m.git` against it. Leave injected git (tests)
      untouched.
  - Validation: `mise run test-int`; new test forks from a nested subdir and
    asserts dirty-state + files land root-relative, not cwd-relative.

- [ ] **Phase 2: Add the include resolver (additive, unwired)**
  - Goal: Land and test the git-backed matcher and the user-include path lookup
    without changing fork/checkout behavior yet.
  - Files: `internal/git/branch.go` (or a new sibling file), `internal/git/*_test.go`,
    `internal/config/global_store.go`.
  - Work:
    - Add `func (g *Git) FilesMatchingPatterns(patterns []string) ([]string, error)`:
      write `patterns` to a temp exclude file, run
      `ls-files --cached --others -i --exclude-from=<temp> --full-name`, parse
      lines, clean up the temp file. Empty `patterns` short-circuits to nil.
    - Add `GlobalStore.UserIncludePath()` returning `s.pathOf("worktreeinclude")`.
  - Validation: `mise run test`; resolver tests cover tracked+untracked union,
    a `!negation` override (last-wins), a gitignored-but-unmatched file being
    excluded (`.gitignore`-orthogonality), and a locally-modified tracked file
    resolving with its working-tree content.

- [ ] **Phase 3: Swap the mechanism and remove the allowlist (atomic)**
  - Goal: Replace allowlist-based copying with `.worktreeinclude` resolution in
    both fork and checkout, delete the allowlist config and the doublestar path,
    and bring docs in line — in one commit so no intermediate state has both or
    neither.
  - Files: `internal/worktree/files.go`, `internal/worktree/fork.go`,
    `internal/worktree/checkout.go`, `internal/files/copy.go`,
    `internal/config/config.go`, `internal/cmd/fork.go`,
    `internal/cmd/checkout.go`, `go.mod`/`go.sum`, `README.md`,
    `plugins/wt/skills/wt/SKILL.md`, and the corresponding `_test.go` files.
  - Work:
    - Rewrite `worktree.copyFiles` to assemble merged patterns in precedence
      order — project `<toplevel>/.worktreeinclude`, user
      `GlobalStore.UserIncludePath()`, then `--with` — skipping absent files, and
      call `git.FilesMatchingPatterns`, then `files.CopyFiles`. Drop the
      `allowlist`/`untracked` glob arguments.
    - Keep the dirty-state untracked copy in `fork.go` as its own pass.
    - Delete `files.BuildFileList`, the doublestar import, and the untracked
      directory walk; retain `CopyFiles` unchanged.
    - Remove the `allowlist` field from `GlobalConfig`, its `ConfigFields`
      entry, and its `DefaultConfig` seed.
    - Reword `--with` help text (fork + checkout) from "allowlist" to include
      patterns; drop "allowlist" from `fork` long help.
    - `go mod tidy` to drop `doublestar`.
    - Update `README.md` (config section, `--with` rows, examples) and
      `SKILL.md` (Allowlist note, `--with` comments) to describe
      `.worktreeinclude` / `worktreeinclude`.
    - Rework tests: delete `files/copy_test.go` `BuildFileList` cases and the
      allowlist config tests; add fork/checkout integration proving a gitignored
      `.env` and a locally-modified tracked file both land on a `--clean` fork,
      and that `--with` overrides a user-level negation.
  - Validation: `mise run test-int && mise run lint`; manual smoke — create
    `.worktreeinclude` with `.env*`, `wt fork --clean`, confirm `.env` is present
    in the new worktree and absent when the pattern is removed.

# wt rebranch

## Status

Accepted

## Decision Summary

`wt rebranch <new-branch>` re-seats a **landed** worktree onto a fresh baseline (`origin/<default>`) under a new branch, in the same directory, carrying its uncommitted dirty-state forward. The spent branch is left behind for `prune`. The hard part — and the reason this design is worth recording — is that you *cannot* reliably detect whether a worktree's committed work already landed (squash-merge destroys commit identity; others' merges pollute tree comparisons), so the design deliberately avoids needing to: it carries only uncommitted state and never deletes the old branch.

## Problem Statement

A common parallel-stream lifecycle leaves a worktree stranded:

1. Fork `wt1` off `main`, do work.
2. Commit, push, open a PR.
3. Keep working in `wt1` — more uncommitted changes accumulate.
4. PR merges (squashed), remote branch deleted.
5. Local `wt1` branch is now orphaned from its upstream, `main` has advanced, but the worktree still holds uncommitted work the user wants to continue.

Today the only path forward is manual git surgery (reset onto fresh main, salvage the delta) — exactly the ugliness `wt` exists to hide. There is no command that says "this stream landed; rebaseline me and let me keep going."

This is the inverse of `prune`: `prune` handles *orphaned branch, no worktree*. This is *landed branch, live worktree, live uncommitted work* — nothing should be deleted, the work should be carried forward.

## Goals

- One command to continue working after a PR lands, from a fresh baseline, in the same directory.
- Preserve the directory name so an attached Claude Code session keeps working from the path it knows.
- Carry uncommitted work (staged, unstaged, untracked) forward with WYSIWYG fidelity.
- Never destroy committed work, even when we cannot prove it landed.
- Surface the "landed" state in `wt list` so it is discoverable.

## Non-Goals

- Automatically carrying committed-but-unmerged ("stray") commits. Deferred to a future `--from <sha>`.
- Detecting *which* commits landed vs. are stray. Proven impossible (see Design Decision 2).
- A `--flatten` mode. Explored and rejected (see Rejected Alternatives).
- Cross-repo or batch rebranch.

## Design Decisions

### 1. Command shape: `wt rebranch <new-branch> [-w <worktree>] [--onto <branch>]`

`rebranch` swaps the branch under a worktree, rebaselined onto fresh parent. The new branch name is explicit and required (like `fork`/`checkout` taking a name). Default target is the current worktree (cwd); `-w`/`--for-worktree` targets another by name. By default the baseline is the remote's default branch; `--onto <branch>` overrides it (accepts a local branch, a remote-tracking ref, or a bare name resolved against the remote), mirroring `wt rebase --onto`.

The verb won over `resync`/`renew`/`refork`: it is honest about the mechanic (a new branch ref replaces the old one under the same directory) and will not be confused with `rebase` (which preserves your commits). The target flag is spelled `--for-worktree` (not `--in`) because "in" reads ambiguously — it could mean "carry these changes *into* another worktree" rather than "operate *on* this worktree."

### 2. The impossibility result that shapes everything

You cannot reliably detect whether a worktree's committed work has already landed. Verified empirically against real git:

- **Squash-merge destroys commit identity.** `git cherry`, `git log main..HEAD`, and patch-id matching all report a landed-via-squash commit as still-unmerged. The squashed commit on `main` has a different hash and a combined patch.
- **Others' merges pollute tree comparison.** `git diff --quiet main HEAD` reports differences whenever another contributor landed work after your PR — files they added exist in `main` but not your branch, so a tree diff reads them as *your deletions*. False positive for "you have stray commits," and actively dangerous if acted on.

Because no query separates "your committed work landed" from "your committed work is unmerged," the design avoids needing the distinction: it carries only **uncommitted** state and **never deletes** the old branch. If committed stray work existed, it survives on the spent branch.

### 3. Mechanic: stash-based dirty-state transfer

```sh
git fetch origin --prune
git stash push -u              # capture staged + unstaged + untracked
git switch -C <new-branch> origin/<default>
git stash pop --index          # replay the delta, preserving staged/unstaged split
git config branch.<new>.wt-parent <default>   # via SetBranchMeta
```

- **Why stash, not the staged-merge flow:** `merge --squash` (used by `wt merge --staged`) moves *committed* content between branches. `rebranch`'s premise is that committed content already landed — there is nothing to squash-merge. The thing to carry is the uncommitted delta, which `stash -u` captures and `--squash` ignores entirely.
- **Why it survives others' merges:** `stash` keeps all committed work (including the landed commits) as the *common base*. `switch -C` brings in fresh parent's full tree, including other contributors' files. `pop` replays only the uncommitted delta on top. No tree comparison happens, so those files never masquerade as deletions. Verified.
- **`--index` preserves the staged/unstaged split** (`MM` stays `MM`), matching the WYSIWYG-transfer tenet `fork` upholds. If the index cannot cleanly reapply, fall back to plain `pop` (collapses to unstaged) and warn.

### 4. Baseline: `origin/<default-branch>`

Rebaseline onto `origin/<default>`, resolving `<default>` exactly as the `WorktreeCreate` hook does: `origin/HEAD` symbolic ref → `main` → `master`. Not the local parent branch (which may lag the remote). `rebranch` fetches first so the baseline is current.

### 5. Refuse if not landed

`rebranch` is meaningful only against a landed worktree. The gate uses local state, no network beyond the initial fetch:

- **Never pushed** (`branch.<name>.merge` unset) → refuse: "wt1 was never pushed; nothing landed."
- **Upstream still exists** (`refs/remotes/origin/<name>` present after fetch+prune) → refuse: "wt1 is not landed (origin/wt1 still exists)."
- **Landed** (`branch.<name>.merge` set AND tracking ref gone) → proceed.

### 6. Spent branch is left behind, never deleted

After rebranch, the old branch still exists locally (now orphaned from any worktree). This is the safety valve for the impossibility result: any committed stray work survives there. `prune` already detects and reaps such branches (`FindOrphanedBranches`). `rebranch` prints where the old work lives.

### 7. Folder/branch name drift is a normal state

The directory keeps its old name (`wt1`) while the branch becomes `<new-branch>`. This is intentional — the attached Claude session continues from the path it knows. Verified that all resolvers already tolerate the mismatch: `Manager.Get` matches folder name *or* branch name and returns `WorktreePath`; `wt path`, `wt cd`, the `WorktreeCreate` hook (`wt path "$name"`), `rm`, `merge`, and `rebase` all route through it without re-deriving folder from branch. No routing changes required. Cosmetic only: `list` shows the folder name under `NAME` and the new branch under `BRANCH`.

### 8. List surfaces landed via a `STATE` column

Rename the `DIRTY` column to `STATE`, carrying comma-joined words: `dirty`, `landed`, or `dirty,landed`. Landed is detected from local state (decision 5) with no network at list time — freshness reflects the last fetch, which is acceptable. Applies to both `wt list` and `wt list -a`.

## Edge Cases & Failure Modes

- **`stash pop` conflict** (uncommitted edit overlaps a change that landed differently in the squash): follow the `rebase` model — leave the worktree mid-pop with conflict markers, print resolve instructions. Safe because the old branch is retained; nothing is lost. Do *not* roll back.
- **`pop --index` cannot reapply cleanly:** fall back to plain `pop`, warn that the staged/unstaged split was collapsed.
- **No uncommitted changes at all:** rebranch still re-seats onto fresh parent under the new branch (a clean rebaseline). Valid, not an error.
- **Worktree not landed:** refuse with the specific reason (decision 5).
- **New branch name already exists:** refuse, as `fork`/`checkout` do (`ErrWorktreeExists` / branch-exists check).
- **Default branch unresolvable on origin:** refuse with the same message the create hook uses.
- **Stray committed work the user wanted:** survives on the spent branch; recoverable manually or via a future `--from`. Documented in the success message.

## Rejected Alternatives

### `--flatten` (reset to merge-base, fold commits into the delta, replay)

Intended as a no-SHA way to carry committed stray work by dissolving it into uncommitted state. Rejected after empirical testing: it **conflicts in exactly its target case**. Folding a landed commit back into the delta and replaying it onto a parent that already contains that commit (positioned differently by the squash) produces a merge conflict. Its entire selling point ("no boundary needed, no conflicts") evaporates; it degrades to "`--from` without the ability to choose the boundary, and you still eat conflicts." Strictly worse than `--from`.

### `reset --soft <fresh-parent>` flatten

Rejected: stages *deletion* of files another contributor landed (their files exist in parent, absent from your tree, so the soft reset's tree-diff reads them as your deletions). Silent data loss of others' merged work. Verified.

### Detect-and-refuse on stray commits via tree-diff gate

Rejected: no reliable gate exists (decision 2). `git diff --quiet parent HEAD` false-positives when another contributor has merged; identity-based tools false-positive under squash. Leaving the branch behind (decision 6) sidesteps the need for a gate entirely.

### `wt merge --staged` machinery for the transfer

Rejected: squash-merge moves committed content, but rebranch's committed content already landed. Wrong inputs (ignores uncommitted/untracked) and re-introduces the overlap conflict. Stash is the correct tool.

### Delete the spent branch after rebranch

Rejected: would destroy committed stray work with no recovery, and cannot be made safe given the impossibility result. Violates "never lie on destructive actions."

## Integration Points

- **`Manager.Get` / resolvers:** unchanged — already match folder-or-branch (verified). Folder/branch drift is now expected.
- **`WorktreeCreate` hook (`plugins/wt/scripts/worktree-create.sh`):** unchanged — routes via `wt path "$name"`, works for both old folder name and new branch name.
- **Default-branch resolution:** mirror the hook's logic (`origin/HEAD` → main → master); extract a shared helper in the git package if not already present.
- **`prune` (`FindOrphanedBranches`):** the spent branch becomes its input. No change needed; it already skips protected branches and active worktrees.
- **`wt list` (`internal/cmd/list.go`):** `DIRTY` → `STATE` column for both single-repo and `-a` paths.
- **Branch metadata (`SetBranchMeta` / `wt-parent`):** the new branch records `<default>` as parent.
- **Completion (`completeWorktreeNames`):** `--for-worktree`/`-w` should complete worktree names; surfaces folder names (already the case).

## Implementation Plan

- [x] Phase 1: Landed detection + git plumbing
  - Goal: Local-state primitives for "landed" and default-branch resolution, no command wiring yet.
  - Files: `internal/git/branch.go` (+ test), possibly `internal/git/merge.go` for fetch reuse.
  - Work: Add `UpstreamRef(branch)` (reads `branch.<n>.merge`), `RemoteTrackingExists(branch)`, and `IsLanded(branch)` = upstream-configured AND tracking-ref-absent. Add/extract `DefaultRemoteBranch()` resolving `origin/HEAD` → main → master (reuse for the create-hook parity).
  - Validation: `go test ./internal/git/...`; unit tests cover pushed-and-deleted, never-pushed, upstream-still-present.

- [x] Phase 2: Rebranch core in the worktree manager
  - Goal: `Manager.Rebranch(opts)` performing fetch → landed-gate → stash → switch -C → pop --index → set parent meta, returning a result struct.
  - Files: `internal/worktree/rebranch.go` (+ test), `internal/worktree/errors.go` (new sentinels: `ErrNotLanded`, `ErrNeverPushed`).
  - Work: Implement the mechanic with conflict handling (rebase-style preserve+instruct on pop conflict; `--index` fallback). Leave the old branch intact. Reuse `resolveWorktree` for `--for-worktree`/cwd targeting.
  - Validation: `go test -tags integration ./internal/worktree/...` with real worktrees covering: dirty-only happy path, independent parent commits preserved, never-landed refusal, name-collision refusal, pop conflict preserves worktree + old branch.

- [x] Phase 3: CLI command
  - Goal: `wt rebranch <new-branch> [-w <worktree>] [--onto <branch>]` wired to the manager, with help text and completion.
  - Files: `internal/cmd/rebranch.go` (+ test), `internal/cmd/root.go` (register).
  - Work: Cobra command, arg validation, `-w`/`--for-worktree` flag with worktree-name completion, `--onto` baseline override, success/refusal output including where the spent branch lives and how to prune it. Emit a `cd` hint to the (unchanged) path via `shell.PrintWithCD`.
  - Validation: `go test ./internal/cmd/...`; manual `just run rebranch ...` against a scratch repo.

- [x] Phase 4: List STATE column
  - Goal: Rename `DIRTY` → `STATE` with `dirty`/`landed`/`dirty,landed` values.
  - Files: `internal/cmd/list.go` (+ test).
  - Work: Compute landed per worktree via Phase 1 primitives; join with dirty into the cell, both `list` and `list -a`.
  - Validation: `go test ./internal/cmd/...`; golden/table assertions for each state combination.

- [x] Phase 5: Docs & integration surfaces
  - Goal: User-facing docs and agent surfaces reflect rebranch.
  - Files: `README`/help, `plugins/wt/skills/wt/SKILL.md`, `CLAUDE.md` feature list, completion if needed.
  - Work: Document the lifecycle (PR lands → rebranch → continue; spent branch → prune). Note folder/branch drift as expected. Update the features list in `CLAUDE.md`.
  - Validation: `just lint` (markdownlint); skim SKILL.md for agent-operability.

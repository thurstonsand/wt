# Design: Replace diff/apply with file copy in fork

## Problem Statement

`wt fork` transfers dirty state (staged/unstaged changes) to new worktrees by capturing text diffs and replaying them with `git apply`. This breaks on binary files — git produces invalid patches without `--binary`, and massive base85 strings with it. The current fix detects binaries and routes them through a separate copy path, creating two parallel pipelines for the same operation.

`CheckoutIndexTo` (extracting index content to disk) and direct file copy already handle both text and binary files uniformly. The diff/apply pipeline can be replaced entirely with file operations, eliminating the binary/text split and the need to hold diffs in memory.

## Design Decisions

**File operations replace patches**: Instead of capturing diffs as strings and applying them, list changed files by name and copy their content directly. `CheckoutIndexTo` extracts staged versions from the index. `files.CopyFiles` copies unstaged versions from the working tree. Both are content-type agnostic.

**Explicit deletion handling**: The diff/apply approach handled deletions implicitly through the patch format. The copy approach requires explicit handling — `git rm` for staged deletions (removes file and stages the removal), `os.Remove` for unstaged deletions (removes file without touching the index).

**`--diff-filter` for file categorization**: `git diff --name-only --diff-filter=d` lists non-deletion changes (added, modified). `git diff --name-only --diff-filter=D` lists deletions only. This is the same pattern already used in `HasConflicts()` (`merge.go:86`).

**`CheckoutIndex()` eliminated**: The current flow calls `CheckoutIndex()` (`-a -f`) to materialize staged files into the working tree after `ApplyPatch`. With `CheckoutIndexTo`, files are written directly to disk and then staged with `Add` — the working tree and index are already in sync. No bulk checkout needed.

## Edge Cases

**File with both staged and unstaged changes**: `CheckoutIndexTo` writes the index (staged) version, `Add` stages it, then `CopyFiles` overwrites the working tree with the source working tree (unstaged) version. The worktree ends up with index = staged version, working tree = unstaged version. Correct.

**Staged deletion + unstaged re-creation**: Unlikely but valid. The deletion appears in staged deletions, the file appears in unstaged non-deletions. `git rm` in the worktree removes and stages the deletion, then `CopyFiles` recreates the file as an unstaged change. Correct.

**New file (staged add)**: `CheckoutIndexTo` creates the file on disk, `Add` stages it. Straightforward.

**No dirty files**: All four `DiffNameOnly` calls return empty lists. Every operation short-circuits on empty input. No-op, same as today.

## Removed Code

- `DiffBinaryFiles` / `parseBinaryPaths` — binary detection no longer needed
- `mergeUnique` — was only used to combine binary file lists
- `applyStagedBinaries` — subsumed by the general staged file path
- `ApplyPatch` / `CheckoutIndex` — only callers are fork.go and tests; deleted entirely
- `DiffCached` / `Diff` exclude params — revert to zero-param signatures
- Associated tests for all removed methods

## Integration Points

1. **`internal/git/diff.go`** — new `DiffNameOnly` method, new `Remove` method
2. **`internal/worktree/fork.go`** — rewritten `!clean` block, removed binary helpers
3. **`internal/files/copy.go`** — existing `CopyFiles`, no changes needed

## Implementation Plan

- [ ] **Add `DiffNameOnly(cached bool, filter string) ([]string, error)`** (`internal/git/diff.go`)
  - `git diff --name-only [--cached] [--diff-filter=<filter>]`
  - Returns file paths, empty slice when no changes

- [ ] **Add `Remove(paths []string) error`** (`internal/git/diff.go`)
  - `git rm -- <paths...>`
  - Short-circuits on empty input

- [ ] **Add tests for new git methods** (`internal/git/diff_test.go`)
  - `TestDiffNameOnly` — staged/unstaged, with filter
  - `TestRemove` — stages file removal

- [ ] **Rewrite fork `!clean` block** (`internal/worktree/fork.go`)
  - Replace diff capture + binary detection with four `DiffNameOnly` calls
  - Replace `ApplyPatch` + `applyStagedBinaries` + `CheckoutIndex` with `CheckoutIndexTo` + `Add` + `Remove`
  - Replace unstaged `ApplyPatch` + `CopyFiles(bins)` with `CopyFiles` + `os.Remove`
  - Remove `applyStagedBinaries`, `mergeUnique`

- [ ] **Remove dead code** (`internal/git/diff.go`, `internal/git/diff_test.go`)
  - Delete `ApplyPatch`, `CheckoutIndex`, `DiffBinaryFiles`, `parseBinaryPaths`
  - Delete their tests: `TestApplyPatch*`, `TestCheckoutIndex`, `TestRoundTripStagedAndUnstaged`, `TestDiffBinary*`
  - Revert `DiffCached`/`Diff` to zero-param signatures, delete `TestDiffCachedExcluding`/`TestDiffExcluding`

- [ ] **Update fork tests** (`internal/worktree/fork_test.go`)
  - Add `TestForkPreservesStagedDeletion`
  - Add `TestForkPreservesUnstagedDeletion`
  - Existing tests (`TestForkPreservesStagedAndUnstagedSeparately`, binary tests) serve as regression gates

- [ ] **Verify full suite** — `just test-int && just lint`

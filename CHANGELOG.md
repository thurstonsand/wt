<!-- markdownlint-disable MD024 -->

# Changelog

## v1.12.0

### Added

- The `@thurstonsand/pi-wt` extension now moves live pi sessions with worktrees through `/wt fork`, `checkout`/`co`, `rm`, `merge`, and `rebranch`.
- Signed and notarized release binaries are available through GitHub Releases and Homebrew.
- `wt rebranch --move` renames a re-seated worktree directory to match its new branch.

### Changed

- `wt checkout` now returns an existing worktree for the requested branch, allowing session migration to recover after a worktree was created successfully but its session copy failed.

## v1.11.0

### Changed

- `wt prune` now flags landed branches — pushed, merged, and deleted on
  the remote — as prunable, not only branches carrying parent metadata.
  `wt checkout` now records a parent branch, so branches you check out
  become visible to prune later.
- Shell completion offers branch names instead of sanitized folder
  names, so candidates match the branch you actually work with (slash
  branches, or a worktree re-seated by `wt rebranch`).

### Fixed

- Forking or checking out from inside a worktree now places the new
  worktree under the correct repository folder, instead of re-nesting it
  under the current worktree's name.

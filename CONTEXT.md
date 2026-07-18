# CONTEXT.md

## Cast

- **Me** / **I**: The author of `wt` and its primary human user.
- **You**: The coding agent working on `wt` itself.
- **The agent**: A coding agent that _uses_ `wt` to manage its own worktrees — via the Claude Code plugin hooks (`EnterWorktree`) or the bundled skill (`SKILL.md`). Distinct from **you**.

## Language

- **Worktree**: A git worktree managed by `wt`.
- **Fork**: Creating a new branch + worktree off the current HEAD (or a chosen base).
- **Dirty-state transfer**: Carrying staged, unstaged, and untracked changes from the source into a newly forked worktree. Respects `.gitignore`, so ignored files are excluded.
- **.worktreeinclude**: A `.gitignore`-syntax pattern naming files always copied into a new worktree, regardless of `--clean`. Sourced from a project-level `.worktreeinclude` (repo root, committed) and a user-level `worktreeinclude` (in `WT_HOME`).
- **Merge-back**: Landing a worktree's commits onto its parent branch.
- **Parent**: The branch a worktree was forked from or tracks for merge/rebase.
- **Parallel stream**: An independent line of work, each in its own worktree, typically headed for its own MR.
- **Landed**: A worktree whose branch was pushed, merged via PR/MR, and deleted on the remote.
- **Rebranch**: Re-seating a **landed** worktree onto a fresh baseline under a new branch, keeping the same directory and carrying its uncommitted dirty-state forward.
- **Session migration**: Relocating a live pi session into another directory's session store.

## Relationships

- A **fork** produces one **worktree** on a new branch, recording its **parent**.
- A **worktree** merges back onto its **parent**.
- A **rebranch** re-seats a **landed** **worktree** on a new branch off fresh **parent**.
- In the pi extension, `/mv` performs a **session migration**, while `/wt` operations also execute the associated **worktree** action.

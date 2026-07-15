#!/usr/bin/env bash
set -euo pipefail

# Claude Code WorktreeCreate hook — creates a wt-managed worktree from origin's default branch.
# Stdin: JSON { name, cwd, ... }
# Stdout: absolute worktree path (one line, nothing else)

input=$(cat)
name=$(echo "$input" | jq -r '.name')
cwd=$(echo "$input" | jq -r '.cwd')

cd "$cwd"

# Refresh remote-tracking refs so the branch checks below see origin's latest.
# Non-fatal: offline checkout of an existing local branch should still work.
git fetch origin >&2 2>/dev/null || true

# If the worktree already exists (e.g. --resume), just return its path.
if path=$(command wt path "$name" 2>/dev/null); then
  echo "$path"
  exit 0
fi

# If the name matches an existing branch, check it out directly.
if git show-ref --verify --quiet "refs/heads/$name" ||
   git show-ref --verify --quiet "refs/remotes/origin/$name"; then
  command wt checkout "$name" >&2
  command wt path "$name"
  exit 0
fi

default_branch=""
if ref=$(git symbolic-ref --quiet refs/remotes/origin/HEAD 2>/dev/null); then
  default_branch="${ref#refs/remotes/origin/}"
elif git show-ref --verify --quiet refs/remotes/origin/main; then
  default_branch="main"
elif git show-ref --verify --quiet refs/remotes/origin/master; then
  default_branch="master"
else
  echo "worktree-create: could not resolve default branch on origin" >&2
  exit 1
fi

command wt fork --clean --base "origin/$default_branch" "$name" >&2
command wt path "$name"

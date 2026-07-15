#!/usr/bin/env bash
set -euo pipefail

# Claude Code WorktreeRemove hook — removes a wt-managed worktree.
# Stdin: JSON { worktree_path, cwd, ... }

input=$(cat)
worktree_path=$(echo "$input" | jq -r '.worktree_path')
cwd=$(echo "$input" | jq -r '.cwd')
name=$(basename "$worktree_path")

if [[ -z "$cwd" ]]; then
  echo "worktree-remove: missing cwd in hook input" >&2
  exit 1
fi

cd "$cwd"
command wt rm -f --preserve-branch "$name" >&2

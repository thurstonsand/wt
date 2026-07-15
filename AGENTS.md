# AGENTS.md

`wt` is a CLI wrapping git worktrees, but offers an actually friendly interface. It is primarily built for humans working with coding agents to quickly parallelize their work in a codebase without conflicting. This has become important in the age of Claude Code, where each agent can independently work at the same time, but run into trouble when they each need to touch the same files.

## Motivation

As I got better at using Claude Code, I found myself wanting to work on multiple issues/features simultaneously. I'd heard that git worktrees were the way to do that, but when I gave them a shot, I found the UX to be very unfriendly. I even tried to get Claude to drive the worktree commands, and it stumbled on them too. So I want to build a tool that improves the ergonomics of git worktrees while leaning on the existing features in git that enable it. Basically, I want to author What Git Worktree Should Have Been.

## Context

see @CONTEXT.md for domain definitions for this project.

## Tenets

- Minimize time-to-parallel: the entire purpose of this tool is to lower the effort to start/land a second stream
- The obvious command wins: prefer the command you'd guess should do the thing
- The parallel streams should fork WYSIWYG: carry dirty-state with you by default
- Ensure agent operability: the agent also has access to the tool, so `wt` should be deterministic and machine-readable
- Wrap git, don't reimplement: I want to hide the ugliness of git's UX, not reinvent the wheel
- Ruthless refactoring encouraged: break backwards compat freely
- Prefactor: every new feature or change should be easy; if it's not easy, what refactor can we do before the change to make it easy?
- Lie to hide complexity: it's ok to lie to the user about internal flow if it simplifies UX
- Never lie on destructive actions: irreversible changes should always be clearly indicated
- Get loud before violating a rule: flag clearly and get approval before breaking any of these tenets

## Features

- `wt` commands: `fork`, `checkout/co`, `merge`, `rebase`, `rebranch`, `rm`, `list/ls`, `path`, `prune`, `config`, `shell`
- dirty-state transfer on fork
- merge strategies: rebase, squash, or staged
- parent/child relationships between branches
- config file for behavioral options
- Claude Code integration: SKILL.md, hooks
- shell integration: completion, `wt cd`, auto-cd
- global prune of orphaned dirs/branches

## Developer Notes

see @DEV.md for coding style.

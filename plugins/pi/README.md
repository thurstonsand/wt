# pi-wt

Session mobility for [pi](https://github.com/earendil-works/pi-mono), with built-in support for additionally managing [`wt`](https://github.com/thurstonsand/wt) worktrees.

## Install

```bash
pi install npm:@thurstonsand/pi-wt
```

Use the `/mv` command out of the box, and install [wt](../../README.md) to gain access to the additional `/wt` command.

## Usage

Move the live session to any existing directory:

```text
/mv <dir>
```

The session continues immediately in the new folder with any env vars pre-loaded.

Move the session as part of a `wt` worktree operation:

```text
/wt fork [name] [flags]
/wt checkout <branch> [flags]
/wt co <branch> [flags]
/wt rm [-f] [name]
/wt merge [name] [flags]
/wt rebranch <branch> [flags]
```

## Skill

The package ships the `wt` skill teaching the agent the worktree workflows and when to hand an operation back to you as a `/wt` or `/mv` command. Disable it from `pi config` if you only want `/mv`.

## Development

```bash
npm run check
pi -e ./extensions/wt.ts
```

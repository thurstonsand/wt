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

The session continues immediately in the new folder.

Move the session as part of a `wt` worktree operation:

```text
/wt fork [name] [flags]
/wt checkout <branch> [flags]
/wt co <branch> [flags]
/wt rm [-f] [name]
/wt merge [name] [flags]
/wt rebranch <branch> [flags]
```

## Limitation

Moving a session does not refresh pi's process environment, meaning env vars don't update based on the new directory (e.g. by direnv or mise). If you need the new env, close the session after moving, `cd` to the new directory, and run `pi --session <uuid>` to pick them up.

## Development

```bash
npm run check
pi -e ./extensions/wt.ts
```

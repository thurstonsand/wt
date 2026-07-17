# pi-wt

A [pi](https://github.com/earendil-works/pi-mono) extension for moving the live session with [`wt`](https://github.com/thurstonsand/wt) worktrees.

## Install

Install `wt` first, then install the extension:

```bash
pi install npm:@thurstonsand/pi-wt
```

Both commands must inherit a `PATH` containing the `wt` binary.

## Usage

```text
/wt fork [name] [flags]
/wt checkout <branch> [flags]
/wt co <branch> [flags]
/wt rm [-f] [name]
/wt merge [name] [flags]
/wt rebranch <branch> [flags]
```

The extension delegates git and filesystem work to `wt`. It relocates the current session file without changing its ID, then asks pi to switch to that file in-process. `rm` and `merge` move a session home before deleting its worktree. `rebranch` also renames the worktree directory to match the new branch.

Other `wt` commands remain terminal commands. Argument completion is provided by `wt __complete`, so branch names and flags match the CLI.

## Limitation

Moving a session does not refresh pi's process environment. Restart pi after moving to a branch whose `.envrc`, secrets, or other environment setup differs.

## Development

```bash
npm run check
pi -e ./extensions/wt.ts
```

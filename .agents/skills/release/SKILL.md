---
name: release
description: Cut a release for wt — review the changelog, run the release task, and verify every channel. Use when the user wants to release, cut or publish a version, or tag a release.
---

# Cut a Release

A release is an annotated **git tag** `vX.Y.Z` whose message is that version's changelog section, verbatim. `mise run release X.Y.Z` creates it; pushing it publishes signed binaries to a GitHub Release, updates the Homebrew tap, and publishes the matching `@thurstonsand/pi-wt` version to npm, which ships `CHANGELOG.md` so Pi shows the new entries after an update.

`CHANGELOG.md` follows one standard, checked by `mise run lint:changelog`: `## [Unreleased]` on top, then `## [X.Y.Z] - YYYY-MM-DD` sections, newest first. Changes add their entries under `## [Unreleased]` as they land.

## 1. Determine the version

```bash
git describe --tags --abbrev=0        # last release tag
git log <last-tag>..HEAD --oneline
```

Release only when something user-facing shipped. Pick the bump by the versioning rule in `docs/release.md`. State the resulting `X.Y.Z` before continuing.

## 2. Review `## [Unreleased]`

The section records **user-facing** changes only — what someone running the tool would notice. Exclude internal mechanics: refactors, test/build/lint/CI chores, dependency bumps, anything invisible from outside. A `chore`/`refactor`/`test` commit is out; a `feat`/`fix` is usually in, reworded for the user.

Shape, under `## [Unreleased]`:

```markdown
<one-line headline — only for a marquee feature; omit the line entirely otherwise>

### Added

### Changed

### Fixed

### Removed
```

Rules:

- Omit the headline unless something major lands.
- Omit any section with no entries.
- Describe the new behavior, not the implementation. Write for the reader who runs the tool, not the one who edits it.
- One bullet per user-visible change; fold related commits together.

Completion criterion: every commit from step 1 is either represented in a bullet or consciously excluded as internal — none left unexamined.

If the section needed edits, commit them as `docs(changelog): …`, push to `main`, and wait for its CI to pass: the release task refuses anything else.

## 3. Run the release task

```bash
mise run release X.Y.Z
```

It refuses unless `main` is checked out, clean, equal to `origin/main` and green in CI, `## [Unreleased]` has entries, and `X.Y.Z` is newer than the latest release; then it runs `mise run release-check`. It then moves the Unreleased entries under `## [X.Y.Z] - <today, UTC>`, writes `X.Y.Z` into `plugins/pi/package.json` (and its lock) and `plugins/wt/.claude-plugin/plugin.json`, commits `chore(release): X.Y.Z`, creates the annotated tag from that section, and pushes `main` and the tag together. The `wt` binary takes its version from the tag at build time.

Confirm the tag message kept its headings:

```bash
git cat-file -p vX.Y.Z | head -20
```

## 4. Watch publishing

Watch the tag-triggered release workflow through completion. Its first job fails fast unless the tag, both manifests and the changelog's latest section agree:

```bash
gh run list --workflow Release --branch vX.Y.Z --limit 5
gh run watch <run-id> --exit-status
```

GoReleaser pushes the generated cask to `thurstonsand/homebrew-tap`, which triggers a separate `Casks` workflow. The release workflow does not wait for that downstream audit. Find the run for the exact generated tap commit and watch it too:

```bash
tap_sha=$(gh api "repos/thurstonsand/homebrew-tap/commits?path=Casks/wt.rb&per_page=1" --jq '.[0].sha')
gh run list --repo thurstonsand/homebrew-tap --workflow Casks --commit "$tap_sha" --limit 1
gh run watch <tap-run-id> --repo thurstonsand/homebrew-tap --exit-status
```

Do not report success until the binary and npm jobs in the release workflow pass **and** the downstream Homebrew cask workflow passes.

## 5. Verify every channel

Prove the Go module tag, GitHub artifacts, Homebrew cask, and npm package all carry the release version:

```bash
go list -m github.com/thurstonsand/wt@latest

gh release view vX.Y.Z
brew update
brew upgrade thurstonsand/tap/wt || brew install thurstonsand/tap/wt
brew info --cask thurstonsand/tap/wt
"$(brew --prefix)/bin/wt" --version

npm view @thurstonsand/pi-wt@X.Y.Z version
```

Every version check must report `vX.Y.Z` for `wt` or `X.Y.Z` for npm.

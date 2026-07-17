---
name: release
description: Cut a release for wt — bump the version, update the changelog, tag, push, and verify. Use when the user wants to release, cut or publish a version, or tag a release.
---

# Cut a Release

A release starts from an annotated **git tag** whose message is that version's changelog entry, verbatim. Pushing the tag publishes signed binaries to a GitHub Release, updates the Homebrew tap, and publishes the matching `@thurstonsand/pi-wt` version to npm.

## 1. Determine the version

```bash
git describe --tags --abbrev=0        # last release tag
git log <last-tag>..HEAD --oneline
```

Pick the bump under SemVer: a breaking change → major, any user-facing feature → minor, only fixes → patch. State the resulting `vX.Y.Z` before continuing.

## 2. Draft the changelog entry

The entry records **user-facing** changes only — what someone running the tool would notice. Exclude internal mechanics: refactors, test/build/lint/CI chores, dependency bumps, anything invisible from outside. A `chore`/`refactor`/`test` commit is out; a `feat`/`fix` is usually in, reworded for the user.

Shape, newest section at the top of the file:

```markdown
## vX.Y.Z

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

## 3. Write CHANGELOG.md

Prepend the section below the preamble. Lint after:

```bash
mise run lint
```

## 4. Commit the changelog

A dedicated commit, nothing else in it:

```bash
git commit CHANGELOG.md -m "docs(changelog): vX.Y.Z"
```

## 5. Tag the release

The tag message is the changelog section, verbatim, produced by `scripts/extract-release-notes.sh`.

```bash
git tag -a vX.Y.Z --cleanup=whitespace -F <(scripts/extract-release-notes.sh vX.Y.Z)
```

`--cleanup=whitespace` is mandatory. Git's default cleanup treats the `##`/`###` markdown headings as comment lines and strips them, silently gutting the message. Confirm the headings survived:

```bash
git cat-file -p vX.Y.Z | head -20
```

## 6. Push

```bash
git push && git push origin vX.Y.Z
```

## 7. Watch publishing

Watch the tag-triggered release workflow through completion:

```bash
gh run list --workflow Release --branch vX.Y.Z --limit 5
gh run watch <run-id> --exit-status
```

Do not report success until the binary, Homebrew, and npm jobs all pass.

## 8. Verify every channel

Prove the Go module tag, GitHub artifacts, Homebrew cask, and npm package all carry the release version:

```bash
go list -m github.com/thurstonsand/wt@latest
go install github.com/thurstonsand/wt@latest
"$(go env GOPATH)/bin/wt" --version

gh release view vX.Y.Z
brew update
brew upgrade thurstonsand/tap/wt || brew install thurstonsand/tap/wt
wt --version

npm view @thurstonsand/pi-wt@X.Y.Z version
```

Every version check must report `vX.Y.Z` for `wt` or `X.Y.Z` for npm.

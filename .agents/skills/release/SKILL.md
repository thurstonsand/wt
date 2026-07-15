---
name: release
description: Cut a release for wt — bump the version, update the changelog, tag, push, and verify. Use when the user wants to release, cut or publish a version, or tag a release.
---

# Cut a Release

A release is a **git tag** whose message is that version's changelog entry, verbatim. No platform release object (GitLab/GitHub) is created — the tag is the release.

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

## 7. Verify the tag is installable

Prove the tag is fetchable and carries the right version. For a Go module:

```bash
go list -m <module-path>@latest        # must resolve to vX.Y.Z
go install <module-path>@latest
```

Run the freshly installed binary's version flag and confirm it prints `vX.Y.Z`. `go install` writes to `GOBIN`; if an older copy earlier on `PATH` shadows it, inspect the `GOBIN` binary directly rather than trusting `which`.

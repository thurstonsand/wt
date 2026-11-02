#!/usr/bin/env bash
# Checks CHANGELOG.md against the release standard: `## [Unreleased]` first, then
# `## [x.y.z] - YYYY-MM-DD` sections, newest first, whose top version is the one every published
# manifest carries. Pi reads these headings to show pi-wt's changelog after an update, and ignores
# any it cannot parse. With a version argument (vX.Y.Z or X.Y.Z), the top release must also be it.
set -euo pipefail

if [[ $# -gt 1 ]]; then
  echo "usage: $0 [vX.Y.Z]" >&2
  exit 2
fi
expected="${1:+${1#v}}"

changelog=CHANGELOG.md
status=0
fail() {
  echo "wt: $*" >&2
  status=1
}

released=()
line_number=0
heading_count=0
while IFS= read -r line; do
  line_number=$((line_number + 1))
  [[ "$line" == "## "* ]] || continue
  heading_count=$((heading_count + 1))
  if [[ "$line" == "## [Unreleased]" ]]; then
    ((heading_count == 1)) || fail "$changelog:$line_number: [Unreleased] must be the first section"
  elif [[ "$line" =~ ^##\ \[([0-9]+\.[0-9]+\.[0-9]+)\]\ -\ [0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
    released+=("${BASH_REMATCH[1]}")
  else
    fail "$changelog:$line_number: \"$line\" is neither \"## [Unreleased]\" nor \"## [x.y.z] - YYYY-MM-DD\""
  fi
done <"$changelog"

grep -qx '## \[Unreleased\]' "$changelog" || fail "$changelog has no ## [Unreleased] section"

for ((i = 1; i < ${#released[@]}; i++)); do
  newer="${released[i - 1]}"
  older="${released[i]}"
  if [[ "$newer" == "$older" || "$(printf '%s\n%s\n' "$newer" "$older" | sort -V | tail -1)" != "$newer" ]]; then
    fail "$changelog: $newer is listed above $older; releases must strictly descend"
  fi
done

top="${released[0]:-}"
[[ -n "$top" ]] || fail "$changelog has no released version"

manifest_version() {
  sed -n 's/^  "version": "\([^"]*\)",$/\1/p' "$1"
}
for manifest in plugins/pi/package.json plugins/wt/.claude-plugin/plugin.json; do
  version="$(manifest_version "$manifest")"
  [[ "$version" == "$top" ]] || fail "$manifest says ${version:-no version}, but $changelog's latest release is $top"
done

if [[ -n "$expected" && "$expected" != "$top" ]]; then
  fail "releasing $expected, but $changelog's latest release is $top"
fi

exit "$status"

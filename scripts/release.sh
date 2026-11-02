#!/usr/bin/env bash
# Releases X.Y.Z from a reviewed `## [Unreleased]`: dates it, writes the version into every
# published manifest, commits, tags, and pushes main and the tag together. The tag push starts the
# release workflow. Run it as `mise run release X.Y.Z`.
set -euo pipefail

die() {
  echo "wt release: $*" >&2
  exit 1
}

[[ $# -eq 1 ]] || die "usage: $0 X.Y.Z"
version="$1"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "\"$version\" is not X.Y.Z"
tag="v$version"
changelog=CHANGELOG.md

branch="$(git symbolic-ref --quiet --short HEAD || true)"
[[ "$branch" == main ]] || die "on ${branch:-a detached HEAD}, not main"
[[ -z "$(git status --porcelain)" ]] || die "the working tree has uncommitted changes"
git fetch --quiet origin main
head="$(git rev-parse HEAD)"
[[ "$head" == "$(git rev-parse origin/main)" ]] || die "main (${head:0:7}) is not origin/main; push or pull first"

ci="$(gh run list --commit "$head" --workflow checks.yml --json status,conclusion \
  --jq '.[0] | if . == null then "not run" elif .status != "completed" then .status else .conclusion end')"
[[ "$ci" == success ]] || die "CI on ${head:0:7} is $ci, not green"

scripts/check-changelog.sh
current="$(sed -n 's/^## \[\([0-9][0-9.]*\)\] - .*$/\1/p' "$changelog" | head -1)"
if [[ "$version" == "$current" || "$(printf '%s\n%s\n' "$current" "$version" | sort -V | tail -1)" != "$version" ]]; then
  die "$version is not newer than $current"
fi
unreleased="$(awk '$0 == "## [Unreleased]" { inside = 1; next } inside && /^## / { exit } inside && NF' "$changelog")"
[[ -n "$unreleased" ]] || die "## [Unreleased] is empty; write the release's entries first"

mise run release-check

today="$(date -u +%Y-%m-%d)"
awk -v heading="## [$version] - $today" '{ print } $0 == "## [Unreleased]" { print ""; print heading }' \
  "$changelog" >"$changelog.tmp"
mv "$changelog.tmp" "$changelog"

# The wt binary needs no edit: goreleaser stamps it with the tag's version at build time.
(cd plugins/pi && npm version "$version" --no-git-tag-version >/dev/null)
plugin=plugins/wt/.claude-plugin/plugin.json
sed "s/^  \"version\": \"[^\"]*\",$/  \"version\": \"$version\",/" "$plugin" >"$plugin.tmp"
mv "$plugin.tmp" "$plugin"

scripts/check-changelog.sh "$version"

git add "$changelog" plugins/pi/package.json plugins/pi/package-lock.json "$plugin"
git commit --quiet -m "chore(release): $version"
# --cleanup=whitespace keeps the section's `##` and `###` headings, which git's default cleanup
# strips as comments.
git tag -a "$tag" --cleanup=whitespace -F <(scripts/extract-release-notes.sh "$version")
git push --atomic origin main "$tag"

echo "Released $tag; the Release workflow publishes it."

#!/usr/bin/env bash
set -euo pipefail

tag="${1:?tag is required}"

previous_tag="$(git describe --tags --abbrev=0 "${tag}^" 2>/dev/null || true)"

printf '# Changelog for %s\n\n' "$tag"

if [[ -n "$previous_tag" ]]; then
  printf 'Changes since `%s`.\n\n' "$previous_tag"
  range="$previous_tag..$tag"
else
  printf 'Initial release.\n\n'
  range="$tag"
fi

printf '## Commits\n\n'
git log --no-merges --pretty=format:'- %s (%h)' "$range"
printf '\n\n## Assets\n\n'
printf 'Release archives are provided for macOS, Linux, and Windows. Verify downloads with `checksums.txt`.\n'

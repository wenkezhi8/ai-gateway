#!/usr/bin/env bash
set -euo pipefail

fetch_tags=0

if [[ "${1:-}" == "--fetch" ]]; then
  fetch_tags=1
  shift
fi

if [[ "${1:-}" != "" ]]; then
  echo "Usage: $0 [--fetch]" >&2
  exit 2
fi

if [[ "$fetch_tags" -eq 1 ]]; then
  git fetch --tags
fi

latest_tag="$(git describe --tags --abbrev=0 2>/dev/null || true)"
if [[ -z "$latest_tag" ]]; then
  echo "No git tags found. Cannot sync version." >&2
  exit 1
fi

version="${latest_tag#v}"
tag_date="$(git log -1 --format=%cs "$latest_tag" 2>/dev/null || true)"
if [[ -z "$tag_date" ]]; then
  tag_date="$(date +%F)"
fi

echo "$version" > VERSION

if rg -q "\\*\\*v[0-9]+\\.[0-9]+\\.[0-9]+\\*\\* \\([0-9]{4}-[0-9]{2}-[0-9]{2}\\)" AGENTS.md; then
  perl -0pi -e "s/\\*\\*v[0-9]+\\.[0-9]+\\.[0-9]+\\*\\* \\([0-9]{4}-[0-9]{2}-[0-9]{2}\\)/\\*\\*v$version\\*\\* ($tag_date)/" AGENTS.md
else
  echo "AGENTS.md current version pattern not found." >&2
  exit 1
fi

if rg -q "^## \\[$version\\] - " CHANGELOG.md; then
  perl -0pi -e "s/## \\[$version\\] - [0-9]{4}-[0-9]{2}-[0-9]{2}/## [$version] - $tag_date/" CHANGELOG.md
else
  echo "CHANGELOG.md missing release section for $version. Add it first." >&2
  exit 1
fi

if rg -q "^\\| $version \\|" CHANGELOG.md; then
  perl -0pi -e "s/^\\| $version \\| [0-9]{4}-[0-9]{2}-[0-9]{2} \\|/| $version | $tag_date |/m" CHANGELOG.md
else
  tmp_file="$(mktemp)"
  awk -v version="$version" -v date="$tag_date" '
    BEGIN{inserted=0}
    {print}
    !inserted && $0 ~ /^\\|---------\\|/ {
      print "| " version " | " date " | Patch release synced from git tag |"
      inserted=1
    }
    END{if(!inserted) exit 1}
  ' CHANGELOG.md > "$tmp_file"
  mv "$tmp_file" CHANGELOG.md
fi

echo "Synced version to $version (tag $latest_tag, date $tag_date)."

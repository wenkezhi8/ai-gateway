#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib/git-branch.sh"

usage() {
  cat <<'USAGE'
Usage: ./scripts/start-feature-branch.sh <task-slug>

Example:
  ./scripts/start-feature-branch.sh release-smoke-hardening
Branch Prefix:
  codex/feature/<agent-id>/<task-slug>
USAGE
}

if [ $# -ne 1 ]; then
  usage
  exit 1
fi

slug="$1"
slug="${slug// /-}"
slug="${slug,,}"
if [[ ! "$slug" =~ ^[a-z0-9._-]+$ ]]; then
  echo "Invalid slug: $slug (allowed: a-z 0-9 . _ -)" >&2
  exit 1
fi

branch="$(git_make_feature_branch_name "auto" "$slug")"

git fetch origin main --quiet || true
git checkout -b "$branch" origin/main

echo "[start-feature-branch] created: $branch"

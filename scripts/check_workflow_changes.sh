#!/bin/bash

set -euo pipefail

BASE_REF="main"
ALLOWLIST_FILE=".github/workflow-change-allowlist.txt"

while [ $# -gt 0 ]; do
  case "$1" in
    --base-ref)
      BASE_REF="${2:-}"
      shift 2
      ;;
    --allowlist)
      ALLOWLIST_FILE="${2:-}"
      shift 2
      ;;
    -h|--help)
      echo "Usage: $0 [--base-ref <branch>] [--allowlist <file>]"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [ ! -f "$ALLOWLIST_FILE" ]; then
  echo "[workflow-guard] allowlist file not found: $ALLOWLIST_FILE" >&2
  exit 1
fi

ALLOW_PATTERNS=()
while IFS= read -r line; do
  trimmed="${line%$'\r'}"
  case "$trimmed" in
    ''|\#*)
      continue
      ;;
  esac
  ALLOW_PATTERNS+=("$trimmed")
done < "$ALLOWLIST_FILE"

CHANGED_WORKFLOWS=()
while IFS= read -r line; do
  [ -z "$line" ] && continue
  CHANGED_WORKFLOWS+=("$line")
done < <(git diff --name-only "origin/${BASE_REF}...HEAD" -- '.github/workflows/*.yml' '.github/workflows/*.yaml')

if [ ${#CHANGED_WORKFLOWS[@]} -eq 0 ]; then
  echo "[workflow-guard] no workflow changes"
  exit 0
fi

violations=()
for file in "${CHANGED_WORKFLOWS[@]}"; do
  allowed=false
  for pattern in "${ALLOW_PATTERNS[@]}"; do
    if [[ "$file" == $pattern ]]; then
      allowed=true
      break
    fi
  done
  if [ "$allowed" = false ]; then
    violations+=("$file")
  fi
done

if [ ${#violations[@]} -gt 0 ]; then
  echo "[workflow-guard] blocked: disallowed workflow changes detected"
  echo "allowed patterns:"
  for pattern in "${ALLOW_PATTERNS[@]}"; do
    echo "  - $pattern"
  done
  echo "violations:"
  for file in "${violations[@]}"; do
    echo "  - $file"
  done
  exit 1
fi

echo "[workflow-guard] pass"
echo "checked files: ${#CHANGED_WORKFLOWS[@]}"

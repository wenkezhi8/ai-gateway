#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

if bash ./scripts/dev-restart.sh "$@"; then
  if bash ./scripts/release-smoke.sh --base-url "http://localhost:8566"; then
    echo "[release-smoke-local] PASS"
    exit 0
  fi
fi

echo "[release-smoke-local] FAIL: dumping latest gateway logs" >&2
tail -n 200 /tmp/ai-gateway.log >&2 || true
exit 1

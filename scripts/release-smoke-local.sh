#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SKIP_RESTART=false
SMOKE_ARGS=()

usage() {
  cat <<'USAGE'
Usage: ./scripts/release-smoke-local.sh [options] [-- <release-smoke args>]

Options:
  --skip-restart  Skip dev-restart and run release-smoke directly.
  -h, --help      Show this help message.
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --skip-restart)
      SKIP_RESTART=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      SMOKE_ARGS+=("$@")
      break
      ;;
    *)
      SMOKE_ARGS+=("$1")
      shift
      ;;
  esac
done

cd "$PROJECT_ROOT"

if [ "$SKIP_RESTART" = false ]; then
  if ! bash ./scripts/dev-restart.sh; then
    echo "[release-smoke-local] FAIL: restart failed, dumping latest gateway logs" >&2
    tail -n 200 /tmp/ai-gateway.log >&2 || true
    exit 1
  fi
else
  echo "[release-smoke-local] skip restart"
fi

if bash ./scripts/release-smoke.sh --base-url "http://localhost:8566" "${SMOKE_ARGS[@]}"; then
    echo "[release-smoke-local] PASS"
    exit 0
fi

echo "[release-smoke-local] FAIL: dumping latest gateway logs" >&2
tail -n 200 /tmp/ai-gateway.log >&2 || true
exit 1

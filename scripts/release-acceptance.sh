#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

DRY_RUN=false
SKIP_BACKEND=false
SKIP_FRONTEND=false
SKIP_DELIVERY_STATUS=false
ALLOW_MISSING_PR=false
BASE_BRANCH="main"
PR_NUMBER=""

usage() {
  cat <<'EOF'
Usage: ./scripts/release-acceptance.sh [options]

Options:
  --dry-run               Print commands without executing.
  --skip-backend          Skip go test/go build checks.
  --skip-frontend         Skip frontend typecheck/build checks.
  --skip-delivery-status  Skip delivery-status.sh checks.
  --allow-missing-pr      Allow delivery status without PR detection.
  --base-branch <branch>  Base branch for delivery status (default: main).
  --pr <number>           Pull request number for merged-state validation.
EOF
}

run_cmd() {
  if [ "$DRY_RUN" = true ]; then
    echo "[dry-run] $*"
    return 0
  fi
  "$@"
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --skip-backend)
      SKIP_BACKEND=true
      shift
      ;;
    --skip-frontend)
      SKIP_FRONTEND=true
      shift
      ;;
    --skip-delivery-status)
      SKIP_DELIVERY_STATUS=true
      shift
      ;;
    --allow-missing-pr)
      ALLOW_MISSING_PR=true
      shift
      ;;
    --base-branch)
      BASE_BRANCH="${2:-}"
      shift 2
      ;;
    --pr)
      PR_NUMBER="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

echo "[release-acceptance] start"
echo "  project_root: $PROJECT_ROOT"
echo "  dry_run: $DRY_RUN"

cd "$PROJECT_ROOT"

echo "[release-acceptance] gate 1/4: git 提交证据三连"
run_cmd git rev-parse --short HEAD
run_cmd git show --name-only --pretty='' HEAD
run_cmd git status --short

if [ "$SKIP_BACKEND" = false ]; then
  echo "[release-acceptance] gate 2/4: backend verify"
  run_cmd go test ./...
  run_cmd go build ./cmd/gateway
else
  echo "[release-acceptance] gate 2/4: backend verify skipped"
fi

if [ "$SKIP_FRONTEND" = false ]; then
  echo "[release-acceptance] gate 3/4: frontend verify"
  if [ "$DRY_RUN" = true ]; then
    echo "[dry-run] (cd web && npm run typecheck && npm run build)"
  else
    (
      cd web
      npm run typecheck
      npm run build
    )
  fi
else
  echo "[release-acceptance] gate 3/4: frontend verify skipped"
fi

if [ "$SKIP_DELIVERY_STATUS" = false ]; then
  echo "[release-acceptance] gate 4/4: delivery status"
  DELIVERY_ARGS=(--base-branch "$BASE_BRANCH")
  if [ -n "$PR_NUMBER" ]; then
    DELIVERY_ARGS+=(--pr "$PR_NUMBER")
  fi
  if [ "$ALLOW_MISSING_PR" = true ]; then
    DELIVERY_ARGS+=(--allow-missing-pr)
  fi
  run_cmd bash "$SCRIPT_DIR/delivery-status.sh" "${DELIVERY_ARGS[@]}"
else
  echo "[release-acceptance] gate 4/4: delivery status skipped"
fi

echo "[release-acceptance] completed"

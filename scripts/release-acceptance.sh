#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
source "$SCRIPT_DIR/lib/cors.sh"

DRY_RUN=false
SKIP_BACKEND=false
SKIP_FRONTEND=false
SKIP_DELIVERY_STATUS=false
SKIP_RUNTIME_SMOKE=false
ALLOW_MISSING_PR=false
BASE_BRANCH="main"
PR_NUMBER=""
RUNTIME_SMOKE_URL="http://localhost:8566"
RUNTIME_SMOKE_METRICS_URL="http://127.0.0.1:9090/metrics"
RUNTIME_SMOKE_SWAGGER_JSON_URL=""
RUNTIME_SMOKE_ALLOWED_ORIGIN=""
RUNTIME_SMOKE_BLOCKED_ORIGIN=""
RUNTIME_SMOKE_CORS_FROM_ENV=false
RUNTIME_SMOKE_CORS_BLOCKED_ORIGIN="https://blocked.invalid"
SPAWN_GATEWAY=false
SPAWN_GATEWAY_SKIP_WEB_BUILD=false
LIMITED_NETWORK_REASON=""
LIMITED_NETWORK_MARKERS=(
  "Could not resolve host"
  "Network is unreachable"
  "Connection timed out"
  "Operation timed out"
  "Operation not permitted"
  "Failed to connect to"
  "No route to host"
)

usage() {
  cat <<'USAGE'
Usage: ./scripts/release-acceptance.sh [options]

Options:
  --dry-run                Print commands without executing.
  --skip-backend           Skip go test/go build checks.
  --skip-frontend          Skip frontend typecheck/build checks.
  --skip-delivery-status   Skip delivery-status.sh checks.
  --skip-runtime-smoke     Skip release-smoke.sh runtime verification.
  --runtime-smoke-url <u>  Base URL for runtime smoke (default: http://localhost:8566).
  --runtime-smoke-metrics-url <u>  Metrics URL for runtime smoke (default: http://127.0.0.1:9090/metrics).
  --runtime-smoke-swagger-json-url <u>  Swagger JSON URL for runtime smoke (default: <base-url>/swagger/doc.json).
  --runtime-smoke-allowed-origin <o>  Allowed Origin passed to release smoke CORS check.
  --runtime-smoke-blocked-origin <o>  Blocked Origin passed to release smoke CORS check.
  --runtime-smoke-cors-from-env   Auto map CORS_ALLOW_ORIGINS to runtime smoke allow/block origins.
  --runtime-smoke-cors-blocked-origin <o>  Override blocked origin used with --runtime-smoke-cors-from-env.
  --spawn-gateway          Start gateway in same session via dev-restart.sh before runtime smoke.
  --spawn-gateway-skip-web-build  Use with --spawn-gateway to skip web build during restart.
  --allow-missing-pr       Allow delivery status without PR detection.
  --base-branch <branch>   Base branch for delivery status (default: main).
  --pr <number>            Pull request number for merged-state validation.
USAGE
}

run_cmd() {
  if [ "$DRY_RUN" = true ]; then
    echo "[dry-run] $*"
    return 0
  fi
  "$@"
}

contains_limited_network_marker() {
  local detail="$1"
  local marker
  for marker in "${LIMITED_NETWORK_MARKERS[@]}"; do
    if [[ "$detail" == *"$marker"* ]]; then
      return 0
    fi
  done
  return 1
}

preflight_runtime_smoke_connectivity() {
  local url="$1"
  local label="$2"
  local stderr_file detail

  if ! command -v curl >/dev/null 2>&1; then
    LIMITED_NETWORK_REASON="$label: curl command is unavailable"
    return 2
  fi

  stderr_file="$(mktemp)"
  if curl -sS --output /dev/null --connect-timeout 2 --max-time 5 "$url" 2>"$stderr_file"; then
    rm -f "$stderr_file"
    return 0
  fi

  detail="$(cat "$stderr_file")"
  rm -f "$stderr_file"

  if contains_limited_network_marker "$detail"; then
    LIMITED_NETWORK_REASON="$label: $detail"
    return 2
  fi

  echo "[release-acceptance] FAIL: runtime smoke connectivity preflight failed target=$label url=$url detail=${detail:-unknown}" >&2
  return 1
}

validate_feature_branch_name() {
  local branch
  branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")"
  if [[ ! "$branch" =~ ^codex/feature/ ]]; then
    echo "[release-acceptance] FAIL: release-acceptance should run on feature branch (expected prefix: codex/feature/, current=$branch)" >&2
    return 1
  fi
  return 0
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
    --skip-runtime-smoke)
      SKIP_RUNTIME_SMOKE=true
      shift
      ;;
    --runtime-smoke-url)
      RUNTIME_SMOKE_URL="${2:-}"
      shift 2
      ;;
    --runtime-smoke-metrics-url)
      RUNTIME_SMOKE_METRICS_URL="${2:-}"
      shift 2
      ;;
    --runtime-smoke-swagger-json-url)
      RUNTIME_SMOKE_SWAGGER_JSON_URL="${2:-}"
      shift 2
      ;;
    --runtime-smoke-allowed-origin)
      RUNTIME_SMOKE_ALLOWED_ORIGIN="${2:-}"
      shift 2
      ;;
    --runtime-smoke-blocked-origin)
      RUNTIME_SMOKE_BLOCKED_ORIGIN="${2:-}"
      shift 2
      ;;
    --runtime-smoke-cors-from-env)
      RUNTIME_SMOKE_CORS_FROM_ENV=true
      shift
      ;;
    --runtime-smoke-cors-blocked-origin)
      RUNTIME_SMOKE_CORS_BLOCKED_ORIGIN="${2:-}"
      shift 2
      ;;
    --spawn-gateway)
      SPAWN_GATEWAY=true
      shift
      ;;
    --spawn-gateway-skip-web-build)
      SPAWN_GATEWAY_SKIP_WEB_BUILD=true
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

if [ "$DRY_RUN" = false ]; then
  validate_feature_branch_name
fi

echo "[release-acceptance] gate 1/5: git 提交证据三连"
run_cmd git rev-parse --short HEAD
run_cmd git show --name-only --pretty='' HEAD
run_cmd git status --short

if [ "$SKIP_BACKEND" = false ]; then
  echo "[release-acceptance] gate 2/5: backend verify"
  run_cmd go test ./...
  run_cmd go build ./cmd/gateway
else
  echo "[release-acceptance] gate 2/5: backend verify skipped"
fi

if [ "$SKIP_FRONTEND" = false ]; then
  echo "[release-acceptance] gate 3/5: frontend verify"
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
  echo "[release-acceptance] gate 3/5: frontend verify skipped"
fi

if [ "$SKIP_DELIVERY_STATUS" = false ]; then
  echo "[release-acceptance] gate 4/5: delivery status"
  DELIVERY_ARGS=(--base-branch "$BASE_BRANCH")
  if [ -n "$PR_NUMBER" ]; then
    DELIVERY_ARGS+=(--pr "$PR_NUMBER")
  fi
  if [ "$ALLOW_MISSING_PR" = true ]; then
    DELIVERY_ARGS+=(--allow-missing-pr)
  fi
  run_cmd bash "$SCRIPT_DIR/delivery-status.sh" "${DELIVERY_ARGS[@]}"
else
  echo "[release-acceptance] gate 4/5: delivery status skipped"
fi

if [ "$SKIP_RUNTIME_SMOKE" = false ]; then
  echo "[release-acceptance] gate 5/5: runtime smoke"

  cors_allow_origins="${CORS_ALLOW_ORIGINS:-}"
  cors_allow_origins_trimmed="$(cors_normalize_csv "$cors_allow_origins")"

  if [ "$RUNTIME_SMOKE_CORS_FROM_ENV" = true ] && [ -n "$cors_allow_origins_trimmed" ] && ! cors_allows_all_origins "$cors_allow_origins"; then
    mapped_origin="$(cors_first_specific_origin "$cors_allow_origins" || true)"
    if [ -n "$mapped_origin" ]; then
      RUNTIME_SMOKE_ALLOWED_ORIGIN="$mapped_origin"
      if [ -z "$RUNTIME_SMOKE_BLOCKED_ORIGIN" ]; then
        RUNTIME_SMOKE_BLOCKED_ORIGIN="$RUNTIME_SMOKE_CORS_BLOCKED_ORIGIN"
      fi
    fi
  fi

  if [ -n "$cors_allow_origins_trimmed" ] && ! cors_allows_all_origins "$cors_allow_origins"; then
    if [ -z "$RUNTIME_SMOKE_ALLOWED_ORIGIN" ] || [ -z "$RUNTIME_SMOKE_BLOCKED_ORIGIN" ]; then
      echo "[release-acceptance] FAIL: runtime smoke CORS whitelist is enabled (CORS_ALLOW_ORIGINS=$cors_allow_origins); --runtime-smoke-allowed-origin and --runtime-smoke-blocked-origin are required together" >&2
      exit 1
    fi
  fi

  if [ "$SPAWN_GATEWAY" = true ]; then
    if [ "$SPAWN_GATEWAY_SKIP_WEB_BUILD" = true ]; then
      run_cmd bash "$SCRIPT_DIR/dev-restart.sh" --skip-web-build
    else
      run_cmd bash "$SCRIPT_DIR/dev-restart.sh"
    fi
  fi

  RUNTIME_SMOKE_ARGS=(--base-url "$RUNTIME_SMOKE_URL" --metrics-url "$RUNTIME_SMOKE_METRICS_URL")
  if [ -n "$RUNTIME_SMOKE_SWAGGER_JSON_URL" ]; then
    RUNTIME_SMOKE_ARGS+=(--swagger-json-url "$RUNTIME_SMOKE_SWAGGER_JSON_URL")
  fi
  if [ -n "$RUNTIME_SMOKE_ALLOWED_ORIGIN" ]; then
    RUNTIME_SMOKE_ARGS+=(--allowed-origin "$RUNTIME_SMOKE_ALLOWED_ORIGIN")
  fi
  if [ -n "$RUNTIME_SMOKE_BLOCKED_ORIGIN" ]; then
    RUNTIME_SMOKE_ARGS+=(--blocked-origin "$RUNTIME_SMOKE_BLOCKED_ORIGIN")
  fi

  if [ "$DRY_RUN" = true ]; then
    run_cmd bash "$SCRIPT_DIR/release-smoke.sh" "${RUNTIME_SMOKE_ARGS[@]}"
  else
    if preflight_runtime_smoke_connectivity "$RUNTIME_SMOKE_URL/health" "runtime smoke base url"; then
      :
    else
      preflight_status=$?
      if [ "$preflight_status" -eq 2 ]; then
        echo "[release-acceptance] SKIP: runtime smoke connectivity preflight detected limited network environment: $LIMITED_NETWORK_REASON"
        echo "[release-acceptance] gate 5/5: runtime smoke skipped by connectivity preflight"
        echo "[release-acceptance] completed"
        exit 0
      fi
      exit "$preflight_status"
    fi

    if preflight_runtime_smoke_connectivity "$RUNTIME_SMOKE_METRICS_URL" "runtime smoke metrics url"; then
      :
    else
      preflight_status=$?
      if [ "$preflight_status" -eq 2 ]; then
        echo "[release-acceptance] SKIP: runtime smoke connectivity preflight detected limited network environment: $LIMITED_NETWORK_REASON"
        echo "[release-acceptance] gate 5/5: runtime smoke skipped by connectivity preflight"
        echo "[release-acceptance] completed"
        exit 0
      fi
      exit "$preflight_status"
    fi

    run_cmd bash "$SCRIPT_DIR/release-smoke.sh" "${RUNTIME_SMOKE_ARGS[@]}"
  fi
else
  echo "[release-acceptance] gate 5/5: runtime smoke skipped"
fi

echo "[release-acceptance] completed"

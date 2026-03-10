#!/bin/bash

set -euo pipefail

BASE_URL="http://localhost:8566"
METRICS_URL="http://127.0.0.1:9090/metrics"
LOG_PATH="/tmp/ai-gateway.log"
TRACE_PATH="/trace"
ALLOWED_ORIGIN=""
BLOCKED_ORIGIN=""
SMOKE_BODY_FILE="/tmp/ai-gateway-smoke-body.txt"
SMOKE_HEADER_FILE="/tmp/ai-gateway-smoke-headers.txt"
CURL_ARGS=(
  -s
  --connect-timeout 2
  --max-time 5
)

usage() {
  cat <<'USAGE'
Usage: ./scripts/release-smoke.sh [options]

Options:
  --base-url <url>      Base URL for runtime smoke checks (default: http://localhost:8566)
  --metrics-url <url>   Metrics URL for localhost/internal checks (default: http://127.0.0.1:9090/metrics)
  --log-path <path>     Gateway log path for cache backend inspection (default: /tmp/ai-gateway.log)
  --allowed-origin <o>  Verify CORS allows this origin (optional; requires whitelist env in gateway).
  --blocked-origin <o>  Verify CORS rejects this origin (optional; requires whitelist env in gateway).
USAGE
}

cleanup_smoke_files() {
  rm -f "$SMOKE_BODY_FILE" "$SMOKE_HEADER_FILE"
}

trap cleanup_smoke_files EXIT

reset_smoke_response_files() {
  : > "$SMOKE_BODY_FILE"
  : > "$SMOKE_HEADER_FILE"
}

while [ $# -gt 0 ]; do
  case "$1" in
    --base-url)
      BASE_URL="${2:-}"
      shift 2
      ;;
    --metrics-url)
      METRICS_URL="${2:-}"
      shift 2
      ;;
    --log-path)
      LOG_PATH="${2:-}"
      shift 2
      ;;
    --allowed-origin)
      ALLOWED_ORIGIN="${2:-}"
      shift 2
      ;;
    --blocked-origin)
      BLOCKED_ORIGIN="${2:-}"
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

curl_status() {
  local url="$1"
  shift || true
  reset_smoke_response_files
  curl "${CURL_ARGS[@]}" -o "$SMOKE_BODY_FILE" -w "%{http_code}" "$@" "$url" || true
}

curl_status_with_headers() {
  local url="$1"
  shift
  reset_smoke_response_files
  curl "${CURL_ARGS[@]}" -o "$SMOKE_BODY_FILE" -D "$SMOKE_HEADER_FILE" -w "%{http_code}" "$@" "$url" || true
}

expect_http_200() {
  local name="$1"
  local url="$2"
  local code
  code="$(curl_status "$url")"
  if [ "$code" = "000" ]; then
    echo "[release-smoke] FAIL: $name connection failed url=$url" >&2
    exit 1
  fi
  if [ "$code" != "200" ]; then
    echo "[release-smoke] FAIL: $name http_code=$code url=$url" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi
  echo "[release-smoke] PASS: $name"
}

assert_spa_shell() {
  local name="$1"
  local body_file="$2"
  local forbid_swagger_redirect="${3:-true}"

  if ! grep -qi '<!doctype html' "$body_file"; then
    echo "[release-smoke] FAIL: $name did not return SPA shell" >&2
    exit 1
  fi
  if [ "$forbid_swagger_redirect" = "true" ] && grep -q '/swagger/index.html' "$body_file"; then
    echo "[release-smoke] FAIL: $name should not redirect to swagger" >&2
    exit 1
  fi
}

expect_health_status() {
  local name="$1"
  local url="$2"
  expect_http_200 "$name" "$url"
  if ! grep -Eq 'healthy|ready' "$SMOKE_BODY_FILE"; then
    echo "[release-smoke] FAIL: $name body missing healthy/ready marker" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi
}

expect_not_http_200() {
  local name="$1"
  local url="$2"
  local code
  code="$(curl_status "$url")"
  if [ "$code" = "000" ]; then
    echo "[release-smoke] FAIL: $name connection failed url=$url" >&2
    exit 1
  fi
  if [ "$code" = "200" ]; then
    echo "[release-smoke] FAIL: $name unexpectedly returned 200 url=$url" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi
  echo "[release-smoke] PASS: $name closed with http_code=$code"
}

echo "[release-smoke] base_url=$BASE_URL"
echo "[release-smoke] metrics_url=$METRICS_URL"
echo "[release-smoke] allowed_origin=${ALLOWED_ORIGIN:-<skip>}"
echo "[release-smoke] blocked_origin=${BLOCKED_ORIGIN:-<skip>}"

echo "[release-smoke] check 1/11: health"
expect_health_status "health" "$BASE_URL/health"

echo "[release-smoke] check 2/11: ready"
expect_health_status "ready" "$BASE_URL/ready"

echo "[release-smoke] check 3/11: docs center"
expect_http_200 "docs center" "$BASE_URL/docs"
assert_spa_shell "docs center" "$SMOKE_BODY_FILE" true

echo "[release-smoke] check 4/11: docs center trailing slash"
expect_http_200 "docs center trailing slash" "$BASE_URL/docs/"
assert_spa_shell "docs center trailing slash" "$SMOKE_BODY_FILE" true

echo "[release-smoke] check 5/11: swagger root redirect"
swaggerCode="$(curl_status_with_headers "$BASE_URL/swagger")"
swaggerLocationLine="$(grep -i '^Location:' "$SMOKE_HEADER_FILE" | tr -d '\r' || true)"
if [ "$swaggerCode" != "302" ] || [ "$swaggerLocationLine" != "Location: /swagger/index.html" ]; then
  echo "[release-smoke] FAIL: swagger root redirect check failed code=$swaggerCode location=$swaggerLocationLine expected='Location: /swagger/index.html'" >&2
  exit 1
fi
echo "[release-smoke] PASS: swagger root redirect => $swaggerLocationLine"

swaggerSlashCode="$(curl_status_with_headers "$BASE_URL/swagger/")"
swaggerSlashLocationLine="$(grep -i '^Location:' "$SMOKE_HEADER_FILE" | tr -d '\r' || true)"
if [ "$swaggerSlashCode" != "302" ] || [ "$swaggerSlashLocationLine" != "Location: /swagger/index.html" ]; then
  echo "[release-smoke] FAIL: swagger trailing slash redirect check failed code=$swaggerSlashCode location=$swaggerSlashLocationLine expected='Location: /swagger/index.html'" >&2
  exit 1
fi
echo "[release-smoke] PASS: swagger trailing slash redirect => $swaggerSlashLocationLine"

echo "[release-smoke] check 6/11: trace page asset"
expect_http_200 "trace page" "$BASE_URL$TRACE_PATH"
TRACE_ASSET="$(grep -oE '/assets/index-[A-Za-z0-9_-]+\.js' "$SMOKE_BODY_FILE" | head -1 || true)"
if [ -z "$TRACE_ASSET" ]; then
  echo "[release-smoke] FAIL: trace page missing asset reference" >&2
  exit 1
fi
expect_http_200 "trace asset" "$BASE_URL$TRACE_ASSET"
if grep -qi '<!doctype html' "$SMOKE_BODY_FILE"; then
  echo "[release-smoke] FAIL: trace asset returned html" >&2
  exit 1
fi

echo "[release-smoke] check 7/11: debug endpoints closed"
expect_not_http_200 "debug pprof" "$BASE_URL/debug/pprof/"

echo "[release-smoke] check 8/11: metrics on gateway port closed"
expect_not_http_200 "metrics on gateway port" "$BASE_URL/metrics"

echo "[release-smoke] check 9/11: metrics localhost only"
expect_http_200 "Metrics (localhost only)" "$METRICS_URL"

echo "[release-smoke] check 10/11: cache backend hint"
if [ ! -f "$LOG_PATH" ]; then
  echo "[release-smoke] FAIL: missing log path $LOG_PATH" >&2
  exit 1
fi
if ! grep -Eq 'Cache backend is memory|Connected to Redis' "$LOG_PATH"; then
  echo "[release-smoke] FAIL: log missing cache backend hint in $LOG_PATH" >&2
  exit 1
fi
CACHE_BACKEND_LINE="$(grep -E 'Cache backend is memory|Connected to Redis' "$LOG_PATH" | tail -1)"
echo "[release-smoke] PASS: cache backend => $CACHE_BACKEND_LINE"

echo "[release-smoke] check 11/11: cors whitelist (optional)"
if [ -n "$ALLOWED_ORIGIN" ]; then
  code="$(curl_status_with_headers "$BASE_URL/health" -H "Origin: $ALLOWED_ORIGIN")"
  allowedHeader="$(grep -i '^Access-Control-Allow-Origin:' "$SMOKE_HEADER_FILE" | awk '{print $2}' | tr -d '\r' || true)"
  if [ "$code" != "200" ] || [ "$allowedHeader" != "$ALLOWED_ORIGIN" ]; then
    echo "[release-smoke] FAIL: cors allowed origin check failed code=$code allow_origin=$allowedHeader expected=$ALLOWED_ORIGIN" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: cors allowed origin => $ALLOWED_ORIGIN"
else
  echo "[release-smoke] SKIP: cors allowed origin"
fi

if [ -n "$BLOCKED_ORIGIN" ]; then
  code="$(curl_status "$BASE_URL/health" -H "Origin: $BLOCKED_ORIGIN")"
  if [ "$code" != "403" ]; then
    echo "[release-smoke] FAIL: cors blocked origin should be 403 code=$code origin=$BLOCKED_ORIGIN" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: cors blocked origin => $BLOCKED_ORIGIN"
else
  echo "[release-smoke] SKIP: cors blocked origin"
fi

echo "[release-smoke] completed"

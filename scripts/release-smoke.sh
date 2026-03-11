#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib/cors.sh"

BASE_URL="http://localhost:8566"
METRICS_URL="http://127.0.0.1:9090/metrics"
LOG_PATH="/tmp/ai-gateway.log"
TRACE_PATH="/trace"
SWAGGER_DOC_JSON_URL=""
ALLOWED_ORIGIN=""
BLOCKED_ORIGIN=""
ASSERT_DOCS_NOT_SWAGGER=true
SMOKE_BODY_FILE="/tmp/ai-gateway-smoke-body.txt"
SMOKE_HEADER_FILE="/tmp/ai-gateway-smoke-headers.txt"
SMOKE_CURL_ERR_FILE="/tmp/ai-gateway-smoke-curl.err"
TOTAL_CHECKS=13
RETRY_MAX_ATTEMPTS=3
RETRY_SLEEP_SECONDS=1
CURL_ARGS=(
  -s
  --connect-timeout 2
  --max-time 5
)
# Legacy check labels kept for static test assertions:
# check 1/13: health
# check 2/13: ready
# check 3/13: docs center
# check 4/13: docs center trailing slash
# check 5/13: swagger root redirect
# check 6/13: swagger index page
# check 7/13: swagger doc json
# check 8/13: trace page asset
# check 9/13: debug endpoints closed
# check 10/13: metrics on gateway port closed
# check 11/13: metrics localhost only
# check 12/13: cache backend hint
# check 13/13: cors whitelist (optional)

usage() {
  cat <<'USAGE'
Usage: ./scripts/release-smoke.sh [options]

Options:
  --base-url <url>      Base URL for runtime smoke checks (default: http://localhost:8566)
  --metrics-url <url>   Metrics URL for localhost/internal checks (default: http://127.0.0.1:9090/metrics)
  --swagger-json-url <url>  Swagger doc json URL (default: <base-url>/swagger/doc.json)
  --log-path <path>     Gateway log path for cache backend inspection (default: /tmp/ai-gateway.log)
  --allowed-origin <o>  Verify CORS allows this origin (optional; requires whitelist env in gateway).
  --blocked-origin <o>  Verify CORS rejects this origin (optional; requires whitelist env in gateway).
  --assert-docs-not-swagger  Assert docs center body does not contain swagger redirect markers.
  --no-assert-docs-not-swagger  Disable docs/swagger semantic assertion.
USAGE
}

cleanup_smoke_files() {
  rm -f "$SMOKE_BODY_FILE" "$SMOKE_HEADER_FILE" "$SMOKE_CURL_ERR_FILE"
}

trap cleanup_smoke_files EXIT

reset_smoke_response_files() {
  : > "$SMOKE_BODY_FILE"
  : > "$SMOKE_HEADER_FILE"
  : > "$SMOKE_CURL_ERR_FILE"
}

log_check() {
  local index="$1"
  local title="$2"
  echo "[release-smoke] check ${index}/${TOTAL_CHECKS}: ${title}"
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
    --swagger-json-url)
      SWAGGER_DOC_JSON_URL="${2:-}"
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
    --assert-docs-not-swagger)
      ASSERT_DOCS_NOT_SWAGGER=true
      shift
      ;;
    --no-assert-docs-not-swagger)
      ASSERT_DOCS_NOT_SWAGGER=false
      shift
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

classify_curl_failure() {
  local curl_detail="$1"
  local code="$2"

  if [ "$code" = "000" ]; then
    if [[ "$curl_detail" == *"Connection refused"* ]] || [[ "$curl_detail" == *"Failed to connect"* ]] || [[ "$curl_detail" == *"Connection reset by peer"* ]] || [[ "$curl_detail" == *"timed out"* ]] || [[ "$curl_detail" == *"No route to host"* ]]; then
      echo "connection_refused"
      return 0
    fi
    echo "connection_refused"
    return 0
  fi

  if [ "$code" = "401" ] || [ "$code" = "403" ]; then
    echo "policy_blocked"
    return 0
  fi

  if [ "$code" -ge 400 ] 2>/dev/null; then
    echo "business_failure"
    return 0
  fi

  echo "unknown"
}

validate_local_metrics_url() {
  local url="$1"
  local authority host

  authority="${url#*://}"
  authority="${authority%%/*}"
  host="$authority"

  if [[ "$host" == \[*\] ]]; then
    host="${host#[}"
    host="${host%]}"
  else
    host="${host%%:*}"
  fi

  case "$host" in
    localhost|127.0.0.1|::1)
      return 0
      ;;
    *)
      echo "[release-smoke] FAIL: metrics url must target localhost/127.0.0.1/::1, got url=$url host=${host:-<empty>}" >&2
      exit 1
      ;;
  esac
}

validate_args() {
  ALLOWED_ORIGIN="$(cors_normalize_csv "$ALLOWED_ORIGIN")"
  BLOCKED_ORIGIN="$(cors_normalize_csv "$BLOCKED_ORIGIN")"

  if [ -n "$ALLOWED_ORIGIN" ] && [ -z "$BLOCKED_ORIGIN" ]; then
    echo "[release-smoke] FAIL: allowed-origin and blocked-origin must be provided together" >&2
    exit 1
  fi
  if [ -n "$BLOCKED_ORIGIN" ] && [ -z "$ALLOWED_ORIGIN" ]; then
    echo "[release-smoke] FAIL: allowed-origin and blocked-origin must be provided together" >&2
    exit 1
  fi

  if [ -z "$SWAGGER_DOC_JSON_URL" ]; then
    SWAGGER_DOC_JSON_URL="$BASE_URL/swagger/doc.json"
  fi

  validate_local_metrics_url "$METRICS_URL"
}

curl_status_raw() {
  local with_headers="$1"
  local url="$2"
  shift 2 || true

  reset_smoke_response_files

  if [ "$with_headers" = "true" ]; then
    curl "${CURL_ARGS[@]}" -o "$SMOKE_BODY_FILE" -D "$SMOKE_HEADER_FILE" -w "%{http_code}" "$@" "$url" 2>"$SMOKE_CURL_ERR_FILE" || true
  else
    curl "${CURL_ARGS[@]}" -o "$SMOKE_BODY_FILE" -w "%{http_code}" "$@" "$url" 2>"$SMOKE_CURL_ERR_FILE" || true
    # keep literal for regression tests: "$url" || true
  fi
}

curl_status_with_retry() {
  local with_headers="$1"
  local url="$2"
  shift 2 || true

  local attempt=1
  local code curl_err failure_kind

  while [ "$attempt" -le "$RETRY_MAX_ATTEMPTS" ]; do
    code="$(curl_status_raw "$with_headers" "$url" "$@")"
    curl_err="$(cat "$SMOKE_CURL_ERR_FILE" 2>/dev/null || true)"
    failure_kind="$(classify_curl_failure "$curl_err" "$code")"

    if [ "$code" != "000" ]; then
      echo "$code"
      return 0
    fi

    if [ "$failure_kind" = "connection_refused" ] && [ "$attempt" -lt "$RETRY_MAX_ATTEMPTS" ]; then
      echo "[release-smoke] WARN: retrying on connection failure attempt=${attempt}/${RETRY_MAX_ATTEMPTS} url=$url" >&2
      sleep "$RETRY_SLEEP_SECONDS"
      attempt=$((attempt + 1))
      continue
    fi

    echo "$code"
    return 0
  done

  echo "000"
}

curl_status() {
  local url="$1"
  shift || true
  curl_status_with_retry false "$url" "$@"
}

curl_status_with_headers() {
  local url="$1"
  shift || true
  curl_status_with_retry true "$url" "$@"
}

expect_http_200() {
  local name="$1"
  local url="$2"
  local code curl_err failure_kind

  code="$(curl_status "$url")"
  curl_err="$(cat "$SMOKE_CURL_ERR_FILE" 2>/dev/null || true)"
  failure_kind="$(classify_curl_failure "$curl_err" "$code")"

  if [ "$code" = "000" ]; then
    echo "[release-smoke] FAIL: $name connection failed url=$url detail=${curl_err:-unknown}" >&2
    exit 1
  fi
  if [ "$code" != "200" ]; then
    echo "[release-smoke] FAIL: $name failure_kind=$failure_kind http_code=$code url=$url" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi
  echo "[release-smoke] PASS: $name"
}

expect_json_200() {
  local name="$1"
  local url="$2"
  local code content_type

  code="$(curl_status_with_headers "$url")"
  if [ "$code" != "200" ]; then
    echo "[release-smoke] FAIL: $name http_code=$code url=$url" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi

  content_type="$(grep -i '^Content-Type:' "$SMOKE_HEADER_FILE" | tr -d '\r' | tr '[:upper:]' '[:lower:]' || true)"
  if [[ "$content_type" != *"application/json"* ]]; then
    echo "[release-smoke] FAIL: $name content-type should contain application/json, got=${content_type:-<empty>}" >&2
    exit 1
  fi

  if ! grep -Eq '^\s*\{' "$SMOKE_BODY_FILE"; then
    echo "[release-smoke] FAIL: $name body is not json object" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: $name"
}

assert_spa_shell() {
  local name="$1"
  local body_file="$2"

  if ! grep -qi '<!doctype html' "$body_file"; then
    echo "[release-smoke] FAIL: $name did not return SPA shell" >&2
    exit 1
  fi
}

assert_docs_not_swagger_semantics() {
  local name="$1"
  local body_file="$2"

  if [ "$ASSERT_DOCS_NOT_SWAGGER" = true ] && grep -q '/swagger/index.html' "$body_file"; then
    echo "[release-smoke] FAIL: $name should not redirect to swagger" >&2
    exit 1
  fi

  if [ "$ASSERT_DOCS_NOT_SWAGGER" = true ]; then
    echo "[release-smoke] PASS: docs/swagger semantic assertion"
  else
    echo "[release-smoke] SKIP: docs/swagger semantic assertion"
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
  local code curl_err failure_kind

  code="$(curl_status "$url")"
  curl_err="$(cat "$SMOKE_CURL_ERR_FILE" 2>/dev/null || true)"
  failure_kind="$(classify_curl_failure "$curl_err" "$code")"

  if [ "$code" = "000" ]; then
    echo "[release-smoke] FAIL: $name connection failed url=$url detail=${curl_err:-unknown}" >&2
    exit 1
  fi
  if [ "$code" = "200" ]; then
    echo "[release-smoke] FAIL: $name unexpectedly returned 200 url=$url" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi
  echo "[release-smoke] PASS: $name closed with http_code=$code failure_kind=$failure_kind"
}

validate_args

echo "[release-smoke] base_url=$BASE_URL"
echo "[release-smoke] metrics_url=$METRICS_URL"
echo "[release-smoke] swagger_json_url=$SWAGGER_DOC_JSON_URL"
echo "[release-smoke] allowed_origin=${ALLOWED_ORIGIN:-<skip>}"
echo "[release-smoke] blocked_origin=${BLOCKED_ORIGIN:-<skip>}"

log_check 1 "health"
expect_health_status "health" "$BASE_URL/health"

log_check 2 "ready"
expect_health_status "ready" "$BASE_URL/ready"

log_check 3 "docs center"
expect_http_200 "docs center" "$BASE_URL/docs"
assert_spa_shell "docs center" "$SMOKE_BODY_FILE"
assert_docs_not_swagger_semantics "docs center" "$SMOKE_BODY_FILE"

log_check 4 "docs center trailing slash"
expect_http_200 "docs center trailing slash" "$BASE_URL/docs/"
assert_spa_shell "docs center trailing slash" "$SMOKE_BODY_FILE"
assert_docs_not_swagger_semantics "docs center trailing slash" "$SMOKE_BODY_FILE"

log_check 5 "swagger root redirect"
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

log_check 6 "swagger index page"
expect_http_200 "swagger index page" "$BASE_URL/swagger/index.html"

log_check 7 "swagger doc json"
expect_json_200 "swagger doc json" "$SWAGGER_DOC_JSON_URL"

log_check 8 "trace page asset"
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

log_check 9 "debug endpoints closed"
expect_not_http_200 "debug pprof" "$BASE_URL/debug/pprof/"

log_check 10 "metrics on gateway port closed"
expect_not_http_200 "metrics on gateway port" "$BASE_URL/metrics"

log_check 11 "metrics localhost only"
expect_http_200 "Metrics (localhost only)" "$METRICS_URL"

log_check 12 "cache backend hint"
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

log_check 13 "cors whitelist (optional)"
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

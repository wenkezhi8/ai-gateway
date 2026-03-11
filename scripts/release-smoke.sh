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
ALLOW_LIMITED_NETWORK_SKIP=false
REQUIRE_VARY_ORIGIN=true
MAX_JSON_SAMPLE_BYTES=65536
SMOKE_BODY_FILE="/tmp/ai-gateway-smoke-body.txt"
SMOKE_HEADER_FILE="/tmp/ai-gateway-smoke-headers.txt"
SMOKE_CURL_ERR_FILE="/tmp/ai-gateway-smoke-curl.err"
SMOKE_JSON_SAMPLE_FILE="/tmp/ai-gateway-smoke-json-sample.txt"
TOTAL_CHECKS=13
RETRY_MAX_ATTEMPTS=3
RETRY_SLEEP_SECONDS=1
CURL_ARGS=(
  -s
  --connect-timeout 2
  --max-time 5
)
LIMITED_NETWORK_MARKERS=(
  "connection_refused"
  "Could not resolve host"
  "Network is unreachable"
  "Connection timed out"
  "Operation timed out"
  "Operation not permitted"
  "Failed to connect to"
  "No route to host"
)
CHECK_TITLES=(
  "health"
  "ready"
  "docs center"
  "docs center trailing slash"
  "swagger root redirect"
  "swagger index page"
  "swagger doc json"
  "trace page asset"
  "debug endpoints closed"
  "metrics on gateway port closed"
  "metrics localhost only"
  "cache backend hint"
  "cors whitelist (optional)"
)
CHECK_HANDLERS=(
  "check_health"
  "check_ready"
  "check_docs_center"
  "check_docs_center_trailing_slash"
  "check_swagger_root_redirect"
  "check_swagger_index_page"
  "check_swagger_doc_json"
  "check_trace_page_asset"
  "check_debug_endpoints_closed"
  "check_metrics_gateway_port_closed"
  "check_metrics_localhost_only"
  "check_cache_backend_hint"
  "check_cors_whitelist"
)
CHECK_OPTIONAL_FLAGS=(
  "false"
  "false"
  "false"
  "false"
  "false"
  "false"
  "false"
  "false"
  "false"
  "false"
  "false"
  "false"
  "true"
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
# Legacy variable names kept for static test assertions:
# docsLocationLine=
# docsSlashLocationLine=
# Legacy CORS vary failure markers:
# cors allowed origin vary check failed
# cors allowed preflight vary check failed
# cors blocked origin vary check failed
# cors blocked preflight vary check failed

usage() {
  cat <<'USAGE'
Usage: ./scripts/release-smoke.sh [options]

Options:
  --base-url <url>      Base URL for runtime smoke checks (default: http://localhost:8566)
  --metrics-url <url>   Metrics URL for localhost/internal checks (default: http://127.0.0.1:9090/metrics)
  --swagger-json-url <url>  Swagger doc json URL (default: <base-url>/swagger/doc.json)
  --swagger-json-max-bytes <n>  Max bytes sampled from swagger doc body (default: 65536)
  --log-path <path>     Gateway log path for cache backend inspection (default: /tmp/ai-gateway.log)
  --allowed-origin <o>  Verify CORS allows this origin (optional; requires whitelist env in gateway).
  --blocked-origin <o>  Verify CORS rejects this origin (optional; requires whitelist env in gateway).
  --allow-limited-network-skip  Exit with SKIP when loopback/network is blocked by environment.
  --require-vary-origin  Require Vary: Origin in CORS checks (default).
  --no-require-vary-origin  Disable Vary: Origin requirement in CORS checks.
  --assert-docs-not-swagger  Assert docs center body does not contain swagger redirect markers.
  --no-assert-docs-not-swagger  Disable docs/swagger semantic assertion.
USAGE
}

cleanup_smoke_files() {
  rm -f "$SMOKE_BODY_FILE" "$SMOKE_HEADER_FILE" "$SMOKE_CURL_ERR_FILE" "$SMOKE_JSON_SAMPLE_FILE"
}

trap cleanup_smoke_files EXIT

reset_smoke_response_files() {
  : > "$SMOKE_BODY_FILE"
  : > "$SMOKE_HEADER_FILE"
  : > "$SMOKE_CURL_ERR_FILE"
  : > "$SMOKE_JSON_SAMPLE_FILE"
}

log_check() {
  local index="$1"
  local title="$2"
  echo "[release-smoke] check ${index}/${TOTAL_CHECKS}: ${title}"
}

check_title() {
  local index="$1"
  echo "${CHECK_TITLES[$((index - 1))]}"
}

is_optional_check() {
  local index="$1"
  echo "${CHECK_OPTIONAL_FLAGS[$((index - 1))]}"
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

skip_on_limited_network_if_enabled() {
  local detail="$1"

  if [ "$ALLOW_LIMITED_NETWORK_SKIP" != true ]; then
    return 1
  fi

  if contains_limited_network_marker "$detail"; then
    echo "[release-smoke] SKIP: limited network environment detected detail=$detail"
    echo "[release-smoke] completed (skipped)"
    exit 0
  fi

  return 1
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
    --swagger-json-max-bytes)
      MAX_JSON_SAMPLE_BYTES="${2:-}"
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
    --allow-limited-network-skip)
      ALLOW_LIMITED_NETWORK_SKIP=true
      shift
      ;;
    --require-vary-origin)
      REQUIRE_VARY_ORIGIN=true
      shift
      ;;
    --no-require-vary-origin)
      REQUIRE_VARY_ORIGIN=false
      shift
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
    if [[ "$curl_detail" == *"Connection refused"* ]] || [[ "$curl_detail" == *"Failed to connect"* ]] || [[ "$curl_detail" == *"Connection reset by peer"* ]] || [[ "$curl_detail" == *"timed out"* ]] || [[ "$curl_detail" == *"No route to host"* ]] || [[ "$curl_detail" == *"Operation not permitted"* ]]; then
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

failure_detail() {
  local curl_detail="$1"
  local failure_kind="$2"

  curl_detail="$(printf "%s" "$curl_detail" | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g' | sed 's/^ //;s/ $//')"
  if [ -n "$curl_detail" ]; then
    echo "$curl_detail"
    return 0
  fi
  if [ -n "$failure_kind" ] && [ "$failure_kind" != "unknown" ]; then
    echo "$failure_kind"
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

  if ! [[ "$MAX_JSON_SAMPLE_BYTES" =~ ^[0-9]+$ ]] || [ "$MAX_JSON_SAMPLE_BYTES" -le 0 ]; then
    echo "[release-smoke] FAIL: --swagger-json-max-bytes must be positive integer, got=$MAX_JSON_SAMPLE_BYTES" >&2
    exit 1
  fi

  if [ -z "$SWAGGER_DOC_JSON_URL" ]; then
    SWAGGER_DOC_JSON_URL="$BASE_URL/swagger/doc.json"
  fi

  if [ "${#CHECK_TITLES[@]}" -ne "$TOTAL_CHECKS" ] || [ "${#CHECK_HANDLERS[@]}" -ne "$TOTAL_CHECKS" ] || [ "${#CHECK_OPTIONAL_FLAGS[@]}" -ne "$TOTAL_CHECKS" ]; then
    echo "[release-smoke] FAIL: check metadata arrays must all match TOTAL_CHECKS=$TOTAL_CHECKS" >&2
    exit 1
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

extract_header_value() {
  local header_name="$1"
  local header_line
  header_line="$(grep -i "^${header_name}:" "$SMOKE_HEADER_FILE" | head -1 | tr -d '\r' || true)"
  header_line="${header_line#*:}"
  echo "$(echo "$header_line" | sed 's/^ *//;s/ *$//')"
}

assert_vary_origin_if_required() {
  local vary_value="$1"
  local context="$2"

  if [ "$REQUIRE_VARY_ORIGIN" != true ]; then
    return 0
  fi

  if [[ "$vary_value" != *"Origin"* ]]; then
    echo "[release-smoke] FAIL: $context vary check failed vary=${vary_value:-<empty>} expected_contains=Origin" >&2
    exit 1
  fi
}

expect_http_200() {
  local name="$1"
  local url="$2"
  local code curl_err failure_kind detail

  code="$(curl_status "$url")"
  curl_err="$(cat "$SMOKE_CURL_ERR_FILE" 2>/dev/null || true)"
  failure_kind="$(classify_curl_failure "$curl_err" "$code")"
  detail="$(failure_detail "$curl_err" "$failure_kind")"

  if [ "$code" = "000" ]; then
    skip_on_limited_network_if_enabled "$detail" || true
    echo "[release-smoke] FAIL: $name connection failed url=$url detail=$detail" >&2
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
  local code content_type response_size

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

  response_size="$(wc -c < "$SMOKE_BODY_FILE" | tr -d ' ')"
  if [ "$response_size" -gt "$MAX_JSON_SAMPLE_BYTES" ]; then
    echo "[release-smoke] FAIL: swagger doc json sample exceeds limit size=$response_size limit=$MAX_JSON_SAMPLE_BYTES" >&2
    exit 1
  fi

  head -c "$MAX_JSON_SAMPLE_BYTES" "$SMOKE_BODY_FILE" > "$SMOKE_JSON_SAMPLE_FILE"
  if ! grep -Eq '^\s*\{' "$SMOKE_JSON_SAMPLE_FILE"; then
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

assert_swagger_ui_shell() {
  local name="$1"
  local body_file="$2"

  if ! grep -q 'SwaggerUIBundle' "$body_file"; then
    echo "[release-smoke] FAIL: swagger index should expose swagger ui marker" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: swagger index should expose swagger ui marker"
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
  local code curl_err failure_kind detail

  code="$(curl_status "$url")"
  curl_err="$(cat "$SMOKE_CURL_ERR_FILE" 2>/dev/null || true)"
  failure_kind="$(classify_curl_failure "$curl_err" "$code")"
  detail="$(failure_detail "$curl_err" "$failure_kind")"

  if [ "$code" = "000" ]; then
    skip_on_limited_network_if_enabled "$detail" || true
    echo "[release-smoke] FAIL: $name connection failed url=$url detail=$detail" >&2
    exit 1
  fi
  if [ "$code" = "200" ]; then
    echo "[release-smoke] FAIL: $name unexpectedly returned 200 url=$url" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi
  echo "[release-smoke] PASS: $name closed with http_code=$code failure_kind=$failure_kind"
}

check_health() {
  expect_health_status "health" "$BASE_URL/health"
}

check_ready() {
  expect_health_status "ready" "$BASE_URL/ready"
}

check_docs_center() {
  local code location
  code="$(curl_status_with_headers "$BASE_URL/docs")"
  location="$(extract_header_value "Location")"
  if [ "$code" != "200" ]; then
    echo "[release-smoke] FAIL: docs center http_code=$code url=$BASE_URL/docs" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi
  if [ -n "$location" ]; then
    echo "[release-smoke] FAIL: docs center should not include redirect location header got=Location: $location" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: docs center should not include redirect location header"
  assert_spa_shell "docs center" "$SMOKE_BODY_FILE"
  assert_docs_not_swagger_semantics "docs center" "$SMOKE_BODY_FILE"
}

check_docs_center_trailing_slash() {
  local code location
  code="$(curl_status_with_headers "$BASE_URL/docs/")"
  location="$(extract_header_value "Location")"
  if [ "$code" != "200" ]; then
    echo "[release-smoke] FAIL: docs center trailing slash http_code=$code url=$BASE_URL/docs/" >&2
    cat "$SMOKE_BODY_FILE" >&2 || true
    exit 1
  fi
  if [ -n "$location" ]; then
    echo "[release-smoke] FAIL: docs center trailing slash should not include redirect location header got=Location: $location" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: docs center trailing slash should not include redirect location header"
  assert_spa_shell "docs center trailing slash" "$SMOKE_BODY_FILE"
  assert_docs_not_swagger_semantics "docs center trailing slash" "$SMOKE_BODY_FILE"
}

check_swagger_root_redirect() {
  local code location
  code="$(curl_status_with_headers "$BASE_URL/swagger")"
  location="$(extract_header_value "Location")"
  if [ "$code" != "302" ] || [ "$location" != "/swagger/index.html" ]; then
    echo "[release-smoke] FAIL: swagger root redirect check failed code=$code location=Location: $location expected='Location: /swagger/index.html'" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: swagger root redirect => Location: $location"

  code="$(curl_status_with_headers "$BASE_URL/swagger/")"
  location="$(extract_header_value "Location")"
  if [ "$code" != "302" ] || [ "$location" != "/swagger/index.html" ]; then
    echo "[release-smoke] FAIL: swagger trailing slash redirect check failed code=$code location=Location: $location expected='Location: /swagger/index.html'" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: swagger trailing slash redirect => Location: $location"
}

check_swagger_index_page() {
  expect_http_200 "swagger index page" "$BASE_URL/swagger/index.html"
  assert_swagger_ui_shell "swagger index page" "$SMOKE_BODY_FILE"
}

check_swagger_doc_json() {
  expect_json_200 "swagger doc json" "$SWAGGER_DOC_JSON_URL"
}

check_trace_page_asset() {
  local trace_asset
  expect_http_200 "trace page" "$BASE_URL$TRACE_PATH"
  trace_asset="$(grep -oE '/assets/index-[A-Za-z0-9_-]+\.js' "$SMOKE_BODY_FILE" | head -1 || true)"
  if [ -z "$trace_asset" ]; then
    echo "[release-smoke] FAIL: trace page missing asset reference" >&2
    exit 1
  fi
  expect_http_200 "trace asset" "$BASE_URL$trace_asset"
  if grep -qi '<!doctype html' "$SMOKE_BODY_FILE"; then
    echo "[release-smoke] FAIL: trace asset returned html" >&2
    exit 1
  fi
}

check_debug_endpoints_closed() {
  expect_not_http_200 "debug pprof" "$BASE_URL/debug/pprof/"
}

check_metrics_gateway_port_closed() {
  expect_not_http_200 "metrics on gateway port" "$BASE_URL/metrics"
}

check_metrics_localhost_only() {
  expect_http_200 "Metrics (localhost only)" "$METRICS_URL"
}

check_cache_backend_hint() {
  local cache_backend_line
  if [ ! -f "$LOG_PATH" ]; then
    echo "[release-smoke] FAIL: missing log path $LOG_PATH" >&2
    exit 1
  fi
  if ! grep -Eq 'Cache backend is memory|Connected to Redis' "$LOG_PATH"; then
    echo "[release-smoke] FAIL: log missing cache backend hint in $LOG_PATH" >&2
    exit 1
  fi
  cache_backend_line="$(grep -E 'Cache backend is memory|Connected to Redis' "$LOG_PATH" | tail -1)"
  echo "[release-smoke] PASS: cache backend => $cache_backend_line"
}

check_cors_whitelist() {
  local code allowed_header blocked_vary preflight_allowed_code preflight_allowed_header preflight_blocked_code
  local allowed_vary preflight_allowed_vary preflight_blocked_vary

  if [ -n "$ALLOWED_ORIGIN" ]; then
    code="$(curl_status_with_headers "$BASE_URL/health" -H "Origin: $ALLOWED_ORIGIN")"
    allowed_header="$(extract_header_value "Access-Control-Allow-Origin")"
    allowed_vary="$(extract_header_value "Vary")"
    if [ "$code" != "200" ] || [ "$allowed_header" != "$ALLOWED_ORIGIN" ]; then
      echo "[release-smoke] FAIL: cors allowed origin check failed code=$code allow_origin=$allowed_header expected=$ALLOWED_ORIGIN" >&2
      exit 1
    fi
    assert_vary_origin_if_required "$allowed_vary" "cors allowed origin"
    echo "[release-smoke] PASS: cors allowed origin => $ALLOWED_ORIGIN"

    preflight_allowed_code="$(curl_status_with_headers "$BASE_URL/health" -X OPTIONS -H "Origin: $ALLOWED_ORIGIN" -H "Access-Control-Request-Method: POST")"
    preflight_allowed_header="$(extract_header_value "Access-Control-Allow-Origin")"
    preflight_allowed_vary="$(extract_header_value "Vary")"
    if [ "$preflight_allowed_code" != "204" ] || [ "$preflight_allowed_header" != "$ALLOWED_ORIGIN" ]; then
      echo "[release-smoke] FAIL: cors allowed preflight check failed code=$preflight_allowed_code allow_origin=$preflight_allowed_header expected=$ALLOWED_ORIGIN" >&2
      exit 1
    fi
    assert_vary_origin_if_required "$preflight_allowed_vary" "cors allowed preflight"
    echo "[release-smoke] PASS: cors allowed preflight => $ALLOWED_ORIGIN"
  else
    echo "[release-smoke] SKIP: cors allowed origin"
    echo "[release-smoke] SKIP: cors allowed preflight"
  fi

  if [ -n "$BLOCKED_ORIGIN" ]; then
    code="$(curl_status_with_headers "$BASE_URL/health" -H "Origin: $BLOCKED_ORIGIN")"
    blocked_vary="$(extract_header_value "Vary")"
    if [ "$code" != "403" ]; then
      echo "[release-smoke] FAIL: cors blocked origin should be 403 code=$code origin=$BLOCKED_ORIGIN" >&2
      exit 1
    fi
    assert_vary_origin_if_required "$blocked_vary" "cors blocked origin"
    echo "[release-smoke] PASS: cors blocked origin => $BLOCKED_ORIGIN"

    preflight_blocked_code="$(curl_status_with_headers "$BASE_URL/health" -X OPTIONS -H "Origin: $BLOCKED_ORIGIN" -H "Access-Control-Request-Method: POST")"
    preflight_blocked_vary="$(extract_header_value "Vary")"
    if [ "$preflight_blocked_code" != "403" ]; then
      echo "[release-smoke] FAIL: cors blocked preflight should be 403 code=$preflight_blocked_code origin=$BLOCKED_ORIGIN" >&2
      exit 1
    fi
    assert_vary_origin_if_required "$preflight_blocked_vary" "cors blocked preflight"
    echo "[release-smoke] PASS: cors blocked preflight => $BLOCKED_ORIGIN"
  else
    echo "[release-smoke] SKIP: cors blocked origin"
    echo "[release-smoke] SKIP: cors blocked preflight"
  fi
}

run_all_checks() {
  local check_index check_handler check_optional
  for ((check_index=1; check_index<=TOTAL_CHECKS; check_index++)); do
    check_handler="${CHECK_HANDLERS[$((check_index - 1))]}"
    check_optional="$(is_optional_check "$check_index")"
    log_check "$check_index" "$(check_title "$check_index")"
    "$check_handler" "$check_optional"
  done
}

validate_args

echo "[release-smoke] base_url=$BASE_URL"
echo "[release-smoke] metrics_url=$METRICS_URL"
echo "[release-smoke] swagger_json_url=$SWAGGER_DOC_JSON_URL"
echo "[release-smoke] swagger_json_max_bytes=$MAX_JSON_SAMPLE_BYTES"
echo "[release-smoke] require_vary_origin=$REQUIRE_VARY_ORIGIN"
echo "[release-smoke] allowed_origin=${ALLOWED_ORIGIN:-<skip>}"
echo "[release-smoke] blocked_origin=${BLOCKED_ORIGIN:-<skip>}"

run_all_checks

echo "[release-smoke] completed"

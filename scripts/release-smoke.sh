#!/bin/bash

set -euo pipefail

BASE_URL="http://localhost:8566"
METRICS_URL="http://127.0.0.1:9090/metrics"
LOG_PATH="/tmp/ai-gateway.log"
TRACE_PATH="/trace"
ALLOWED_ORIGIN=""
BLOCKED_ORIGIN=""

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
  : > /tmp/ai-gateway-smoke-body.txt
  curl -s -o /tmp/ai-gateway-smoke-body.txt -w "%{http_code}" "$url" || true
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
    cat /tmp/ai-gateway-smoke-body.txt >&2 || true
    exit 1
  fi
  echo "[release-smoke] PASS: $name"
}

expect_health_status() {
  local name="$1"
  local url="$2"
  expect_http_200 "$name" "$url"
  if ! grep -Eq 'healthy|ready' /tmp/ai-gateway-smoke-body.txt; then
    echo "[release-smoke] FAIL: $name body missing healthy/ready marker" >&2
    cat /tmp/ai-gateway-smoke-body.txt >&2 || true
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
    cat /tmp/ai-gateway-smoke-body.txt >&2 || true
    exit 1
  fi
  echo "[release-smoke] PASS: $name closed with http_code=$code"
}

echo "[release-smoke] base_url=$BASE_URL"
echo "[release-smoke] metrics_url=$METRICS_URL"
echo "[release-smoke] allowed_origin=${ALLOWED_ORIGIN:-<skip>}"
echo "[release-smoke] blocked_origin=${BLOCKED_ORIGIN:-<skip>}"

echo "[release-smoke] check 1/10: health"
expect_health_status "health" "$BASE_URL/health"

echo "[release-smoke] check 2/10: ready"
expect_health_status "ready" "$BASE_URL/ready"

echo "[release-smoke] check 3/10: docs center"
expect_http_200 "docs center" "$BASE_URL/docs"
if ! grep -qi '<!doctype html' /tmp/ai-gateway-smoke-body.txt; then
  echo "[release-smoke] FAIL: docs center did not return SPA shell" >&2
  exit 1
fi
if grep -q '/swagger/index.html' /tmp/ai-gateway-smoke-body.txt; then
  echo "[release-smoke] FAIL: docs center should not redirect to swagger" >&2
  exit 1
fi

echo "[release-smoke] check 4/10: swagger root redirect"
swaggerCode="$(curl -s -o /tmp/ai-gateway-smoke-body.txt -D /tmp/ai-gateway-smoke-headers.txt -w "%{http_code}" "$BASE_URL/swagger")"
swaggerLocationLine="$(grep -i '^Location:' /tmp/ai-gateway-smoke-headers.txt | tr -d '\r' || true)"
if [ "$swaggerCode" != "302" ] || [ "$swaggerLocationLine" != "Location: /swagger/index.html" ]; then
  echo "[release-smoke] FAIL: swagger root redirect check failed code=$swaggerCode location=$swaggerLocationLine expected='Location: /swagger/index.html'" >&2
  exit 1
fi
echo "[release-smoke] PASS: swagger root redirect => $swaggerLocationLine"

swaggerSlashCode="$(curl -s -o /tmp/ai-gateway-smoke-body.txt -D /tmp/ai-gateway-smoke-headers.txt -w "%{http_code}" "$BASE_URL/swagger/")"
swaggerSlashLocationLine="$(grep -i '^Location:' /tmp/ai-gateway-smoke-headers.txt | tr -d '\r' || true)"
if [ "$swaggerSlashCode" != "302" ] || [ "$swaggerSlashLocationLine" != "Location: /swagger/index.html" ]; then
  echo "[release-smoke] FAIL: swagger trailing slash redirect check failed code=$swaggerSlashCode location=$swaggerSlashLocationLine expected='Location: /swagger/index.html'" >&2
  exit 1
fi
echo "[release-smoke] PASS: swagger trailing slash redirect => $swaggerSlashLocationLine"

echo "[release-smoke] check 5/10: trace page asset"
expect_http_200 "trace page" "$BASE_URL$TRACE_PATH"
TRACE_ASSET="$(grep -oE '/assets/index-[A-Za-z0-9_-]+\.js' /tmp/ai-gateway-smoke-body.txt | head -1 || true)"
if [ -z "$TRACE_ASSET" ]; then
  echo "[release-smoke] FAIL: trace page missing asset reference" >&2
  exit 1
fi
expect_http_200 "trace asset" "$BASE_URL$TRACE_ASSET"
if grep -qi '<!doctype html' /tmp/ai-gateway-smoke-body.txt; then
  echo "[release-smoke] FAIL: trace asset returned html" >&2
  exit 1
fi

echo "[release-smoke] check 6/10: debug endpoints closed"
expect_not_http_200 "debug pprof" "$BASE_URL/debug/pprof/"

echo "[release-smoke] check 7/10: metrics on gateway port closed"
expect_not_http_200 "metrics on gateway port" "$BASE_URL/metrics"

echo "[release-smoke] check 8/10: metrics localhost only"
expect_http_200 "Metrics (localhost only)" "$METRICS_URL"

echo "[release-smoke] check 9/10: cache backend hint"
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

echo "[release-smoke] check 10/10: cors whitelist (optional)"
if [ -n "$ALLOWED_ORIGIN" ]; then
  code="$(curl -s -o /tmp/ai-gateway-smoke-body.txt -D /tmp/ai-gateway-smoke-headers.txt -w "%{http_code}" -H "Origin: $ALLOWED_ORIGIN" "$BASE_URL/health")"
  allowedHeader="$(grep -i '^Access-Control-Allow-Origin:' /tmp/ai-gateway-smoke-headers.txt | awk '{print $2}' | tr -d '\r' || true)"
  if [ "$code" != "200" ] || [ "$allowedHeader" != "$ALLOWED_ORIGIN" ]; then
    echo "[release-smoke] FAIL: cors allowed origin check failed code=$code allow_origin=$allowedHeader expected=$ALLOWED_ORIGIN" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: cors allowed origin => $ALLOWED_ORIGIN"
else
  echo "[release-smoke] SKIP: cors allowed origin"
fi

if [ -n "$BLOCKED_ORIGIN" ]; then
  code="$(curl -s -o /tmp/ai-gateway-smoke-body.txt -w "%{http_code}" -H "Origin: $BLOCKED_ORIGIN" "$BASE_URL/health" || true)"
  if [ "$code" != "403" ]; then
    echo "[release-smoke] FAIL: cors blocked origin should be 403 code=$code origin=$BLOCKED_ORIGIN" >&2
    exit 1
  fi
  echo "[release-smoke] PASS: cors blocked origin => $BLOCKED_ORIGIN"
else
  echo "[release-smoke] SKIP: cors blocked origin"
fi

echo "[release-smoke] completed"

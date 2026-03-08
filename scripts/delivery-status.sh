#!/bin/bash

set -euo pipefail

BASE_BRANCH="main"
PR_NUMBER=""
ALLOW_MISSING_PR=false

usage() {
  cat <<'EOF'
Usage: ./scripts/delivery-status.sh [--base-branch <branch>] [--pr <number>] [--allow-missing-pr]
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --base-branch)
      BASE_BRANCH="${2:-}"
      shift 2
      ;;
    --pr)
      PR_NUMBER="${2:-}"
      shift 2
      ;;
    --allow-missing-pr)
      ALLOW_MISSING_PR=true
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

detect_repo_slug() {
  if [ -n "${GITHUB_REPOSITORY:-}" ]; then
    echo "$GITHUB_REPOSITORY"
    return
  fi

  local remote_url
  remote_url="$(git remote get-url origin 2>/dev/null || true)"
  if [ -z "$remote_url" ]; then
    return
  fi

  remote_url="${remote_url%.git}"
  if [[ "$remote_url" == git@github.com:* ]]; then
    echo "${remote_url#git@github.com:}"
    return
  fi
  if [[ "$remote_url" == https://github.com/* ]]; then
    echo "${remote_url#https://github.com/}"
    return
  fi
}

find_pr_by_head() {
  local repo_slug="$1"
  local head_sha="$2"
  gh api "repos/${repo_slug}/commits/${head_sha}/pulls" --jq '.[0].number' 2>/dev/null || true
}

echo "[delivery-status] check 1/3: HEAD 是否命中 tag"
HEAD_SHA="$(git rev-parse --short HEAD)"
HEAD_TAGS="$(git tag --points-at HEAD | tr '\n' ' ' | xargs || true)"
if [ -z "$HEAD_TAGS" ]; then
  echo "[delivery-status] FAIL: HEAD(${HEAD_SHA}) 未命中 tag"
  exit 1
fi
echo "[delivery-status] PASS: tags=${HEAD_TAGS}"

echo "[delivery-status] check 2/3: PR 是否 merged"
if [ -z "$PR_NUMBER" ]; then
  if command -v gh >/dev/null 2>&1; then
    REPO_SLUG="$(detect_repo_slug)"
    if [ -n "$REPO_SLUG" ]; then
      PR_NUMBER="$(find_pr_by_head "$REPO_SLUG" "$(git rev-parse HEAD)")"
    fi
  fi
fi

if [ -z "$PR_NUMBER" ]; then
  if [ "$ALLOW_MISSING_PR" = true ]; then
    echo "[delivery-status] SKIP: 未提供 PR 且无法自动识别"
  else
    echo "[delivery-status] FAIL: 未提供 PR 且无法自动识别。请传入 --pr <number>" >&2
    exit 1
  fi
else
  if ! command -v gh >/dev/null 2>&1; then
    echo "[delivery-status] FAIL: gh CLI 不可用，无法校验 PR 状态" >&2
    exit 1
  fi
  PR_STATE="$(gh pr view "$PR_NUMBER" --json state --jq '.state')"
  if [ "$PR_STATE" != "MERGED" ]; then
    echo "[delivery-status] FAIL: PR #${PR_NUMBER} 状态为 ${PR_STATE}" >&2
    exit 1
  fi
  echo "[delivery-status] PASS: PR #${PR_NUMBER} 已 merged"
fi

echo "[delivery-status] check 3/3: ${BASE_BRANCH} 是否与 origin/${BASE_BRANCH} 对齐"
git fetch origin "$BASE_BRANCH"
LOCAL_SHA="$(git rev-parse "$BASE_BRANCH")"
REMOTE_SHA="$(git rev-parse "origin/${BASE_BRANCH}")"
if [ "$LOCAL_SHA" != "$REMOTE_SHA" ]; then
  echo "[delivery-status] FAIL: ${BASE_BRANCH} 与 origin/${BASE_BRANCH} 不一致" >&2
  echo "local=${LOCAL_SHA}" >&2
  echo "remote=${REMOTE_SHA}" >&2
  exit 1
fi
echo "[delivery-status] PASS: ${BASE_BRANCH} 对齐 origin/${BASE_BRANCH}"

echo "[delivery-status] all checks passed"

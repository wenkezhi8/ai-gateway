#!/bin/bash

set -euo pipefail

BASE_BRANCH="main"
OUTPUT_PATH=""

usage() {
  cat <<'EOF'
Usage: ./scripts/branch-hygiene-report.sh [--base-branch <branch>] [--output <path>]
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --base-branch)
      BASE_BRANCH="${2:-}"
      shift 2
      ;;
    --output)
      OUTPUT_PATH="${2:-}"
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

if [ -z "$OUTPUT_PATH" ]; then
  OUTPUT_PATH="/tmp/branch-hygiene-report.md"
fi

git fetch origin --prune

NOW="$(date -u '+%Y-%m-%d %H:%M:%S UTC')"

{
  echo "# 遗留分支定期报告"
  echo
  echo "- 生成时间: ${NOW}"
  echo "- 基线分支: origin/${BASE_BRANCH}"
  echo "- 策略: 仅报告，人工清理"
  echo
  echo "| 分支 | 最近提交 | 是否已合并到 ${BASE_BRANCH} |"
  echo "| --- | --- | --- |"

  while IFS='|' read -r branch_name commit_date; do
    [ -z "$branch_name" ] && continue

    if [ "$branch_name" = "HEAD" ] || [ "$branch_name" = "$BASE_BRANCH" ] || [ "$branch_name" = "develop" ]; then
      continue
    fi

    if git merge-base --is-ancestor "origin/${branch_name}" "origin/${BASE_BRANCH}" 2>/dev/null; then
      merged="是"
    else
      merged="否"
    fi

    echo "| ${branch_name} | ${commit_date} | ${merged} |"
  done < <(git for-each-ref --sort=-committerdate --format='%(refname:lstrip=3)|%(committerdate:iso8601)' refs/remotes/origin)
} > "$OUTPUT_PATH"

echo "[branch-hygiene] report generated: $OUTPUT_PATH"

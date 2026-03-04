#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

WORKTREE_INPUT=""
WORKTREE_PATH=""
BRANCH=""
REMOTE="origin"
DRY_RUN=false

usage() {
  cat <<'EOF'
Usage: ./scripts/pr-cleanup.sh --worktree <name-or-path> --branch <feature-branch> [--remote <remote>] [--dry-run]

Examples:
  ./scripts/pr-cleanup.sh --worktree settings-trace-logo-remediation --branch feature/settings-trace-logo-remediation
  ./scripts/pr-cleanup.sh --worktree /abs/path/to/.worktrees/foo --branch feature/foo --dry-run
EOF
}

run_cmd() {
  if [ "$DRY_RUN" = "true" ]; then
    echo "[dry-run] $*"
    return 0
  fi
  "$@"
}

while [ $# -gt 0 ]; do
  case "$1" in
    --worktree)
      WORKTREE_INPUT="${2:-}"
      shift 2
      ;;
    --branch)
      BRANCH="${2:-}"
      shift 2
      ;;
    --remote)
      REMOTE="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=true
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

if [ -z "$WORKTREE_INPUT" ] || [ -z "$BRANCH" ]; then
  usage
  exit 1
fi

if [[ "$WORKTREE_INPUT" = /* ]]; then
  WORKTREE_PATH="$WORKTREE_INPUT"
else
  WORKTREE_PATH="$PROJECT_ROOT/.worktrees/$WORKTREE_INPUT"
fi

echo "🧹 PR 交付收尾清理"
echo "   worktree: $WORKTREE_PATH"
echo "   branch:   $BRANCH"
echo "   remote:   $REMOTE"
echo ""

cd "$PROJECT_ROOT"

run_cmd git worktree remove "$WORKTREE_PATH"
run_cmd git checkout main
run_cmd git branch -D "$BRANCH"
run_cmd git push "$REMOTE" --delete "$BRANCH"

echo ""
echo "🔍 清理后状态"
run_cmd git worktree list
run_cmd git branch --list

echo "✅ 清理完成"

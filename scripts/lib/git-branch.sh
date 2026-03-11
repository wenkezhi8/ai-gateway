#!/bin/bash

GIT_FEATURE_BRANCH_PREFIX="codex/feature/"

git_current_branch() {
  git rev-parse --abbrev-ref HEAD 2>/dev/null || echo ""
}

git_is_feature_branch() {
  local branch="${1:-}"
  [[ "$branch" == "${GIT_FEATURE_BRANCH_PREFIX}"* ]]
}

git_require_feature_branch() {
  local script_name="${1:-script}"
  local branch
  branch="$(git_current_branch)"
  if ! git_is_feature_branch "$branch"; then
    echo "[$script_name] FAIL: $script_name should run on feature branch (expected prefix: ${GIT_FEATURE_BRANCH_PREFIX}, current=$branch)" >&2
    return 1
  fi
  return 0
}

git_make_feature_branch_name() {
  local agent_id="${1:-auto}"
  local slug="${2:-task}"
  echo "${GIT_FEATURE_BRANCH_PREFIX}${agent_id}/${slug}"
}

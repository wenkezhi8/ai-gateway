#!/bin/bash

cors_normalize_csv() {
  local raw="${1:-}"
  raw="${raw//$'\n'/,}"
  raw="${raw//$'\r'/,}"
  echo "${raw//[[:space:]]/}"
}

cors_allows_all_origins() {
  local normalized token
  normalized="$(cors_normalize_csv "${1:-}")"
  if [ -z "$normalized" ]; then
    return 1
  fi
  IFS=',' read -r -a tokens <<<"$normalized"
  for token in "${tokens[@]}"; do
    if [ "$token" = "*" ]; then
      return 0
    fi
  done
  return 1
}

cors_first_specific_origin() {
  local normalized token
  normalized="$(cors_normalize_csv "${1:-}")"
  if [ -z "$normalized" ]; then
    return 1
  fi
  IFS=',' read -r -a tokens <<<"$normalized"
  for token in "${tokens[@]}"; do
    if [ -n "$token" ] && [ "$token" != "*" ]; then
      echo "$token"
      return 0
    fi
  done
  return 1
}

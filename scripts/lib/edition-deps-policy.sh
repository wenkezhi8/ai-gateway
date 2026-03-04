#!/bin/bash

edition_required_dependencies() {
  local edition="${1:-standard}"
  case "$edition" in
    basic)
      printf '%s\n' "redis"
      ;;
    standard)
      printf '%s\n' "redis ollama"
      ;;
    enterprise)
      printf '%s\n' "redis ollama qdrant"
      ;;
    *)
      printf '%s\n' "redis ollama"
      ;;
  esac
}

edition_all_dependencies() {
  printf '%s\n' "redis ollama qdrant"
}

edition_dep_in_list() {
  local dep="$1"
  shift
  local item
  for item in "$@"; do
    if [[ "$item" == "$dep" ]]; then
      return 0
    fi
  done
  return 1
}

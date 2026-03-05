#!/bin/bash

edition_policy_path() {
  local script_dir project_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  project_dir="$(dirname "$(dirname "$script_dir")")"
  printf '%s\n' "${EDITION_POLICY_PATH:-$project_dir/configs/edition-dependency-policy.json}"
}

edition_policy_base_dependencies() {
  local edition="${1:-standard}"
  local policy_path="${2:-$(edition_policy_path)}"

  python3 - "$policy_path" "$edition" <<'PY'
import json
import pathlib
import sys

policy_path = pathlib.Path(sys.argv[1])
edition = sys.argv[2].strip().lower() or "standard"

if not policy_path.exists():
    print("__MISSING__")
    raise SystemExit(0)

try:
    data = json.loads(policy_path.read_text(encoding="utf-8") or "{}")
except Exception:
    print("__MISSING__")
    raise SystemExit(0)

base = data.get("base_by_edition")
if not isinstance(base, dict):
    print("__MISSING__")
    raise SystemExit(0)

deps = base.get(edition)
if not isinstance(deps, list):
    print("__MISSING__")
    raise SystemExit(0)

clean = [str(item).strip() for item in deps if str(item).strip()]
print(" ".join(clean))
PY
}

edition_policy_vector_enabled_dependencies() {
  local policy_path="${1:-$(edition_policy_path)}"

  python3 - "$policy_path" <<'PY'
import json
import pathlib
import sys

policy_path = pathlib.Path(sys.argv[1])

if not policy_path.exists():
    print("__MISSING__")
    raise SystemExit(0)

try:
    data = json.loads(policy_path.read_text(encoding="utf-8") or "{}")
except Exception:
    print("__MISSING__")
    raise SystemExit(0)

extra = data.get("vector_enabled_append_dependencies")
if not isinstance(extra, list):
    print("__MISSING__")
    raise SystemExit(0)

clean = [str(item).strip() for item in extra if str(item).strip()]
print(" ".join(clean))
PY
}

edition_base_dependencies() {
  local edition="${1:-standard}"
  local policy_line

  policy_line="$(edition_policy_base_dependencies "$edition")"
  if [[ "$policy_line" != "__MISSING__" ]]; then
    printf '%s\n' "$policy_line"
    return
  fi

  case "$edition" in
    basic)
      printf '%s\n' ""
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

edition_vector_cache_enabled_from_config() {
  local config_path="${1:-${CONFIG_PATH:-./configs/config.json}}"

  python3 - "$config_path" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])

# Keep backward compatibility with historical default config behavior.
default_enabled = True

if not path.exists():
    print("true" if default_enabled else "false")
    raise SystemExit(0)

try:
    data = json.loads(path.read_text(encoding="utf-8") or "{}")
except Exception:
    print("true" if default_enabled else "false")
    raise SystemExit(0)

vector_cache = data.get("vector_cache")
if isinstance(vector_cache, dict) and isinstance(vector_cache.get("enabled"), bool):
    print("true" if vector_cache["enabled"] else "false")
else:
    print("true" if default_enabled else "false")
PY
}

edition_required_dependencies() {
  local edition="${1:-standard}"
  local config_path="${2:-${CONFIG_PATH:-./configs/config.json}}"
  local base_line vector_enabled vector_extra_line
  local -a deps=()
  local -a vector_extra_deps=()

  base_line="$(edition_base_dependencies "$edition")"
  if [[ -n "$base_line" ]]; then
    # shellcheck disable=SC2206
    deps=($base_line)
  fi

  vector_enabled="$(edition_vector_cache_enabled_from_config "$config_path")"
  if [[ "$vector_enabled" == "true" ]]; then
    vector_extra_line="$(edition_policy_vector_enabled_dependencies)"
    if [[ "$vector_extra_line" == "__MISSING__" ]]; then
      vector_extra_line="redis"
    fi
    if [[ -n "$vector_extra_line" ]]; then
      # shellcheck disable=SC2206
      vector_extra_deps=($vector_extra_line)
    fi

    local dep
    for dep in "${vector_extra_deps[@]}"; do
      if [[ ${#deps[@]} -eq 0 ]] || ! edition_dep_in_list "$dep" "${deps[@]}"; then
        deps+=("$dep")
      fi
    done

    if [[ ${#deps[@]} -eq 0 ]]; then
      deps+=("redis")
    fi
  fi

  if [[ ${#deps[@]} -eq 0 ]]; then
    printf '%s\n' ""
    return
  fi
  printf '%s\n' "${deps[*]}"
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

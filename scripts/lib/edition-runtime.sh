#!/bin/bash

AIG_DEFAULT_EDITION_TYPE="${AIG_DEFAULT_EDITION_TYPE:-standard}"
AIG_DEFAULT_EDITION_RUNTIME="${AIG_DEFAULT_EDITION_RUNTIME:-docker}"
AIG_DEFAULT_REDIS_VERSION="${AIG_DEFAULT_REDIS_VERSION:-7.2.0-v18}"
AIG_DEFAULT_OLLAMA_VERSION="${AIG_DEFAULT_OLLAMA_VERSION:-latest}"
AIG_DEFAULT_QDRANT_VERSION="${AIG_DEFAULT_QDRANT_VERSION:-latest}"

normalize_edition_type() {
  local value
  value="$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "$value" in
    basic|standard|enterprise) printf '%s' "$value" ;;
    *) printf '%s' "$AIG_DEFAULT_EDITION_TYPE" ;;
  esac
}

normalize_edition_runtime() {
  local value
  value="$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "$value" in
    docker|native) printf '%s' "$value" ;;
    *) printf '%s' "$AIG_DEFAULT_EDITION_RUNTIME" ;;
  esac
}

load_edition_runtime() {
  local config_path="${1:-${CONFIG_PATH:-./configs/config.json}}"
  local resolved

  resolved="$(python3 - "$config_path" \
    "$AIG_DEFAULT_EDITION_TYPE" \
    "$AIG_DEFAULT_EDITION_RUNTIME" \
    "$AIG_DEFAULT_REDIS_VERSION" \
    "$AIG_DEFAULT_OLLAMA_VERSION" \
    "$AIG_DEFAULT_QDRANT_VERSION" <<'PY'
import json
import pathlib
import sys

config_path = pathlib.Path(sys.argv[1])
default_type = sys.argv[2]
default_runtime = sys.argv[3]
default_redis = sys.argv[4]
default_ollama = sys.argv[5]
default_qdrant = sys.argv[6]

if config_path.exists():
    data = json.loads(config_path.read_text(encoding="utf-8") or "{}")
else:
    data = {}

edition = data.get("edition")
if not isinstance(edition, dict):
    edition = {}

edition_type = str(edition.get("type", default_type)).strip().lower()
if edition_type not in {"basic", "standard", "enterprise"}:
    edition_type = default_type

runtime = str(edition.get("runtime", default_runtime)).strip().lower()
if runtime not in {"docker", "native"}:
    runtime = default_runtime

versions = edition.get("dependency_versions")
if not isinstance(versions, dict):
    versions = {}

redis_version = str(versions.get("redis", default_redis)).strip() or default_redis
ollama_version = str(versions.get("ollama", default_ollama)).strip() or default_ollama
qdrant_version = str(versions.get("qdrant", default_qdrant)).strip() or default_qdrant

print(f"EDITION_TYPE={edition_type}")
print(f"EDITION_RUNTIME={runtime}")
print(f"REDIS_VERSION={redis_version}")
print(f"OLLAMA_VERSION={ollama_version}")
print(f"QDRANT_VERSION={qdrant_version}")
PY
)"

  while IFS='=' read -r key value; do
    case "$key" in
      EDITION_TYPE|EDITION_RUNTIME|REDIS_VERSION|OLLAMA_VERSION|QDRANT_VERSION)
        export "$key=$value"
        ;;
    esac
  done <<EOF
$resolved
EOF

  EDITION_TYPE="$(normalize_edition_type "${EDITION_TYPE:-$AIG_DEFAULT_EDITION_TYPE}")"
  EDITION_RUNTIME="$(normalize_edition_runtime "${EDITION_RUNTIME:-$AIG_DEFAULT_EDITION_RUNTIME}")"
  REDIS_VERSION="${REDIS_VERSION:-$AIG_DEFAULT_REDIS_VERSION}"
  OLLAMA_VERSION="${OLLAMA_VERSION:-$AIG_DEFAULT_OLLAMA_VERSION}"
  QDRANT_VERSION="${QDRANT_VERSION:-$AIG_DEFAULT_QDRANT_VERSION}"
  export EDITION_TYPE EDITION_RUNTIME REDIS_VERSION OLLAMA_VERSION QDRANT_VERSION
}

#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

EDITION="standard"
RUNTIME="docker"
APPLY_CONFIG="false"
PULL_EMBEDDING_MODEL="false"
CONFIG_PATH="${CONFIG_PATH:-$PROJECT_DIR/configs/config.json}"

REDIS_CONTAINER="ai-gateway-redis-stack"
OLLAMA_CONTAINER="ai-gateway-ollama"
QDRANT_CONTAINER="ai-gateway-qdrant"

usage() {
  cat <<'EOF'
Usage: setup-edition-env.sh [options]

Options:
  --edition <basic|standard|enterprise>
  --runtime <docker|native>
  --apply-config <true|false>
  --pull-embedding-model <true|false>
  --config-path <path>
EOF
}

log() {
  printf '[setup-edition] %s\n' "$*"
}

fail() {
  printf '[setup-edition] ERROR: %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "missing required command: $1"
  fi
}

is_bool() {
  case "$1" in
    true|false) return 0 ;;
    *) return 1 ;;
  esac
}

docker_available() {
  command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

ensure_docker_container() {
  local name="$1"
  local image="$2"
  local ports="$3"
  local -a port_args=()
  read -r -a port_args <<< "$ports"

  if docker ps --format '{{.Names}}' | grep -Fxq "$name"; then
    log "container $name already running"
    return
  fi

  if docker ps -a --format '{{.Names}}' | grep -Fxq "$name"; then
    log "starting existing container $name"
    docker start "$name" >/dev/null
    return
  fi

  log "creating container $name ($image)"
  docker run -d --name "$name" "${port_args[@]}" "$image" >/dev/null
}

ensure_redis_docker() {
  ensure_docker_container "$REDIS_CONTAINER" "redis/redis-stack-server:7.2.0-v18" "-p 6379:6379 -p 8001:8001"
}

ensure_ollama_docker() {
  ensure_docker_container "$OLLAMA_CONTAINER" "ollama/ollama:latest" "-p 11434:11434"
}

ensure_qdrant_docker() {
  ensure_docker_container "$QDRANT_CONTAINER" "qdrant/qdrant:latest" "-p 6333:6333"
}

ensure_redis_native_or_docker() {
  if command -v redis-cli >/dev/null 2>&1 && redis-cli -h 127.0.0.1 -p 6379 PING >/dev/null 2>&1; then
    log "redis native service detected"
    return
  fi
  log "redis native unavailable, fallback to docker"
  ensure_redis_docker
}

ensure_ollama_native_or_docker() {
  if ! command -v ollama >/dev/null 2>&1; then
    case "$(uname -s)" in
      Darwin)
        require_cmd brew
        log "installing ollama via brew"
        brew install ollama >/dev/null
        ;;
      Linux)
        require_cmd curl
        log "installing ollama via official script"
        curl -fsSL https://ollama.com/install.sh | sh >/dev/null
        ;;
      *)
        log "unsupported OS for native ollama install, fallback to docker"
        ensure_ollama_docker
        return
        ;;
    esac
  fi

  if ! curl -fsS "http://127.0.0.1:11434/api/tags" >/dev/null 2>&1; then
    if ! pgrep -f "ollama serve" >/dev/null 2>&1; then
      log "starting ollama serve"
      nohup ollama serve >/tmp/ollama-serve.log 2>&1 &
      sleep 3
    fi
  fi

  if ! curl -fsS "http://127.0.0.1:11434/api/tags" >/dev/null 2>&1; then
    log "ollama native unavailable, fallback to docker"
    ensure_ollama_docker
  else
    log "ollama native service detected"
  fi
}

ensure_qdrant_native_or_docker() {
  if curl -fsS "http://127.0.0.1:6333/collections" >/dev/null 2>&1; then
    log "qdrant native service detected"
    return
  fi
  log "qdrant native unavailable, fallback to docker"
  ensure_qdrant_docker
}

apply_config_values() {
  python3 - "$CONFIG_PATH" "$EDITION" <<'PY'
import json
import pathlib
import sys

config_path = pathlib.Path(sys.argv[1])
edition = sys.argv[2]

if not config_path.exists():
    config_path.parent.mkdir(parents=True, exist_ok=True)
    config_path.write_text("{}", encoding="utf-8")

data = json.loads(config_path.read_text(encoding="utf-8") or "{}")
data.setdefault("edition", {})["type"] = edition

vector_cache = data.setdefault("vector_cache", {})
if edition in ("standard", "enterprise"):
    vector_cache.setdefault("ollama_base_url", "http://127.0.0.1:11434")
if edition == "enterprise" and not vector_cache.get("cold_vector_qdrant_url"):
    vector_cache["cold_vector_qdrant_url"] = "http://127.0.0.1:6333"

config_path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

pull_embedding_model_if_needed() {
  if [[ "$PULL_EMBEDDING_MODEL" != "true" ]]; then
    return
  fi

  if command -v ollama >/dev/null 2>&1; then
    log "pulling embedding model via native ollama"
    ollama pull nomic-embed-text >/dev/null || true
    return
  fi

  if docker ps --format '{{.Names}}' | grep -Fxq "$OLLAMA_CONTAINER"; then
    log "pulling embedding model via docker ollama"
    docker exec "$OLLAMA_CONTAINER" ollama pull nomic-embed-text >/dev/null || true
  fi
}

check_redis() {
  command -v redis-cli >/dev/null 2>&1 && redis-cli -h 127.0.0.1 -p 6379 PING >/dev/null 2>&1
}

check_ollama() {
  curl -fsS "http://127.0.0.1:11434/api/tags" >/dev/null 2>&1
}

check_qdrant() {
  curl -fsS "http://127.0.0.1:6333/collections" >/dev/null 2>&1
}

validate_required_health() {
  local required=("redis")
  if [[ "$EDITION" == "standard" || "$EDITION" == "enterprise" ]]; then
    required+=("ollama")
  fi
  if [[ "$EDITION" == "enterprise" ]]; then
    required+=("qdrant")
  fi

  local dep
  for dep in "${required[@]}"; do
    case "$dep" in
      redis)
        check_redis || fail "redis health check failed"
        ;;
      ollama)
        check_ollama || fail "ollama health check failed"
        ;;
      qdrant)
        check_qdrant || fail "qdrant health check failed"
        ;;
    esac
    log "$dep healthy"
  done
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --edition)
      EDITION="$2"
      shift 2
      ;;
    --runtime)
      RUNTIME="$2"
      shift 2
      ;;
    --apply-config)
      APPLY_CONFIG="$2"
      shift 2
      ;;
    --pull-embedding-model)
      PULL_EMBEDDING_MODEL="$2"
      shift 2
      ;;
    --config-path)
      CONFIG_PATH="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

case "$EDITION" in
  basic|standard|enterprise) ;;
  *) fail "invalid edition: $EDITION" ;;
esac

case "$RUNTIME" in
  docker|native) ;;
  *) fail "invalid runtime: $RUNTIME" ;;
esac

is_bool "$APPLY_CONFIG" || fail "apply-config must be true/false"
is_bool "$PULL_EMBEDDING_MODEL" || fail "pull-embedding-model must be true/false"

mkdir -p "$(dirname "$CONFIG_PATH")"
if [[ ! -f "$CONFIG_PATH" ]]; then
  cp "$PROJECT_DIR/configs/config.example.json" "$CONFIG_PATH"
fi

if [[ "$RUNTIME" == "docker" ]]; then
  docker_available || fail "docker runtime requested but docker is unavailable"
  ensure_redis_docker
  if [[ "$EDITION" == "standard" || "$EDITION" == "enterprise" ]]; then
    ensure_ollama_docker
  fi
  if [[ "$EDITION" == "enterprise" ]]; then
    ensure_qdrant_docker
  fi
else
  if [[ "$EDITION" == "basic" || "$EDITION" == "standard" || "$EDITION" == "enterprise" ]]; then
    ensure_redis_native_or_docker
  fi
  if [[ "$EDITION" == "standard" || "$EDITION" == "enterprise" ]]; then
    ensure_ollama_native_or_docker
  fi
  if [[ "$EDITION" == "enterprise" ]]; then
    ensure_qdrant_native_or_docker
  fi
fi

if [[ "$APPLY_CONFIG" == "true" ]]; then
  log "applying edition config to $CONFIG_PATH"
  apply_config_values
fi

pull_embedding_model_if_needed
validate_required_health

log "setup completed: edition=$EDITION runtime=$RUNTIME apply_config=$APPLY_CONFIG pull_embedding_model=$PULL_EMBEDDING_MODEL"

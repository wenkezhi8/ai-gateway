#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

source "$SCRIPT_DIR/lib/container-names.sh"
source "$SCRIPT_DIR/lib/edition-deps-policy.sh"

EDITION="standard"
RUNTIME="docker"
APPLY_CONFIG="false"
PULL_EMBEDDING_MODEL="false"
CONFIG_PATH="${CONFIG_PATH:-$PROJECT_DIR/configs/config.json}"
EDITION_EXPLICIT="false"
RUNTIME_EXPLICIT="false"

LEGACY_REDIS_CONTAINER="redis-stack"

REDIS_VERSION="${REDIS_VERSION:-7.2.0-v18}"
OLLAMA_VERSION="${OLLAMA_VERSION:-latest}"
QDRANT_VERSION="${QDRANT_VERSION:-latest}"

REQUIRED_DEPENDENCIES=()
SUMMARY_ACTION=""
SUMMARY_DEPENDENCIES=""

usage() {
  cat <<'EOF'
Usage: setup-edition-env.sh [options]

Options:
  --edition <basic|standard|enterprise>
  --runtime <docker|native>
  --apply-config <true|false>
  --pull-embedding-model <true|false>
  --config-path <path>

说明:
  未传 --edition 时，脚本会在交互终端中提示选择版本（basic/standard/enterprise）。
  未传 --runtime 时，脚本会在交互终端中提示选择安装环境（docker/native）。
  非交互环境默认使用 standard + docker。
EOF
}

log() {
  printf '[setup-edition] %s\n' "$*"
}

fail() {
  printf '[setup-edition] ERROR: %s\n' "$*" >&2
  exit 1
}

print_manual_install_guidance() {
  local dep="$1"
  case "$dep" in
    redis)
      printf '[setup-edition] HINT: native install (example): brew install redis-stack && redis-stack-server\n' >&2
      printf '[setup-edition] HINT: docker manual (no auto fallback): docker run -d --name %s -p 6379:6379 -p 8001:8001 redis/redis-stack-server:7.2.0-v18\n' "$REDIS_CONTAINER" >&2
      ;;
    ollama)
      printf '[setup-edition] HINT: native install: https://ollama.com/download then run `ollama serve`\n' >&2
      printf '[setup-edition] HINT: docker manual (no auto fallback): docker run -d --name %s -p 11434:11434 ollama/ollama:latest\n' "$OLLAMA_CONTAINER" >&2
      ;;
    qdrant)
      printf '[setup-edition] HINT: native install: https://qdrant.tech/documentation/quickstart/\n' >&2
      printf '[setup-edition] HINT: docker manual (no auto fallback): docker run -d --name %s -p 6333:6333 qdrant/qdrant:latest\n' "$QDRANT_CONTAINER" >&2
      ;;
  esac
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

normalize_choice_input() {
  printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]' | xargs
}

parse_edition_choice() {
  case "$(normalize_choice_input "$1")" in
    1|basic) printf 'basic' ;;
    2|standard) printf 'standard' ;;
    3|enterprise) printf 'enterprise' ;;
    *) printf '' ;;
  esac
}

parse_runtime_choice() {
  case "$(normalize_choice_input "$1")" in
    1|docker) printf 'docker' ;;
    2|native) printf 'native' ;;
    *) printf '' ;;
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
  ensure_no_redis_container_conflict
  ensure_docker_container "$REDIS_CONTAINER" "redis/redis-stack-server:${REDIS_VERSION}" "-p 6379:6379 -p 8001:8001"
}

ensure_ollama_docker() {
  ensure_docker_container "$OLLAMA_CONTAINER" "ollama/ollama:${OLLAMA_VERSION}" "-p 11434:11434"
}

ensure_qdrant_docker() {
  ensure_docker_container "$QDRANT_CONTAINER" "qdrant/qdrant:${QDRANT_VERSION}" "-p 6333:6333"
}

ensure_no_redis_container_conflict() {
  if docker ps -a --format '{{.Names}}' | grep -Fxq "$REDIS_CONTAINER" && docker ps -a --format '{{.Names}}' | grep -Fxq "$LEGACY_REDIS_CONTAINER"; then
    fail "conflicting redis containers detected: $REDIS_CONTAINER and $LEGACY_REDIS_CONTAINER coexist. Cleanup manually: docker rm -f $LEGACY_REDIS_CONTAINER (or keep legacy and remove $REDIS_CONTAINER). No auto cleanup performed"
  fi
}

ensure_redis_native() {
  if command -v redis-cli >/dev/null 2>&1 && redis-cli -h 127.0.0.1 -p 6379 PING >/dev/null 2>&1; then
    log "redis native service detected"
    return
  fi
  print_manual_install_guidance redis
  fail "redis native dependency unavailable on 127.0.0.1:6379; docker fallback disabled for native runtime. Please install/start native Redis Stack manually"
}

ensure_ollama_native() {
  if ! command -v ollama >/dev/null 2>&1; then
    print_manual_install_guidance ollama
    fail "ollama binary not found; docker fallback disabled for native runtime. Please install/start native ollama manually"
  fi

  if ! curl -fsS "http://127.0.0.1:11434/api/tags" >/dev/null 2>&1; then
    if ! pgrep -f "ollama serve" >/dev/null 2>&1; then
      log "starting ollama serve"
      nohup ollama serve >/tmp/ollama-serve.log 2>&1 &
      sleep 3
    fi
  fi

  if ! curl -fsS "http://127.0.0.1:11434/api/tags" >/dev/null 2>&1; then
    print_manual_install_guidance ollama
    fail "ollama native service unavailable on 127.0.0.1:11434; docker fallback disabled for native runtime. Please start native ollama manually"
  fi
  log "ollama native service detected"
}

ensure_qdrant_native() {
  if curl -fsS "http://127.0.0.1:6333/collections" >/dev/null 2>&1; then
    log "qdrant native service detected"
    return
  fi
  print_manual_install_guidance qdrant
  fail "qdrant native service unavailable on 127.0.0.1:6333; docker fallback disabled for native runtime. Please install/start native qdrant manually"
}

apply_config_values() {
  python3 - "$CONFIG_PATH" "$EDITION" "$RUNTIME" "$REDIS_VERSION" "$OLLAMA_VERSION" "$QDRANT_VERSION" <<'PY'
import json
import pathlib
import sys

config_path = pathlib.Path(sys.argv[1])
edition = sys.argv[2]
runtime = sys.argv[3]
redis_version = sys.argv[4]
ollama_version = sys.argv[5]
qdrant_version = sys.argv[6]

if not config_path.exists():
    config_path.parent.mkdir(parents=True, exist_ok=True)
    config_path.write_text("{}", encoding="utf-8")

data = json.loads(config_path.read_text(encoding="utf-8") or "{}")
edition_data = data.setdefault("edition", {})
edition_data["type"] = edition
edition_data["runtime"] = runtime

dependency_versions = edition_data.get("dependency_versions")
if not isinstance(dependency_versions, dict):
    dependency_versions = {}
dependency_versions["redis"] = dependency_versions.get("redis") or redis_version
dependency_versions["ollama"] = dependency_versions.get("ollama") or ollama_version
dependency_versions["qdrant"] = dependency_versions.get("qdrant") or qdrant_version
edition_data["dependency_versions"] = dependency_versions

vector_cache = data.setdefault("vector_cache", {})
if edition in ("standard", "enterprise"):
    vector_cache["enabled"] = True
elif edition == "basic":
    vector_cache["enabled"] = False
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

resolve_required_dependencies() {
  local required_line
  required_line="$(edition_required_dependencies "$EDITION" "$CONFIG_PATH")"

  # Apply-config 模式下按目标版本默认行为计算，避免 basic 被旧配置中的向量开关误伤。
  if [[ "$APPLY_CONFIG" == "true" && "$EDITION" == "basic" ]]; then
    required_line=""
  fi

  # shellcheck disable=SC2206
  REQUIRED_DEPENDENCIES=($required_line)
}

validate_required_health() {
  local required=()
  if [[ ${#REQUIRED_DEPENDENCIES[@]} -gt 0 ]]; then
    required=("${REQUIRED_DEPENDENCIES[@]}")
  fi

  if [[ ${#required[@]} -eq 0 ]]; then
    return
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

stop_all_dependencies_by_runtime() {
  if [[ "$RUNTIME" == "docker" ]]; then
    local running_names container
    local -a running_containers=()
    running_names="$(docker ps --format '{{.Names}}')"

    for container in "$REDIS_CONTAINER" "$OLLAMA_CONTAINER" "$QDRANT_CONTAINER"; do
      if printf '%s\n' "$running_names" | grep -Fxq "$container"; then
        running_containers+=("$container")
      fi
    done

    if [[ ${#running_containers[@]} -gt 0 ]]; then
      log "基础版：停止所有依赖 -> 批量停止容器: ${running_containers[*]}"
      docker stop "${running_containers[@]}" >/dev/null 2>&1 || true
      SUMMARY_DEPENDENCIES="${running_containers[*]}"
    else
      log "基础版：停止所有依赖 -> 未检测到运行中的依赖容器"
      SUMMARY_DEPENDENCIES="无（未检测到运行中的依赖）"
    fi
    return
  fi

  # native 模式仅停止，不执行卸载。
  local -a stopped_native_deps=()

  if pgrep -f "redis-server|redis-stack-server" >/dev/null 2>&1; then
    pkill -f "redis-server|redis-stack-server" >/dev/null 2>&1 || true
    log "基础版：停止所有依赖 -> 已尝试停止 native redis"
    stopped_native_deps+=("redis")
  else
    log "基础版：停止所有依赖 -> native redis 未运行，跳过"
  fi

  if pgrep -f "ollama serve" >/dev/null 2>&1; then
    pkill -f "ollama serve" >/dev/null 2>&1 || true
    log "基础版：停止所有依赖 -> 已尝试停止 native ollama"
    stopped_native_deps+=("ollama")
  else
    log "基础版：停止所有依赖 -> native ollama 未运行，跳过"
  fi

  if pgrep -f "qdrant" >/dev/null 2>&1; then
    pkill -f "qdrant" >/dev/null 2>&1 || true
    log "基础版：停止所有依赖 -> 已尝试停止 native qdrant"
    stopped_native_deps+=("qdrant")
  else
    log "基础版：停止所有依赖 -> native qdrant 未运行，跳过"
  fi

  if [[ ${#stopped_native_deps[@]} -gt 0 ]]; then
    SUMMARY_DEPENDENCIES="${stopped_native_deps[*]}"
  else
    SUMMARY_DEPENDENCIES="无（未检测到运行中的依赖）"
  fi
}

choose_edition_interactively_if_needed() {
  if [[ "$EDITION_EXPLICIT" == "true" ]]; then
    return
  fi

  if [[ ! -t 0 ]]; then
    log "未指定 --edition，当前为非交互环境，默认使用 standard。"
    EDITION="standard"
    return
  fi

  echo "请选择版本（basic/standard/enterprise）："
  echo "  1) basic（基础版：停止所有依赖）"
  echo "  2) standard（推荐）"
  echo "  3) enterprise"

  while true; do
    read -r -p "请输入 1/2/3（默认 2）: " edition_choice
    edition_choice="${edition_choice:-2}"
    parsed_edition="$(parse_edition_choice "$edition_choice")"
    if [[ -n "$parsed_edition" ]]; then
      EDITION="$parsed_edition"
      break
    fi
    echo "输入无效，请输入 1、2、3 或 basic/standard/enterprise。"
  done

  log "已选择版本: $EDITION"
}

choose_runtime_interactively_if_needed() {
  if [[ "$RUNTIME_EXPLICIT" == "true" ]]; then
    return
  fi

  if [[ ! -t 0 ]]; then
    log "未指定 --runtime，当前为非交互环境，默认使用 docker。"
    RUNTIME="docker"
    return
  fi

  echo "请选择安装环境（docker/native）："
  echo "  1) docker（推荐）"
  echo "  2) native（本机安装）"

  while true; do
    read -r -p "请输入 1 或 2（默认 1）: " runtime_choice
    runtime_choice="${runtime_choice:-1}"
    parsed_runtime="$(parse_runtime_choice "$runtime_choice")"
    if [[ -n "$parsed_runtime" ]]; then
      RUNTIME="$parsed_runtime"
      break
    fi
    echo "输入无效，请输入 1 或 2。"
  done

  log "已选择安装环境: $RUNTIME"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --edition)
      EDITION="$2"
      EDITION_EXPLICIT="true"
      shift 2
      ;;
    --runtime)
      RUNTIME="$2"
      RUNTIME_EXPLICIT="true"
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

choose_edition_interactively_if_needed
choose_runtime_interactively_if_needed

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

resolve_required_dependencies

if [[ "$EDITION" == "basic" ]]; then
  SUMMARY_ACTION="停止依赖"
  log "基础版：停止所有依赖"
  if [[ "$RUNTIME" == "docker" ]]; then
    docker_available || fail "docker runtime requested but docker is unavailable"
  fi
  stop_all_dependencies_by_runtime
else
  SUMMARY_ACTION="安装/确保依赖运行"
  if [[ ${#REQUIRED_DEPENDENCIES[@]} -gt 0 ]]; then
    SUMMARY_DEPENDENCIES="${REQUIRED_DEPENDENCIES[*]}"
  else
    SUMMARY_DEPENDENCIES="无"
  fi
  if [[ "$RUNTIME" == "docker" ]]; then
    docker_available || fail "docker runtime requested but docker is unavailable"
    local_dep=""
    if [[ ${#REQUIRED_DEPENDENCIES[@]} -gt 0 ]]; then
      for local_dep in "${REQUIRED_DEPENDENCIES[@]}"; do
        case "$local_dep" in
          redis)
            ensure_redis_docker
            ;;
          ollama)
            ensure_ollama_docker
            ;;
          qdrant)
            ensure_qdrant_docker
            ;;
        esac
      done
    fi
  else
    local_dep=""
    if [[ ${#REQUIRED_DEPENDENCIES[@]} -gt 0 ]]; then
      for local_dep in "${REQUIRED_DEPENDENCIES[@]}"; do
        case "$local_dep" in
          redis)
            ensure_redis_native
            ;;
          ollama)
            ensure_ollama_native
            ;;
          qdrant)
            ensure_qdrant_native
            ;;
        esac
      done
    fi
  fi
fi

if [[ "$APPLY_CONFIG" == "true" ]]; then
  log "applying edition config to $CONFIG_PATH"
  apply_config_values
fi

pull_embedding_model_if_needed
if [[ "$EDITION" != "basic" ]]; then
  validate_required_health
fi

log "本次动作摘要:"
log "  版本: $EDITION"
log "  环境: $RUNTIME"
log "  动作: $SUMMARY_ACTION"
log "  依赖清单: $SUMMARY_DEPENDENCIES"

log "setup completed: edition=$EDITION runtime=$RUNTIME apply_config=$APPLY_CONFIG pull_embedding_model=$PULL_EMBEDDING_MODEL"

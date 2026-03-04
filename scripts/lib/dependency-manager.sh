#!/bin/bash

DEPENDENCY_MANAGER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$DEPENDENCY_MANAGER_DIR/container-names.sh"
source "$DEPENDENCY_MANAGER_DIR/edition-deps-policy.sh"

LEGACY_REDIS_CONTAINER="redis-stack"

dep_log() {
  printf '[dependency-manager] %s\n' "$*"
}

dep_fail() {
  printf '[dependency-manager] ERROR: %s\n' "$*" >&2
  exit 1
}

check_command_exists() {
  command -v "$1" >/dev/null 2>&1
}

docker_available() {
  check_command_exists docker && docker info >/dev/null 2>&1
}

ensure_no_redis_container_conflict() {
  if docker ps -a --format '{{.Names}}' | grep -Fxq "$REDIS_CONTAINER" && docker ps -a --format '{{.Names}}' | grep -Fxq "$LEGACY_REDIS_CONTAINER"; then
    dep_fail "发现冲突容器: $REDIS_CONTAINER 与 $LEGACY_REDIS_CONTAINER 共存，请手动清理旧容器后重试"
  fi
}

dependency_container_name() {
  case "$1" in
    redis) printf '%s\n' "$REDIS_CONTAINER" ;;
    ollama) printf '%s\n' "$OLLAMA_CONTAINER" ;;
    qdrant) printf '%s\n' "$QDRANT_CONTAINER" ;;
    *) printf '%s\n' "" ;;
  esac
}

dependency_image() {
  case "$1" in
    redis) printf 'redis/redis-stack-server:%s\n' "${REDIS_VERSION:-7.2.0-v18}" ;;
    ollama) printf 'ollama/ollama:%s\n' "${OLLAMA_VERSION:-latest}" ;;
    qdrant) printf 'qdrant/qdrant:%s\n' "${QDRANT_VERSION:-latest}" ;;
    *) printf '%s\n' "" ;;
  esac
}

dependency_ports() {
  case "$1" in
    redis) printf '%s\n' "-p 6379:6379 -p 8001:8001" ;;
    ollama) printf '%s\n' "-p 11434:11434" ;;
    qdrant) printf '%s\n' "-p 6333:6333" ;;
    *) printf '%s\n' "" ;;
  esac
}

dependency_native_install_hint() {
  case "$1" in
    redis)
      printf '%s\n' "brew install redis-stack && redis-stack-server"
      ;;
    ollama)
      printf '%s\n' "https://ollama.com/download (安装后执行: ollama serve)"
      ;;
    qdrant)
      printf '%s\n' "https://qdrant.tech/documentation/quickstart/"
      ;;
  esac
}

dependency_docker_manual_hint() {
  local dep="$1"
  local name image ports
  name="$(dependency_container_name "$dep")"
  image="$(dependency_image "$dep")"
  ports="$(dependency_ports "$dep")"
  printf 'docker run -d --name %s %s %s\n' "$name" "$ports" "$image"
}

required_dependencies_array() {
  local required_line
  required_line="$(edition_required_dependencies "${EDITION_TYPE:-standard}")"
  # shellcheck disable=SC2206
  REQUIRED_DEPENDENCIES=($required_line)
}

all_dependencies_array() {
  local all_line
  all_line="$(edition_all_dependencies)"
  # shellcheck disable=SC2206
  ALL_DEPENDENCIES=($all_line)
}

dependency_health_check_docker() {
  local dep="$1"
  case "$dep" in
    redis)
      docker exec "$REDIS_CONTAINER" redis-cli PING >/dev/null 2>&1
      ;;
    ollama)
      curl -fsS "http://127.0.0.1:11434/api/tags" >/dev/null 2>&1
      ;;
    qdrant)
      curl -fsS "http://127.0.0.1:6333/collections" >/dev/null 2>&1
      ;;
    *)
      return 1
      ;;
  esac
}

dependency_health_check_native() {
  local dep="$1"
  case "$dep" in
    redis)
      check_command_exists redis-cli && redis-cli -h 127.0.0.1 -p 6379 PING >/dev/null 2>&1
      ;;
    ollama)
      check_command_exists ollama && curl -fsS "http://127.0.0.1:11434/api/tags" >/dev/null 2>&1
      ;;
    qdrant)
      curl -fsS "http://127.0.0.1:6333/collections" >/dev/null 2>&1
      ;;
    *)
      return 1
      ;;
  esac
}

validate_required_health() {
  required_dependencies_array
  local dep
  for dep in "${REQUIRED_DEPENDENCIES[@]}"; do
    if [[ "${EDITION_RUNTIME:-docker}" == "docker" ]]; then
      dependency_health_check_docker "$dep" || dep_fail "$dep 健康检查失败，请先执行“安装依赖”或手动修复。Docker 手动命令: $(dependency_docker_manual_hint "$dep")"
    else
      dependency_health_check_native "$dep" || dep_fail "$dep 健康检查失败（native 模式不会自动回退到 Docker）。native 安装指引: $(dependency_native_install_hint "$dep")"
    fi
    dep_log "$dep 健康检查通过"
  done
}

ensure_docker_container() {
  local dep="$1"
  local name image ports
  name="$(dependency_container_name "$dep")"
  image="$(dependency_image "$dep")"
  ports="$(dependency_ports "$dep")"
  local -a port_args=()
  read -r -a port_args <<<"$ports"

  if docker ps --format '{{.Names}}' | grep -Fxq "$name"; then
    dep_log "$name 已运行"
    return
  fi
  if docker ps -a --format '{{.Names}}' | grep -Fxq "$name"; then
    dep_log "启动已有容器 $name"
    docker start "$name" >/dev/null
    return
  fi
  dep_log "创建容器 $name ($image)"
  docker run -d --name "$name" "${port_args[@]}" "$image" >/dev/null
}

ensure_required_running() {
  required_dependencies_array
  local dep

  if [[ "${EDITION_RUNTIME:-docker}" == "docker" ]]; then
    docker_available || dep_fail "当前配置 runtime=docker，但 Docker 不可用"
    ensure_no_redis_container_conflict
    for dep in "${REQUIRED_DEPENDENCIES[@]}"; do
      ensure_docker_container "$dep"
    done
    return
  fi

  for dep in "${REQUIRED_DEPENDENCIES[@]}"; do
    case "$dep" in
      redis)
        check_command_exists redis-cli || dep_fail "缺少 redis-cli（native 模式不会自动回退到 Docker）。安装参考: $(dependency_native_install_hint redis)"
        dependency_health_check_native redis || dep_fail "redis 未就绪（native 模式不会自动回退到 Docker）。安装参考: $(dependency_native_install_hint redis)"
        ;;
      ollama)
        check_command_exists ollama || dep_fail "缺少 ollama（native 模式不会自动回退到 Docker）。安装参考: $(dependency_native_install_hint ollama)"
        dependency_health_check_native ollama || dep_fail "ollama 未就绪（native 模式不会自动回退到 Docker）。安装参考: $(dependency_native_install_hint ollama)"
        ;;
      qdrant)
        dependency_health_check_native qdrant || dep_fail "qdrant 未就绪（native 模式不会自动回退到 Docker）。安装参考: $(dependency_native_install_hint qdrant)"
        ;;
    esac
  done
}

stop_non_required_native_best_effort() {
  local dep="$1"
  case "$dep" in
    ollama)
      if pgrep -f "ollama serve" >/dev/null 2>&1; then
        pkill -f "ollama serve" >/dev/null 2>&1 || true
        dep_log "已停止非必需 native 依赖: ollama"
      fi
      ;;
    qdrant)
      if pgrep -f "qdrant" >/dev/null 2>&1; then
        pkill -f "qdrant" >/dev/null 2>&1 || true
        dep_log "已停止非必需 native 依赖: qdrant"
      fi
      ;;
  esac
}

stop_non_required() {
  required_dependencies_array
  all_dependencies_array

  local dep name
  for dep in "${ALL_DEPENDENCIES[@]}"; do
    if edition_dep_in_list "$dep" "${REQUIRED_DEPENDENCIES[@]}"; then
      continue
    fi

    if [[ "${EDITION_RUNTIME:-docker}" == "docker" ]]; then
      name="$(dependency_container_name "$dep")"
      if [[ -n "$name" ]] && docker ps --format '{{.Names}}' | grep -Fxq "$name"; then
        dep_log "停止非必需容器 $name"
        docker stop "$name" >/dev/null || true
      fi
    else
      stop_non_required_native_best_effort "$dep"
    fi
  done
}

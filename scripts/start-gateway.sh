#!/bin/bash

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_PATH="${CONFIG_PATH:-$PROJECT_DIR/configs/config.json}"
COMPOSE_FILE="${PROJECT_DIR}/docker-compose.yml"
ENV_FILE="${PROJECT_DIR}/.env"

source "$SCRIPT_DIR/lib/edition-runtime.sh"
source "$SCRIPT_DIR/lib/edition-deps-policy.sh"

WITH_MONITORING=false
ACTION="start"

print_banner() {
  echo -e "${BLUE}"
  echo "  _    ___   __  ____  ____  ____  ____  ____     _    ____  ____  "
  echo " / \  |_ _| /  \/ ___||  _ \/ ___||  _ \|  _ \   / \  |  _ \|  _ \ "
  echo "/ _ \  | | / _ \___ \| |_) \___ \| |_) | |_) | / _ \ | |_) | |_) |"
  echo "/ ___ \ | |/ ___ \__) |  __/ ___) |  __/|  _ < / ___ \|  __/|  __/ "
  echo "/_/   \_\___/_/   \_____|_|   |_____|_|   |_| \_\_/   \_\_|   |_|    "
  echo -e "${NC}"
}

print_usage() {
  cat <<'EOF'
Usage: ./scripts/start-gateway.sh [OPTIONS]

Options:
  -m, --monitoring    Start with monitoring stack
  -s, --stop          Stop services
  -r, --restart       Restart services
  -l, --logs          Show logs
  -h, --help          Show this help
EOF
}

require_docker() {
  echo -e "${YELLOW}[1/4] Checking Docker availability...${NC}"
  if ! command -v docker >/dev/null 2>&1; then
    echo -e "${RED}Docker is not installed.${NC}"
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}Docker daemon is not running.${NC}"
    exit 1
  fi
  echo -e "${GREEN}[OK] Docker is available${NC}"
}

resolve_compose_cmd() {
  if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD=(docker compose)
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD=(docker-compose)
  else
    echo -e "${RED}Docker Compose is not installed.${NC}"
    exit 1
  fi
}

compose() {
  "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" "$@"
}

ensure_env_file() {
  if [[ -f "$ENV_FILE" ]]; then
    return
  fi
  if [[ -f "${PROJECT_DIR}/.env.example" ]]; then
    cp "${PROJECT_DIR}/.env.example" "$ENV_FILE"
    echo -e "${YELLOW}Created .env from .env.example${NC}"
    return
  fi
  cat >"$ENV_FILE" <<'EOF'
GATEWAY_PORT=8000
WEB_PORT=3000
REDIS_PORT=6379
EOF
  echo -e "${YELLOW}Created minimal .env${NC}"
}

resolve_required_services() {
  local required_line
  required_line="$(edition_required_dependencies "$EDITION_TYPE")"
  # shellcheck disable=SC2206
  REQUIRED_DEPS=($required_line)
  SERVICES=(gateway web "${REQUIRED_DEPS[@]}")
  if [[ "$WITH_MONITORING" == "true" ]]; then
    SERVICES+=(prometheus grafana alertmanager)
  fi
}

guard_runtime_for_docker_entry() {
  load_edition_runtime "$CONFIG_PATH"
  if [[ "$EDITION_RUNTIME" == "native" ]]; then
    echo -e "${RED}runtime=native，不支持通过 start-gateway.sh 启动 Docker 入口。${NC}"
    echo "请改用 ./scripts/dev-restart.sh，或在 /settings 中切换 runtime 为 docker。"
    exit 1
  fi
}

stop_non_required_dependencies() {
  local all_line dep
  all_line="$(edition_all_dependencies)"
  # shellcheck disable=SC2206
  ALL_DEPS=($all_line)

  for dep in "${ALL_DEPS[@]}"; do
    if edition_dep_in_list "$dep" "${REQUIRED_DEPS[@]}"; then
      continue
    fi
    compose stop "$dep" >/dev/null 2>&1 || true
  done
}

start_services() {
  load_edition_runtime "$CONFIG_PATH"
  resolve_required_services

  echo -e "${YELLOW}[2/4] Edition policy: type=${EDITION_TYPE} runtime=${EDITION_RUNTIME}${NC}"
  echo -e "${YELLOW}[3/4] Starting required services: ${SERVICES[*]}${NC}"
  compose pull "${SERVICES[@]}" >/dev/null 2>&1 || true
  compose up -d --build "${SERVICES[@]}"

  stop_non_required_dependencies
  echo -e "${GREEN}[4/4] Services are up (only required dependencies enabled)${NC}"
}

stop_services() {
  resolve_compose_cmd
  compose down
  echo -e "${GREEN}Services stopped${NC}"
}

show_logs() {
  resolve_compose_cmd
  compose logs -f
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--monitoring)
      WITH_MONITORING=true
      shift
      ;;
    -s|--stop)
      ACTION="stop"
      shift
      ;;
    -r|--restart)
      ACTION="restart"
      shift
      ;;
    -l|--logs)
      ACTION="logs"
      shift
      ;;
    -h|--help)
      print_usage
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      print_usage
      exit 1
      ;;
  esac
done

print_banner

case "$ACTION" in
  start)
    guard_runtime_for_docker_entry
    require_docker
    resolve_compose_cmd
    ensure_env_file
    start_services
    ;;
  stop)
    stop_services
    ;;
  restart)
    guard_runtime_for_docker_entry
    stop_services
    sleep 1
    require_docker
    resolve_compose_cmd
    ensure_env_file
    start_services
    ;;
  logs)
    show_logs
    ;;
esac

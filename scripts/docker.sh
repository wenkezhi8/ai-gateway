#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$PROJECT_ROOT/deploy/docker"
COMPOSE_FILE="$DEPLOY_DIR/docker-compose.yml"
CONFIG_PATH="${CONFIG_PATH:-$PROJECT_ROOT/configs/config.json}"

source "$SCRIPT_DIR/lib/edition-runtime.sh"
source "$SCRIPT_DIR/lib/edition-deps-policy.sh"

resolve_compose_cmd() {
  if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD=(docker compose)
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD=(docker-compose)
  else
    echo "Docker Compose not found"
    exit 1
  fi
}

compose() {
  "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" "$@"
}

resolve_required_services() {
  local required_line
  required_line="$(edition_required_dependencies "$EDITION_TYPE")"
  # shellcheck disable=SC2206
  REQUIRED_DEPS=($required_line)
  SERVICES=(gateway nginx "${REQUIRED_DEPS[@]}")
}

stop_non_required_deps() {
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

guard_runtime_for_docker_entry() {
  load_edition_runtime "$CONFIG_PATH"
  if [[ "$EDITION_RUNTIME" == "native" ]]; then
    echo "runtime=native，不支持 docker.sh up/restart。"
    echo "请改用 ./scripts/dev-restart.sh，或在 /settings 中切换 runtime 为 docker。"
    exit 1
  fi
}

resolve_compose_cmd

echo "🐳 AI Gateway Docker Deployment"

case "${1:-}" in
  build)
    echo "🔨 Building Docker images..."
    compose build
    echo "✅ Build complete"
    ;;

  up)
    guard_runtime_for_docker_entry
    resolve_required_services
    echo "🚀 Starting required services: ${SERVICES[*]}"
    compose up -d "${SERVICES[@]}"
    stop_non_required_deps
    echo "✅ Services started"
    echo "   Gateway: http://localhost:8080"
    echo "   Console: http://localhost:80"
    ;;

  down)
    echo "🛑 Stopping services..."
    compose down
    echo "✅ Services stopped"
    ;;

  logs)
    compose logs -f "${2:-}"
    ;;

  restart)
    guard_runtime_for_docker_entry
    resolve_required_services
    echo "🔄 Restarting required services: ${SERVICES[*]}"
    compose restart "${SERVICES[@]}"
    stop_non_required_deps
    echo "✅ Services restarted"
    ;;

  status)
    compose ps
    ;;

  clean)
    echo "🧹 Cleaning up..."
    compose down -v --remove-orphans
    docker system prune -f
    echo "✅ Cleanup complete"
    ;;

  *)
    echo "Usage: $0 {build|up|down|logs|restart|status|clean}"
    echo ""
    echo "Commands:"
    echo "  build   - Build Docker images"
    echo "  up      - Start required services by edition policy"
    echo "  down    - Stop services"
    echo "  logs    - View logs (optional: service name)"
    echo "  restart - Restart required services"
    echo "  status  - Show service status"
    echo "  clean   - Remove containers, volumes and cleanup"
    exit 1
    ;;
esac

#!/bin/bash

# AI Gateway - Docker Deployment Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$PROJECT_ROOT/deploy/docker"

echo "🐳 AI Gateway Docker Deployment"

case "$1" in
    build)
        echo "🔨 Building Docker images..."
        cd "$DEPLOY_DIR"
        docker-compose build
        echo "✅ Build complete"
        ;;

    up)
        echo "🚀 Starting services..."
        cd "$DEPLOY_DIR"
        docker-compose up -d
        echo "✅ Services started"
        echo "   Gateway: http://localhost:8080"
        echo "   Console: http://localhost:80"
        ;;

    down)
        echo "🛑 Stopping services..."
        cd "$DEPLOY_DIR"
        docker-compose down
        echo "✅ Services stopped"
        ;;

    logs)
        cd "$DEPLOY_DIR"
        docker-compose logs -f ${2:-}
        ;;

    restart)
        echo "🔄 Restarting services..."
        cd "$DEPLOY_DIR"
        docker-compose restart
        echo "✅ Services restarted"
        ;;

    status)
        cd "$DEPLOY_DIR"
        docker-compose ps
        ;;

    clean)
        echo "🧹 Cleaning up..."
        cd "$DEPLOY_DIR"
        docker-compose down -v --remove-orphans
        docker system prune -f
        echo "✅ Cleanup complete"
        ;;

    *)
        echo "Usage: $0 {build|up|down|logs|restart|status|clean}"
        echo ""
        echo "Commands:"
        echo "  build   - Build Docker images"
        echo "  up      - Start all services"
        echo "  down    - Stop all services"
        echo "  logs    - View logs (optional: service name)"
        echo "  restart - Restart all services"
        echo "  status  - Show service status"
        echo "  clean   - Remove containers, volumes and cleanup"
        exit 1
        ;;
esac

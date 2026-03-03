#!/bin/bash

# ============================================
# AI Gateway - Startup Script (Linux/Mac)
# ============================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default values
COMPOSE_FILE="${PROJECT_DIR}/docker-compose.yml"
ENV_FILE="${PROJECT_DIR}/.env"
WITH_MONITORING=false

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "  _    ___   __  ____  ____  ____  ____  ____     _    ____  ____  "
    echo " / \  |_ _| /  \/ ___||  _ \/ ___||  _ \|  _ \   / \  |  _ \|  _ \ "
    echo "/ _ \  | | / _ \___ \| |_) \___ \| |_) | |_) | / _ \ | |_) | |_) |"
    echo "/ ___ \ | |/ ___ \__) |  __/ ___) |  __/|  _ < / ___ \|  __/|  __/ "
    echo "/_/   \_\___/_/   \_____|_|   |_____|_|   |_| \_\_/   \_\_|   |_|    "
    echo -e "${NC}"
    echo -e "${GREEN}AI Gateway - One-Click Deployment${NC}"
    echo ""
}

# Print usage
print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -m, --monitoring    Start with monitoring stack (Prometheus + Grafana)"
    echo "  -s, --stop          Stop all services"
    echo "  -r, --restart       Restart all services"
    echo "  -l, --logs          Show logs"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                  # Start basic services"
    echo "  $0 --monitoring     # Start with monitoring"
    echo "  $0 --stop           # Stop all services"
    echo ""
}

# Check Docker installation
check_docker() {
    echo -e "${YELLOW}[1/5] Checking Docker installation...${NC}"
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: Docker is not installed.${NC}"
        echo "Please install Docker from: https://docs.docker.com/get-docker/"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo -e "${RED}Error: Docker Compose is not installed.${NC}"
        echo "Please install Docker Compose from: https://docs.docker.com/compose/install/"
        exit 1
    fi

    # Check if Docker is running
    if ! docker info &> /dev/null; then
        echo -e "${RED}Error: Docker daemon is not running.${NC}"
        echo "Please start Docker and try again."
        exit 1
    fi

    echo -e "${GREEN}[OK] Docker is installed and running${NC}"
}

# Check environment file
check_env_file() {
    echo -e "${YELLOW}[2/5] Checking environment configuration...${NC}"

    if [ ! -f "$ENV_FILE" ]; then
        echo -e "${YELLOW}Creating .env file from template...${NC}"
        if [ -f "${PROJECT_DIR}/.env.example" ]; then
            cp "${PROJECT_DIR}/.env.example" "$ENV_FILE"
            echo -e "${GREEN}[OK] Created .env file. Please edit it to add your API keys.${NC}"
        else
            echo -e "${YELLOW}Warning: .env.example not found, creating minimal .env${NC}"
            cat > "$ENV_FILE" << EOF
# AI Gateway Environment Configuration
# Generated on $(date)

# Server Ports
GATEWAY_PORT=8000
WEB_PORT=3000
REDIS_PORT=6379

# API Keys (Configure these!)
OPENAI_API_KEY=your-openai-api-key-here
ANTHROPIC_API_KEY=your-anthropic-api-key-here
AZURE_OPENAI_API_KEY=
AZURE_OPENAI_ENDPOINT=

# Monitoring (optional)
PROMETHEUS_PORT=9090
GRAFANA_PORT=3001
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin123
EOF
            echo -e "${GREEN}[OK] Created minimal .env file${NC}"
        fi
    else
        echo -e "${GREEN}[OK] .env file exists${NC}"
    fi
}

# Create necessary directories
create_directories() {
    echo -e "${YELLOW}[3/5] Creating necessary directories...${NC}"

    mkdir -p "${PROJECT_DIR}/data"
    mkdir -p "${PROJECT_DIR}/logs"

    echo -e "${GREEN}[OK] Directories created${NC}"
}

# Pull images
pull_images() {
    echo -e "${YELLOW}[4/5] Pulling Docker images...${NC}"

    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi

    cd "$PROJECT_DIR"

    if [ "$WITH_MONITORING" = true ]; then
        $compose_cmd --profile monitoring pull
    else
        $compose_cmd pull
    fi

    echo -e "${GREEN}[OK] Images pulled${NC}"
}

# Start services
start_services() {
    echo -e "${YELLOW}[5/5] Starting AI Gateway services...${NC}"

    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi

    cd "$PROJECT_DIR"

    if [ "$WITH_MONITORING" = true ]; then
        echo -e "${BLUE}Starting with monitoring stack...${NC}"
        $compose_cmd --profile monitoring up -d --build
    else
        echo -e "${BLUE}Starting basic services...${NC}"
        $compose_cmd up -d --build
    fi

    echo ""
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}   AI Gateway Started Successfully!${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo ""
    echo -e "Services:"
    echo -e "  ${BLUE}Gateway API:${NC}    http://localhost:${GATEWAY_PORT:-8000}"
    echo -e "  ${BLUE}Web Dashboard:${NC}  http://localhost:${WEB_PORT:-3000}"
    echo -e "  ${BLUE}Redis Stack:${NC}    localhost:${REDIS_PORT:-6379}"

    if [ "$WITH_MONITORING" = true ]; then
        echo ""
        echo -e "Monitoring:"
        echo -e "  ${BLUE}Prometheus:${NC}     http://localhost:${PROMETHEUS_PORT:-9090}"
        echo -e "  ${BLUE}Grafana:${NC}        http://localhost:${GRAFANA_PORT:-3001}"
        echo -e "    ${YELLOW}User:${NC} ${GRAFANA_ADMIN_USER:-admin}"
        echo -e "    ${YELLOW}Pass:${NC} ${GRAFANA_ADMIN_PASSWORD:-admin123}"
    fi

    echo ""
    echo -e "Commands:"
    echo -e "  ${YELLOW}View logs:${NC}     $0 --logs"
    echo -e "  ${YELLOW}Stop services:${NC} $0 --stop"
    echo ""
}

# Stop services
stop_services() {
    echo -e "${YELLOW}Stopping AI Gateway services...${NC}"

    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi

    cd "$PROJECT_DIR"
    $compose_cmd --profile monitoring down

    echo -e "${GREEN}[OK] Services stopped${NC}"
}

# Show logs
show_logs() {
    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi

    cd "$PROJECT_DIR"
    $compose_cmd logs -f
}

# Main function
main() {
    print_banner

    local action="start"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -m|--monitoring)
                WITH_MONITORING=true
                shift
                ;;
            -s|--stop)
                action="stop"
                shift
                ;;
            -r|--restart)
                action="restart"
                shift
                ;;
            -l|--logs)
                action="logs"
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

    case $action in
        start)
            check_docker
            check_env_file
            create_directories
            pull_images
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            stop_services
            sleep 2
            check_docker
            start_services
            ;;
        logs)
            show_logs
            ;;
    esac
}

# Run main function
main "$@"

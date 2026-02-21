#!/bin/bash

# ============================================
# AI Gateway - Quick Start Script (Mac/Linux)
# 一键启动 - 无需手动配置
# ============================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Print banner
print_banner() {
    clear
    echo -e "${CYAN}"
    echo "  ========================================     "
    echo "      AI Gateway - Quick Start                "
    echo "  ========================================     "
    echo -e "${NC}"
}

# Check Docker
check_docker() {
    echo -e "${YELLOW}[Step 1/5]${NC} Checking Docker..."

    if ! command -v docker &> /dev/null; then
        echo -e "${RED}[ERROR] Docker is not installed!${NC}"
        echo ""
        echo "Please install Docker:"
        echo "  Mac:     https://docs.docker.com/docker-for-mac/install/"
        echo "  Linux:   https://docs.docker.com/engine/install/"
        echo "  Windows: https://docs.docker.com/docker-for-windows/install/"
        echo ""
        exit 1
    fi

    if ! docker info &> /dev/null; then
        echo -e "${RED}[ERROR] Docker is not running!${NC}"
        echo "Please start Docker and try again."
        echo ""
        exit 1
    fi

    echo -e "${GREEN}[OK]${NC} Docker is ready"
}

# Setup environment
setup_env() {
    echo ""
    echo -e "${YELLOW}[Step 2/5]${NC} Setting up environment..."

    ENV_FILE="${PROJECT_DIR}/.env"

    if [ ! -f "$ENV_FILE" ]; then
        if [ -f "${PROJECT_DIR}/.env.example" ]; then
            cp "${PROJECT_DIR}/.env.example" "$ENV_FILE"
            echo -e "${GREEN}[OK]${NC} Created .env file from template"
        else
            cat > "$ENV_FILE" << EOF
# AI Gateway Configuration
# Generated on $(date)

# Server Ports
GATEWAY_PORT=8000
WEB_PORT=3000
REDIS_PORT=6379

# API Keys - Please configure these!
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
AZURE_OPENAI_API_KEY=
AZURE_OPENAI_ENDPOINT=

# Monitoring (Optional)
PROMETHEUS_PORT=9090
GRAFANA_PORT=3001
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin123
EOF
            echo -e "${GREEN}[OK]${NC} Created default .env file"
        fi
    else
        echo -e "${GREEN}[OK]${NC} .env file already exists"
    fi
}

# Check API keys
check_api_keys() {
    echo ""
    echo -e "${YELLOW}[Step 3/5]${NC} Checking API keys..."

    ENV_FILE="${PROJECT_DIR}/.env"
    HAS_KEY=false

    if grep -q "OPENAI_API_KEY=sk-" "$ENV_FILE" 2>/dev/null; then
        echo -e "${GREEN}[OK]${NC} OpenAI API key configured"
        HAS_KEY=true
    fi

    if grep -q "ANTHROPIC_API_KEY=sk-ant-" "$ENV_FILE" 2>/dev/null; then
        echo -e "${GREEN}[OK]${NC} Anthropic API key configured"
        HAS_KEY=true
    fi

    if grep -q "VOLCANO_API_KEY=.\+" "$ENV_FILE" 2>/dev/null && ! grep -q "VOLCANO_API_KEY=your-volcano" "$ENV_FILE" 2>/dev/null; then
        echo -e "${GREEN}[OK]${NC} Volcano Ark API key configured"
        HAS_KEY=true
    fi

    if [ "$HAS_KEY" = false ]; then
        echo -e "${YELLOW}[WARNING]${NC} No API keys configured!"
        echo ""
        echo "Please edit .env file and add your API keys:"
        echo "  ${CYAN}$ENV_FILE${NC}"
        echo ""
        echo "Get your keys from:"
        echo "  - OpenAI:        https://platform.openai.com/api-keys"
        echo "  - Anthropic:     https://console.anthropic.com/settings/keys"
        echo "  - Volcano Ark:   https://console.volcengine.com/ark"
        echo ""

        read -p "Continue without API keys? (y/N) " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# Pull images
pull_images() {
    echo ""
    echo -e "${YELLOW}[Step 4/5]${NC} Pulling Docker images..."

    cd "$PROJECT_DIR"

    if docker compose version &> /dev/null; then
        docker compose pull
    elif command -v docker-compose &> /dev/null; then
        docker-compose pull
    else
        echo -e "${RED}[ERROR] Docker Compose not found!${NC}"
        exit 1
    fi

    echo -e "${GREEN}[OK]${NC} Images pulled"
}

# Start services
start_services() {
    echo ""
    echo -e "${YELLOW}[Step 5/5]${NC} Starting services..."

    cd "$PROJECT_DIR"

    if docker compose version &> /dev/null; then
        docker compose up -d --build
    elif command -v docker-compose &> /dev/null; then
        docker-compose up -d --build
    fi

    echo ""
    echo -e "${CYAN}Waiting for services to start...${NC}"
    sleep 5

    # Check health
    if curl -f http://localhost:8000/health &> /dev/null; then
        echo -e "${GREEN}[OK]${NC} Gateway is healthy"
    else
        echo -e "${YELLOW}[WARNING]${NC} Gateway may still be starting..."
    fi
}

# Print success message
print_success() {
    clear
    echo -e "${GREEN}"
    echo "  ========================================     "
    echo "      AI Gateway Started Successfully!        "
    echo "  ========================================     "
    echo -e "${NC}"
    echo ""
    echo -e "${CYAN}Access Points:${NC}"
    echo ""
    echo -e "  ${BLUE}Gateway API:${NC}    http://localhost:8000"
    echo -e "  ${BLUE}Web Dashboard:${NC}  http://localhost:3000"
    echo -e "  ${BLUE}Health Check:${NC}   http://localhost:8000/health"
    echo ""
    echo -e "${CYAN}Quick Start Guide:${NC}"
    echo ""
    echo "  1. Open http://localhost:3000 in your browser"
    echo "  2. Configure your API keys in Settings"
    echo "  3. Start making API requests!"
    echo ""
    echo -e "${CYAN}Management Commands:${NC}"
    echo ""
    echo "  Stop:    ./scripts/start-gateway.sh --stop"
    echo "  Logs:    ./scripts/start-gateway.sh --logs"
    echo "  Restart: ./scripts/start-gateway.sh --restart"
    echo ""
    echo "  ========================================     "
    echo ""
}

# Open browser
open_browser() {
    if command -v open &> /dev/null; then
        # macOS
        open http://localhost:3000
    elif command -v xdg-open &> /dev/null; then
        # Linux
        xdg-open http://localhost:3000
    fi
}

# Main
main() {
    print_banner
    check_docker
    setup_env
    check_api_keys
    pull_images
    start_services
    print_success

    # Ask to open browser
    read -p "Open web dashboard in browser? (Y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        open_browser
    fi
}

# Run
main

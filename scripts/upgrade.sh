#!/bin/bash

# ============================================
# AI Gateway - Upgrade Script
# Handles version upgrades with data migration
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
BACKUP_DIR="${PROJECT_DIR}/backups"

# Current version file
VERSION_FILE="${PROJECT_DIR}/VERSION"

# Print banner
echo -e "${BLUE}"
echo "AI Gateway - Upgrade Script"
echo -e "${NC}"

# Get current version
get_current_version() {
    if [ -f "$VERSION_FILE" ]; then
        cat "$VERSION_FILE"
    else
        echo "unknown"
    fi
}

# Create backup
create_backup() {
    echo -e "${YELLOW}Creating backup...${NC}"

    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="${BACKUP_DIR}/backup_${timestamp}"

    mkdir -p "$backup_path"

    # Backup data directory
    if [ -d "${PROJECT_DIR}/data" ]; then
        echo "  - Backing up data directory..."
        cp -r "${PROJECT_DIR}/data" "${backup_path}/"
    fi

    # Backup config
    if [ -f "${PROJECT_DIR}/.env" ]; then
        echo "  - Backing up environment file..."
        cp "${PROJECT_DIR}/.env" "${backup_path}/"
    fi

    if [ -d "${PROJECT_DIR}/configs" ]; then
        echo "  - Backing up configs..."
        cp -r "${PROJECT_DIR}/configs" "${backup_path}/"
    fi

    # Backup version info
    get_current_version > "${backup_path}/version.txt"

    echo -e "${GREEN}[OK] Backup created at: ${backup_path}${NC}"
    echo "$backup_path"
}

# Stop services gracefully
stop_services() {
    echo -e "${YELLOW}Stopping services...${NC}"

    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi

    cd "$PROJECT_DIR"
    $compose_cmd --profile monitoring down --remove-orphans

    echo -e "${GREEN}[OK] Services stopped${NC}"
}

# Pull latest images/code
pull_updates() {
    echo -e "${YELLOW}Pulling latest updates...${NC}"

    # Check if git repo
    if [ -d "${PROJECT_DIR}/.git" ]; then
        echo "  - Pulling from git..."
        cd "$PROJECT_DIR"
        git fetch --all
        git pull
    fi

    # Pull Docker images
    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi

    cd "$PROJECT_DIR"
    $compose_cmd --profile monitoring pull

    echo -e "${GREEN}[OK] Updates pulled${NC}"
}

# Migrate data if needed
migrate_data() {
    local old_version="$1"
    local new_version="$2"

    echo -e "${YELLOW}Checking for data migrations...${NC}"

    # Add version-specific migrations here
    # Example:
    # if [ "$old_version" = "1.0.0" ] && [ "$new_version" != "1.0.0" ]; then
    #     echo "  - Running migration from 1.0.0 to ${new_version}"
    #     # migration commands
    # fi

    echo -e "${GREEN}[OK] No migrations needed${NC}"
}

# Start services
start_services() {
    echo -e "${YELLOW}Starting services...${NC}"

    local compose_cmd="docker-compose"
    if docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi

    cd "$PROJECT_DIR"

    # Check if monitoring was enabled before
    if [ -f "${PROJECT_DIR}/.monitoring_enabled" ]; then
        $compose_cmd --profile monitoring up -d --build
    else
        $compose_cmd up -d --build
    fi

    echo -e "${GREEN}[OK] Services started${NC}"
}

# Verify upgrade
verify_upgrade() {
    echo -e "${YELLOW}Verifying upgrade...${NC}"

    sleep 5

    # Check gateway health
    if curl -sf http://localhost:8000/health > /dev/null; then
        echo -e "${GREEN}[OK] Gateway is healthy${NC}"
    else
        echo -e "${RED}[ERROR] Gateway health check failed${NC}"
        return 1
    fi

    # Check Redis
    if docker exec ai-gateway-redis redis-cli ping | grep -q "PONG"; then
        echo -e "${GREEN}[OK] Redis is healthy${NC}"
    else
        echo -e "${RED}[ERROR] Redis health check failed${NC}"
        return 1
    fi

    echo -e "${GREEN}Upgrade completed successfully!${NC}"
}

# Rollback function
rollback() {
    local backup_path="$1"

    echo -e "${RED}Rolling back to backup: ${backup_path}${NC}"

    # Stop services
    stop_services

    # Restore data
    if [ -d "${backup_path}/data" ]; then
        rm -rf "${PROJECT_DIR}/data"
        cp -r "${backup_path}/data" "${PROJECT_DIR}/"
    fi

    # Restore config
    if [ -f "${backup_path}/.env" ]; then
        cp "${backup_path}/.env" "${PROJECT_DIR}/"
    fi

    # Start services
    start_services

    echo -e "${YELLOW}Rollback completed. Please check your services.${NC}"
}

# Print usage
print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --backup-only       Create backup only, don't upgrade"
    echo "  --rollback PATH     Rollback to specified backup path"
    echo "  --skip-backup       Skip backup creation (not recommended)"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                  # Full upgrade with backup"
    echo "  $0 --backup-only    # Create backup only"
    echo "  $0 --rollback ./backups/backup_20240101_120000"
    echo ""
}

# Main function
main() {
    local backup_only=false
    local skip_backup=false
    local rollback_path=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --backup-only)
                backup_only=true
                shift
                ;;
            --skip-backup)
                skip_backup=true
                shift
                ;;
            --rollback)
                rollback_path="$2"
                shift 2
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

    # Handle rollback
    if [ -n "$rollback_path" ]; then
        rollback "$rollback_path"
        exit 0
    fi

    # Get versions
    local old_version=$(get_current_version)
    echo -e "Current version: ${BLUE}${old_version}${NC}"

    # Create backup
    local backup_path=""
    if [ "$skip_backup" = false ]; then
        backup_path=$(create_backup)
    fi

    # If backup only, exit
    if [ "$backup_only" = true ]; then
        echo -e "${GREEN}Backup completed.${NC}"
        exit 0
    fi

    # Perform upgrade
    echo ""
    echo -e "${BLUE}Starting upgrade process...${NC}"
    echo ""

    stop_services
    pull_updates
    migrate_data "$old_version" "latest"
    start_services

    # Verify
    if verify_upgrade; then
        echo ""
        echo -e "${GREEN}============================================${NC}"
        echo -e "${GREEN}   Upgrade Completed Successfully!${NC}"
        echo -e "${GREEN}============================================${NC}"

        if [ -n "$backup_path" ]; then
            echo -e "Backup saved to: ${backup_path}"
        fi
    else
        echo ""
        echo -e "${RED}Upgrade verification failed!${NC}"
        if [ -n "$backup_path" ]; then
            echo -e "${YELLOW}To rollback, run: $0 --rollback ${backup_path}${NC}"
        fi
        exit 1
    fi
}

# Run main function
main "$@"

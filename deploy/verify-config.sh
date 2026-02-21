#!/bin/bash

# ============================================
# AI Gateway - Deployment Verification Script
# 验证部署配置是否正确
# ============================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   AI Gateway - Deployment Verification${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

ERRORS=0
WARNINGS=0

# Check function
check() {
    local name="$1"
    local result="$2"

    if [ "$result" = "OK" ]; then
        echo -e "${GREEN}[✓]${NC} $name"
    elif [ "$result" = "WARN" ]; then
        echo -e "${YELLOW}[!]${NC} $name"
        WARNINGS=$((WARNINGS + 1))
    else
        echo -e "${RED}[✗]${NC} $name"
        ERRORS=$((ERRORS + 1))
    fi
}

# 1. Check required files
echo -e "${YELLOW}1. Checking required files...${NC}"
echo ""

[ -f "$PROJECT_DIR/docker-compose.yml" ] && check "docker-compose.yml" "OK" || check "docker-compose.yml" "FAIL"
[ -f "$PROJECT_DIR/Dockerfile" ] && check "Dockerfile" "OK" || check "Dockerfile" "FAIL"
[ -f "$PROJECT_DIR/.env.example" ] && check ".env.example" "OK" || check ".env.example" "FAIL"
[ -f "$PROJECT_DIR/scripts/start-gateway.sh" ] && check "start-gateway.sh" "OK" || check "start-gateway.sh" "FAIL"
[ -f "$PROJECT_DIR/scripts/start-gateway.bat" ] && check "start-gateway.bat" "OK" || check "start-gateway.bat" "FAIL"
[ -f "$PROJECT_DIR/DEPLOYMENT.md" ] && check "DEPLOYMENT.md" "OK" || check "DEPLOYMENT.md" "FAIL"

echo ""

# 2. Check deploy directory files
echo -e "${YELLOW}2. Checking deploy directory...${NC}"
echo ""

[ -f "$SCRIPT_DIR/quick-start.sh" ] && check "quick-start.sh" "OK" || check "quick-start.sh" "FAIL"
[ -f "$SCRIPT_DIR/quick-start.bat" ] && check "quick-start.bat" "OK" || check "quick-start.bat" "FAIL"
[ -f "$SCRIPT_DIR/docker-compose.prod.yml" ] && check "docker-compose.prod.yml" "OK" || check "docker-compose.prod.yml" "FAIL"
[ -f "$SCRIPT_DIR/README.md" ] && check "deploy/README.md" "OK" || check "deploy/README.md" "FAIL"
[ -f "$SCRIPT_DIR/PRODUCTION-CHECKLIST.md" ] && check "PRODUCTION-CHECKLIST.md" "OK" || check "PRODUCTION-CHECKLIST.md" "FAIL"
[ -f "$SCRIPT_DIR/ARCHITECTURE.md" ] && check "ARCHITECTURE.md" "OK" || check "ARCHITECTURE.md" "FAIL"

echo ""

# 3. Check monitoring configuration
echo -e "${YELLOW}3. Checking monitoring configuration...${NC}"
echo ""

[ -f "$PROJECT_DIR/monitoring/prometheus.yml" ] && check "prometheus.yml" "OK" || check "prometheus.yml" "WARN"
[ -f "$PROJECT_DIR/monitoring/alert_rules.yml" ] && check "alert_rules.yml" "OK" || check "alert_rules.yml" "WARN"
[ -f "$PROJECT_DIR/monitoring/alertmanager.yml" ] && check "alertmanager.yml" "OK" || check "alertmanager.yml" "WARN"

echo ""

# 4. Check environment file
echo -e "${YELLOW}4. Checking environment configuration...${NC}"
echo ""

if [ -f "$PROJECT_DIR/.env" ]; then
    check ".env file exists" "OK"

    # Check API keys
    if grep -q "OPENAI_API_KEY=sk-" "$PROJECT_DIR/.env" 2>/dev/null; then
        check "OpenAI API key configured" "OK"
    else
        check "OpenAI API key not configured" "WARN"
    fi

    if grep -q "ANTHROPIC_API_KEY=sk-ant-" "$PROJECT_DIR/.env" 2>/dev/null; then
        check "Anthropic API key configured" "OK"
    else
        check "Anthropic API key not configured" "WARN"
    fi
else
    check ".env file (will be created on first run)" "WARN"
fi

echo ""

# 5. Check Docker
echo -e "${YELLOW}5. Checking Docker...${NC}"
echo ""

if command -v docker &> /dev/null; then
    check "Docker installed" "OK"

    if docker info &> /dev/null; then
        check "Docker running" "OK"
    else
        check "Docker not running" "WARN"
    fi

    if docker compose version &> /dev/null || command -v docker-compose &> /dev/null; then
        check "Docker Compose installed" "OK"
    else
        check "Docker Compose not installed" "FAIL"
    fi
else
    check "Docker not installed" "FAIL"
fi

echo ""

# 6. Check script permissions
echo -e "${YELLOW}6. Checking script permissions...${NC}"
echo ""

[ -x "$PROJECT_DIR/scripts/start-gateway.sh" ] && check "start-gateway.sh executable" "OK" || check "start-gateway.sh not executable" "WARN"
[ -x "$SCRIPT_DIR/quick-start.sh" ] && check "quick-start.sh executable" "OK" || check "quick-start.sh not executable" "WARN"

echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Verification Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    echo ""
    echo "You can now deploy with:"
    echo "  ./deploy/quick-start.sh"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}! Checks passed with warnings${NC}"
    echo ""
    echo "Warnings: $WARNINGS"
    echo ""
    echo "You can still deploy, but review the warnings above."
    exit 0
else
    echo -e "${RED}✗ Some checks failed${NC}"
    echo ""
    echo "Errors: $ERRORS"
    echo "Warnings: $WARNINGS"
    echo ""
    echo "Please fix the errors before deploying."
    exit 1
fi

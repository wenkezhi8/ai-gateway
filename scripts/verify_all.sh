#!/bin/bash

# AI Gateway 全功能验证脚本
# 每次修改后运行此脚本，确保所有功能正常

GATEWAY_URL="http://localhost:8566"
FAILED=0
PASSED=0

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

test_api() {
    local name="$1"
    local url="$2"
    local expected_code="${3:-200}"
    local check_json="${4:-true}"
    
    response=$(/usr/bin/curl -s -w "\n%{http_code}" "$url")
    http_code=$(echo "$response" | tail -1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" != "$expected_code" ]; then
        echo -e "${RED}✗${NC} $name - HTTP $http_code (expected $expected_code)"
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    if [ "$check_json" = "true" ]; then
        success=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)
        if [ "$success" != "True" ]; then
            echo -e "${RED}✗${NC} $name - success != true"
            FAILED=$((FAILED + 1))
            return 1
        fi
    fi
    
    echo -e "${GREEN}✓${NC} $name"
    PASSED=$((PASSED + 1))
    return 0
}

test_page() {
    local name="$1"
    local url="$2"
    
    http_code=$(/usr/bin/curl -s -o /dev/null -w "%{http_code}" "$url")
    
    if [ "$http_code" != "200" ]; then
        echo -e "${RED}✗${NC} $name - HTTP $http_code"
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    echo -e "${GREEN}✓${NC} $name"
    PASSED=$((PASSED + 1))
    return 0
}

test_chat_api() {
    local name="$1"
    
    response=$(/usr/bin/curl -s -X POST "$GATEWAY_URL/api/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"hi"}],"max_tokens":5}')
    
    has_choices=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print('yes' if 'choices' in d else 'no')" 2>/dev/null)
    
    if [ "$has_choices" != "yes" ]; then
        echo -e "${RED}✗${NC} $name - No choices in response"
        FAILED=$((FAILED + 1))
        return 1
    fi
    
    echo -e "${GREEN}✓${NC} $name"
    PASSED=$((PASSED + 1))
    return 0
}

echo "=========================================="
echo "    AI Gateway 全功能验证"
echo "    $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="
echo ""

echo "【1. 健康检查】"
test_api "Health" "$GATEWAY_URL/health" "200" "false"
echo ""

echo "【2. 核心API】"
test_api "Models" "$GATEWAY_URL/api/v1/models"
test_api "Config Providers" "$GATEWAY_URL/api/v1/config/providers"
echo ""

echo "【3. 管理API】"
test_api "Accounts" "$GATEWAY_URL/api/admin/accounts"
test_api "Routing" "$GATEWAY_URL/api/admin/routing"
test_api "Cache Stats" "$GATEWAY_URL/api/admin/cache/stats"
test_api "Cache Health" "$GATEWAY_URL/api/admin/cache/health"
echo ""

echo "【4. Dashboard API】"
test_api "Stats" "$GATEWAY_URL/api/admin/dashboard/stats"
test_api "Realtime" "$GATEWAY_URL/api/admin/dashboard/realtime"
test_api "Requests" "$GATEWAY_URL/api/admin/dashboard/requests"
test_api "System" "$GATEWAY_URL/api/admin/dashboard/system"
echo ""

echo "【5. 请求趋势数据验证】"
requests_data=$(/usr/bin/curl -s "$GATEWAY_URL/api/admin/dashboard/requests")
points_count=$(echo "$requests_data" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('data',[])))" 2>/dev/null)
if [ "$points_count" -ge 24 ]; then
    echo -e "${GREEN}✓${NC} Request Trends - $points_count data points"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} Request Trends - Only $points_count points (need 24+)"
    FAILED=$((FAILED + 1))
fi
echo ""

echo "【6. 服务商分布数据验证】"
system_data=$(/usr/bin/curl -s "$GATEWAY_URL/api/admin/dashboard/system")
has_distribution=$(echo "$system_data" | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print('yes' if 'distribution' in d else 'no')" 2>/dev/null)
if [ "$has_distribution" = "yes" ]; then
    echo -e "${GREEN}✓${NC} Provider Distribution exists"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} Provider Distribution missing"
    FAILED=$((FAILED + 1))
fi
echo ""

echo "【7. Chat API】"
test_chat_api "Chat Completions"
echo ""

echo "【8. 前端页面】"
test_page "Dashboard" "$GATEWAY_URL/dashboard"
test_page "Routing" "$GATEWAY_URL/routing"
test_page "Cache" "$GATEWAY_URL/cache"
test_page "Alerts" "$GATEWAY_URL/alerts"
test_page "Model Management" "$GATEWAY_URL/model-management"
test_page "Providers Accounts" "$GATEWAY_URL/providers-accounts"
test_page "Limit Management" "$GATEWAY_URL/limit-management"
test_page "Chat" "$GATEWAY_URL/chat"
test_page "Settings" "$GATEWAY_URL/settings"
test_page "Login" "$GATEWAY_URL/login"
test_page "Public Chat" "$GATEWAY_URL/p/chat"
echo ""

echo "【9. Swagger文档】"
test_page "Swagger UI" "$GATEWAY_URL/swagger/index.html"
echo ""

echo "【10. Prometheus指标（仅本机）】"
test_page "Metrics (localhost only)" "http://127.0.0.1:9090/metrics"
echo ""

echo "=========================================="
echo "    验证结果"
echo "=========================================="
echo -e "${GREEN}通过: $PASSED${NC}"
echo -e "${RED}失败: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}所有功能验证通过！${NC}"
    exit 0
else
    echo -e "${RED}有 $FAILED 项验证失败，请检查！${NC}"
    exit 1
fi

#!/bin/bash
# 开发完成后重启脚本（严格模式）
# 使用方法: ./scripts/dev-restart.sh

set -euo pipefail

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

if [ ! -f "configs/config.json" ]; then
    cp "configs/config.example.json" "configs/config.json"
    echo "⚠️  检测到缺少 configs/config.json，已自动从示例文件创建"
fi

SERVER_PORT=$(python3 -c 'import json;print(str((json.load(open("configs/config.json")).get("server") or {}).get("port") or "8566"))' 2>/dev/null || true)
if [ -z "$SERVER_PORT" ]; then
    SERVER_PORT="8566"
fi

echo "🔨 构建前端..."
cd web
npm run build

echo ""
echo "🔨 构建后端二进制..."
cd "$PROJECT_DIR"
go build -o bin/gateway ./cmd/gateway

echo ""
echo "🔍 检查 Redis Stack (6379)..."
if ! lsof -ti:6379 >/dev/null 2>&1; then
    echo "⚠️  Redis Stack 未运行，尝试自动启动 Docker 容器..."
    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        docker rm -f redis-stack >/dev/null 2>&1 || true
        docker run -d --name redis-stack -p 6379:6379 -p 8001:8001 redis/redis-stack-server:7.2.0-v18 >/dev/null
        sleep 2
    fi
fi

if ! lsof -ti:6379 >/dev/null 2>&1; then
    echo "❌ Redis Stack 启动失败，无法继续（向量缓存依赖 RediSearch/RedisJSON）"
    echo "   可手动执行: docker run -d --name redis-stack -p 6379:6379 -p 8001:8001 redis/redis-stack-server:7.2.0-v18"
    exit 1
fi

if ! redis-cli -p 6379 FT._LIST >/dev/null 2>&1; then
    echo "❌ 当前 6379 不是 Redis Stack（缺少 FT 命令），请切换到 redis-stack-server"
    echo "   建议执行: docker rm -f redis-stack >/dev/null 2>&1 || true"
    echo "   然后执行: docker run -d --name redis-stack -p 6379:6379 -p 8001:8001 redis/redis-stack-server:7.2.0-v18"
    exit 1
fi

echo "✓ Redis Stack 已就绪 (6379)"

echo ""
echo "🛑 停止所有旧服务（包括僵尸进程）..."
# 停止所有 gateway 相关进程
pkill -9 -f "go run ./cmd/gateway" 2>/dev/null || true
pkill -9 -f "clawdbot-gateway" 2>/dev/null || true
pkill -9 -f "openclaw-gateway" 2>/dev/null || true
pkill -9 -f "/bin/gateway" 2>/dev/null || true
sleep 2

# 再次确认没有残留
REMAINING=$( (pgrep -f "go run ./cmd/gateway|clawdbot-gateway|openclaw-gateway|/bin/gateway" 2>/dev/null || true) | wc -l | tr -d ' ' )
if [ "$REMAINING" -gt 0 ]; then
    echo "⚠️  发现 $REMAINING 个残留进程，强制清理..."
    pkill -9 -f "/bin/gateway" 2>/dev/null || true
    sleep 1
fi

if lsof -iTCP:"$SERVER_PORT" -sTCP:LISTEN -ti >/dev/null 2>&1; then
	echo "❌ 端口 $SERVER_PORT 仍被占用，停止失败"
	lsof -i :"$SERVER_PORT" -n -P
	exit 1
fi

echo "✓ 旧服务已停止"

echo ""
echo "🚀 启动新服务（二进制）..."
cd "$PROJECT_DIR"
nohup ./bin/gateway > /tmp/ai-gateway.log 2>&1 &
GATEWAY_PID=$!

echo ""
echo "⏳ 等待服务启动 (PID: $GATEWAY_PID)..."

HEALTH=""
MAX_WAIT_SECONDS=30
for ((i=1; i<=MAX_WAIT_SECONDS; i++)); do
    HEALTH=$(curl -s --max-time 2 "http://localhost:$SERVER_PORT/health" || true)
    if echo "$HEALTH" | grep -q "healthy"; then
        break
    fi
    sleep 1
done

echo ""
echo "🔍 检查服务状态..."
if echo "$HEALTH" | grep -q "healthy"; then
    echo "✅ 服务启动成功"

    if ! lsof -iTCP:"$SERVER_PORT" -sTCP:LISTEN -ti >/dev/null 2>&1; then
        echo "❌ 未检测到 $SERVER_PORT 监听"
        exit 1
    fi

    TRACE_HTML=$(curl -s "http://localhost:$SERVER_PORT/trace")
    TRACE_ASSET=$(echo "$TRACE_HTML" | grep -oE '/assets/index-[A-Za-z0-9_-]+\.js' | head -1 || true)
    if [ -z "$TRACE_ASSET" ]; then
        echo "❌ /trace 未找到前端资产引用"
        exit 1
    fi

    TRACE_ASSET_BODY=$(curl -s "http://localhost:$SERVER_PORT$TRACE_ASSET")
    TRACE_ASSET_HEAD="${TRACE_ASSET_BODY:0:20}"
    if echo "$TRACE_ASSET_HEAD" | grep -qi "<!doctype html"; then
        echo "❌ /trace 资产返回了 HTML，疑似资源不一致"
        echo "   资产路径: $TRACE_ASSET"
        exit 1
    fi

    CACHE_BACKEND_LINE=$(grep -E "Cache backend is memory|Connected to Redis" /tmp/ai-gateway.log | tail -1 || true)
    if [ -n "$CACHE_BACKEND_LINE" ]; then
        echo ""
        echo "🧠 缓存后端:"
        echo "   $CACHE_BACKEND_LINE"
    fi
    
    # 显示内存使用
    MEM=$(ps -p $GATEWAY_PID -o rss= 2>/dev/null | awk '{print int($1/1024) "MB"}')
    echo ""
    echo "📊 进程信息:"
    echo "   PID:  $GATEWAY_PID"
    echo "   内存: $MEM (物理内存)"
    
    echo ""
    echo "📌 访问地址:"
    echo "   - 首页:     http://localhost:$SERVER_PORT/"
    echo "   - 路由策略: http://localhost:$SERVER_PORT/routing"
    echo "   - 缓存管理: http://localhost:$SERVER_PORT/cache"
    echo "   - 请求链路: http://localhost:$SERVER_PORT/trace"
    echo ""
    echo "⚠️  如果页面显示异常，请强制刷新浏览器:"
    echo "   Mac:     Cmd + Shift + R"
    echo "   Windows: Ctrl + Shift + R"
else
    echo "❌ 服务启动失败，请检查日志:"
    echo "   tail -f /tmp/ai-gateway.log"
    exit 1
fi

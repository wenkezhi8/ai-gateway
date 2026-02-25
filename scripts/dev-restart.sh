#!/bin/bash
# 开发完成后重启脚本
# 使用方法: ./scripts/dev-restart.sh

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo "🔨 构建前端..."
cd web
npm run build

echo ""
echo "🔍 检查 Redis (6379)..."
if lsof -ti:6379 >/dev/null 2>&1; then
    echo "✓ Redis 端口可用 (6379)"
else
    echo "⚠️  Redis 未运行，尝试自动启动..."

    if command -v brew >/dev/null 2>&1; then
        brew services start redis >/dev/null 2>&1 || true
        sleep 2
    fi

    if ! lsof -ti:6379 >/dev/null 2>&1 && command -v redis-server >/dev/null 2>&1; then
        nohup redis-server > /tmp/redis.log 2>&1 &
        sleep 2
    fi

    if lsof -ti:6379 >/dev/null 2>&1; then
        echo "✓ Redis 启动成功 (6379)"
    else
        echo "❌ Redis 启动失败，缓存将降级为内存模式（重启丢失）"
        echo "   可手动执行: brew services start redis"
    fi
fi

echo ""
echo "🛑 停止所有旧服务（包括僵尸进程）..."
# 停止所有 gateway 相关进程
pkill -9 -f "go run ./cmd/gateway" 2>/dev/null || true
pkill -9 -f "ai-gateway" 2>/dev/null || true
pkill -9 -f "openclaw-gateway" 2>/dev/null || true
pkill -9 -f "/gateway" 2>/dev/null || true
sleep 2

# 再次确认没有残留
REMAINING=$(pgrep -f "gateway" 2>/dev/null | wc -l)
if [ "$REMAINING" -gt 0 ]; then
    echo "⚠️  发现 $REMAINING 个残留进程，强制清理..."
    pkill -9 gateway 2>/dev/null || true
    sleep 1
fi

echo "✓ 旧服务已停止"

echo ""
echo "🚀 启动新服务..."
cd "$PROJECT_DIR"
nohup go run ./cmd/gateway > /tmp/ai-gateway.log 2>&1 &
GATEWAY_PID=$!

echo ""
echo "⏳ 等待服务启动 (PID: $GATEWAY_PID)..."
sleep 4

echo ""
echo "🔍 检查服务状态..."
HEALTH=$(curl -s http://localhost:8566/health)
if echo "$HEALTH" | grep -q "healthy"; then
    echo "✅ 服务启动成功"

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
    echo "   - 首页:     http://localhost:8566/"
    echo "   - 路由策略: http://localhost:8566/routing"
    echo "   - 缓存管理: http://localhost:8566/cache"
    echo ""
    echo "⚠️  如果页面显示异常，请强制刷新浏览器:"
    echo "   Mac:     Cmd + Shift + R"
    echo "   Windows: Ctrl + Shift + R"
else
    echo "❌ 服务启动失败，请检查日志:"
    echo "   tail -f /tmp/ai-gateway.log"
    exit 1
fi

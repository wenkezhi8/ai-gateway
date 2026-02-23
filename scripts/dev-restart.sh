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

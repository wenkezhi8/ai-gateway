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
echo "🛑 停止旧服务..."
lsof -ti:8566 | xargs kill -9 2>/dev/null || true
sleep 1

echo ""
echo "🚀 启动新服务..."
cd "$PROJECT_DIR"
nohup go run ./cmd/gateway > /tmp/ai-gateway.log 2>&1 &

echo ""
echo "⏳ 等待服务启动..."
sleep 4

echo ""
echo "🔍 检查服务状态..."
HEALTH=$(curl -s http://localhost:8566/health)
if echo "$HEALTH" | grep -q "healthy"; then
    echo "✅ 服务启动成功"
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

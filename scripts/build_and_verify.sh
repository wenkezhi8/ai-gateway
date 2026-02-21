#!/bin/bash

echo "=== AI Gateway 一键构建验证 ==="
echo ""

cd /Users/openclaw/ai-gateway

# 1. 编译后端
echo "1. 编译后端..."
go build -o bin/gateway ./cmd/gateway
if [ $? -ne 0 ]; then
    echo "❌ 后端编译失败"
    exit 1
fi
echo "✅ 后端编译成功"
echo ""

# 2. 构建前端
echo "2. 构建前端..."
cd web && npm run build > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ 前端构建失败"
    exit 1
fi
cd ..
echo "✅ 前端构建成功"
echo ""

# 3. 重启服务
echo "3. 重启服务..."
pkill -f "bin/gateway" 2>/dev/null
sleep 1
./bin/gateway > /tmp/gateway.log 2>&1 &
sleep 3
echo "✅ 服务已启动"
echo ""

# 4. 运行验证
echo "4. 运行功能验证..."
./scripts/verify_all.sh

exit $?

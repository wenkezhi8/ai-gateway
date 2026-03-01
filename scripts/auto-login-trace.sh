#!/bin/bash

# 完全自动登录并打开 Trace 页面

echo "=========================================="
echo "  自动登录并打开链路追踪页面"
echo "=========================================="
echo ""

# 1. 检查服务
if ! curl -s http://localhost:8566/health > /dev/null 2>&1; then
    echo "⚠️  服务未运行，正在启动..."
    cd /Users/openclaw/ai-gateway
    ./bin/gateway > /tmp/gateway.log 2>&1 &
    sleep 3
fi

echo "✅ 服务运行中"

# 2. 登录获取 token
echo ""
echo "正在自动登录..."

LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8566/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败"
    echo "$LOGIN_RESPONSE"
    exit 1
fi

echo "✅ 登录成功"
echo ""

# 3. 创建自动登录 HTML
AUTO_LOGIN_HTML="/Users/openclaw/ai-gateway/web/dist/auto-login.html"

cat > "$AUTO_LOGIN_HTML" << HTMLEOF
<!DOCTYPE html>
<html>
<head>
    <title>自动登录中...</title>
    <meta charset="UTF-8">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .container {
            text-align: center;
        }
        .spinner {
            border: 4px solid rgba(255,255,255,.3);
            border-radius: 50%;
            border-top: 4px solid white;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 20px auto;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
    </style>
    <script>
        // 设置 token 到 localStorage
        localStorage.setItem('token', '$TOKEN');
        
        // 设置用户信息
        localStorage.setItem('user', JSON.stringify({
            id: "1",
            username: "admin",
            role: "admin"
        }));
        
        // 设置 token 过期时间 (24小时后)
        const expiresAt = Date.now() + (24 * 60 * 60 * 1000);
        localStorage.setItem('token_expires_at', expiresAt.toString());
        
        // 自动跳转到 trace 页面
        setTimeout(function() {
            window.location.href = '/trace';
        }, 1000);
    </script>
</head>
<body>
    <div class="container">
        <h2>🔐 自动登录成功</h2>
        <div class="spinner"></div>
        <p>正在跳转到链路追踪页面...</p>
    </div>
</body>
</html>
HTMLEOF

echo "✅ 已创建自动登录页面"
echo ""

# 4. 打开浏览器
echo "🌐 正在打开浏览器..."
open "http://localhost:8566/auto-login.html"

echo ""
echo "✅ 完成！浏览器将自动登录并跳转到链路追踪页面"

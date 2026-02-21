# AI Gateway 常见问题（FAQ）

> 收录了用户最常遇到的问题和解决方案

---

## 目录

- [安装部署](#安装部署)
- [配置相关](#配置相关)
- [API使用](#api使用)
- [服务商问题](#服务商问题)
- [性能优化](#性能优化)
- [错误排查](#错误排查)
- [安全相关](#安全相关)
- [计费与用量](#计费与用量)

---

## 安装部署

### Q1：Docker启动失败，提示端口被占用怎么办？

**问题**：
```
Error: bind: address already in use
```

**解决方案**：

**Mac/Linux**：
```bash
# 查看占用8080端口的进程
lsof -i :8080

# 结束占用的进程（替换PID）
kill -9 <PID>
```

**Windows**：
```cmd
# 查看占用端口的进程
netstat -ano | findstr :8080

# 结束进程
taskkill /F /PID <PID>
```

**或者修改端口**：
编辑 `docker-compose.yml`，将 `8080:8080` 改为 `8081:8080`

---

### Q2：启动后无法访问Web界面？

**排查步骤**：

1. **检查容器状态**：
```bash
docker-compose ps
```
确保所有容器状态为 `Up`

2. **检查日志**：
```bash
docker-compose logs gateway
```

3. **检查防火墙**：
```bash
# Mac
sudo pfctl -d  # 临时关闭

# Ubuntu
sudo ufw status
```

4. **尝试127.0.0.1**：
如果 `localhost` 不行，尝试 `http://127.0.0.1:8080`

---

### Q3：如何更新到最新版本？

```bash
# 停止服务
docker-compose down

# 拉取最新代码
git pull origin main

# 重新构建并启动
docker-compose build
docker-compose up -d
```

---

## 配置相关

### Q4：如何配置多个API Key？

在 `.env` 文件中添加：

```bash
# OpenAI
OPENAI_API_KEY=sk-xxxxx
OPENAI_API_KEY_2=sk-yyyyy  # 备用Key

# Claude
ANTHROPIC_API_KEY=sk-ant-xxxxx

# 火山引擎
VOLCENGINE_API_KEY=xxxxx
```

在 `configs/config.json` 中配置：

```json
{
  "providers": {
    "openai": {
      "api_keys": ["sk-xxxxx", "sk-yyyyy"],
      "strategy": "round-robin"
    }
  }
}
```

---

### Q5：如何启用认证功能？

1. **生成API Key**：
```bash
# 使用openssl生成随机key
openssl rand -hex 32
```

2. **配置认证**：
```json
{
  "auth": {
    "enabled": true,
    "api_keys": ["your-generated-key"]
  }
}
```

3. **客户端使用**：
```python
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-generated-key"
)
```

---

### Q6：如何配置速率限制？

```json
{
  "limiter": {
    "enabled": true,
    "requests_per_minute": 60,
    "tokens_per_minute": 40000,
    "strategy": "sliding_window"
  }
}
```

针对不同用户设置不同限额：
```json
{
  "user_limits": {
    "user-basic": {
      "requests_per_day": 100,
      "tokens_per_day": 10000
    },
    "user-pro": {
      "requests_per_day": 1000,
      "tokens_per_day": 100000
    }
  }
}
```

---

## API使用

### Q7：如何实现流式输出？

**Python示例**：
```python
stream = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "讲个故事"}],
    stream=True  # 关键参数
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

**JavaScript示例**：
```javascript
const stream = await client.chat.completions.create({
  model: 'gpt-4',
  messages: [{ role: 'user', content: '讲个故事' }],
  stream: true
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}
```

---

### Q8：如何处理超时？

**设置超时时间**：
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-key",
    timeout=60.0  # 60秒
)
```

**添加重试逻辑**：
```python
import time
from openai import OpenAI, APITimeoutError

def call_with_retry(client, messages, max_retries=3):
    for attempt in range(max_retries):
        try:
            return client.chat.completions.create(
                model="gpt-4",
                messages=messages
            )
        except APITimeoutError:
            if attempt < max_retries - 1:
                wait_time = 2 ** attempt  # 指数退避
                time.sleep(wait_time)
                continue
            raise
```

---

### Q9：返回的数据格式是什么？

**标准响应格式**：
```json
{
  "id": "chatcmpl-123456",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "回复内容"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

---

## 服务商问题

### Q10：OpenAI请求失败，提示API Key无效？

**检查清单**：

1. **Key格式正确**：
   - 应以 `sk-` 开头
   - 长度通常为51个字符

2. **Key有效**：
```bash
# 直接测试OpenAI API
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer sk-your-key"
```

3. **账户余额**：
   - 登录 https://platform.openai.com 检查余额

4. **网络问题**：
   - 如果在中国大陆，可能需要代理

---

### Q11：如何配置代理访问OpenAI？

**方式1：环境变量**：
```bash
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
```

**方式2：配置文件**：
```json
{
  "providers": {
    "openai": {
      "proxy": "http://127.0.0.1:7890"
    }
  }
}
```

---

### Q12：Claude和OpenAI的模型有什么区别？

| 特性 | Claude | OpenAI |
|------|--------|--------|
| 上下文长度 | 200K tokens | 128K tokens |
| 最佳用途 | 长文档分析、代码 | 通用对话、创意 |
| 价格 | 相对便宜 | 相对较贵 |
| 中文能力 | 优秀 | 优秀 |
| 代码能力 | 强 | 强 |

**选择建议**：
- 长文本处理 → Claude
- 快速对话 → GPT-3.5
- 复杂推理 → GPT-4 / Claude-3-Opus

---

## 性能优化

### Q13：如何提高响应速度？

**1. 启用缓存**：
```json
{
  "cache": {
    "enabled": true,
    "ttl": 3600,
    "max_size": 10000
  }
}
```

**2. 使用更快的模型**：
- GPT-3.5 比 GPT-4 快5-10倍
- Claude-3-Haiku 最快最便宜

**3. 减少输出长度**：
```python
response = client.chat.completions.create(
    model="gpt-4",
    messages=[...],
    max_tokens=100  # 限制输出长度
)
```

---

### Q14：如何处理高并发？

**1. 调整worker数量**：
```json
{
  "server": {
    "workers": 4
  }
}
```

**2. 启用连接池**：
```json
{
  "http_client": {
    "max_idle_conns": 100,
    "max_conns_per_host": 10
  }
}
```

**3. 使用Redis集群**：
```yaml
# docker-compose.yml
redis:
  image: redis:7-alpine
  command: redis-server --cluster-enabled yes
```

---

## 错误排查

### Q15：常见错误码及解决方案

| 错误码 | 含义 | 解决方案 |
|--------|------|---------|
| 400 | 请求参数错误 | 检查请求体格式 |
| 401 | 认证失败 | 检查API Key |
| 403 | 权限不足 | 检查账户权限 |
| 404 | 资源不存在 | 检查URL路径 |
| 429 | 速率限制 | 降低请求频率或提高限额 |
| 500 | 服务器错误 | 查看日志排查 |
| 502 | 上游服务错误 | 检查AI服务商状态 |
| 503 | 服务不可用 | 稍后重试 |

---

### Q16：如何查看详细日志？

**查看容器日志**：
```bash
# 查看所有日志
docker-compose logs -f

# 只看网关日志
docker-compose logs -f gateway

# 查看最近100行
docker-compose logs --tail=100 gateway
```

**调整日志级别**：
```json
{
  "log": {
    "level": "debug",  // debug, info, warn, error
    "output": "stdout"
  }
}
```

---

### Q17：请求返回空响应怎么办？

**可能原因**：

1. **模型未配置**：
```bash
# 检查可用模型
curl http://localhost:8080/v1/models
```

2. **服务商未启用**：
```json
{
  "providers": {
    "openai": {
      "enabled": true  // 确保为true
    }
  }
}
```

3. **路由配置错误**：
检查路由策略是否正确匹配模型

---

## 安全相关

### Q18：如何保护API Key安全？

**最佳实践**：

1. **不要硬编码**：
```python
# 错误做法
api_key = "sk-xxxxx"

# 正确做法
import os
api_key = os.environ.get("OPENAI_API_KEY")
```

2. **使用环境变量**：
```bash
export OPENAI_API_KEY=sk-xxxxx
```

3. **不要提交到Git**：
```gitignore
# .gitignore
.env
*.key
secrets/
```

4. **定期轮换Key**：
建议每3个月更换一次API Key

---

### Q19：如何限制访问来源？

**IP白名单**：
```json
{
  "security": {
    "ip_whitelist": [
      "192.168.1.0/24",
      "10.0.0.0/8"
    ]
  }
}
```

**Nginx配置**：
```nginx
location /v1/ {
    allow 192.168.1.0/24;
    deny all;
    proxy_pass http://gateway:8080;
}
```

---

## 计费与用量

### Q20：如何估算成本？

**Token价格参考**（2024年）：

| 模型 | 输入价格 | 输出价格 |
|------|---------|---------|
| GPT-4 | $0.03/1K | $0.06/1K |
| GPT-3.5-Turbo | $0.001/1K | $0.002/1K |
| Claude-3-Opus | $0.015/1K | $0.075/1K |

**成本计算公式**：
```
总成本 = (输入tokens × 输入价格) + (输出tokens × 输出价格)
```

**示例**：
- 输入：1000 tokens
- 输出：500 tokens
- 模型：GPT-4
- 成本：(1000 × 0.03/1000) + (500 × 0.06/1000) = $0.06

---

### Q21：如何查看用量统计？

**API查询**：
```bash
curl http://localhost:8080/api/v1/usage \
  -H "Authorization: Bearer YOUR_KEY"
```

**响应示例**：
```json
{
  "period": "2024-01",
  "total_requests": 15000,
  "total_tokens": 500000,
  "by_provider": {
    "openai": {
      "requests": 10000,
      "tokens": 350000
    },
    "claude": {
      "requests": 5000,
      "tokens": 150000
    }
  }
}
```

---

### Q22：如何设置用量告警？

```json
{
  "alerts": {
    "enabled": true,
    "thresholds": {
      "daily_tokens": 100000,
      "daily_cost": 10.0
    },
    "notifications": {
      "email": "admin@example.com",
      "webhook": "https://your-webhook-url"
    }
  }
}
```

---

## 更多问题？

如果你在本文档中没有找到答案：

1. 查看 [API参考文档](./API_REFERENCE.md)
2. 查看 [部署指南](./DEPLOYMENT.md)
3. 在GitHub提交Issue
4. 加入社区讨论

---

**最后更新**：2024年1月

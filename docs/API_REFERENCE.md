# AI Gateway API 参考文档

## 概述

AI智能网关提供与OpenAI兼容的API接口，让您可以轻松切换不同的AI服务提供商（OpenAI、Claude、火山引擎等），而无需修改代码。

**基础URL**: `http://localhost:8080`

**认证方式**: Bearer Token（在请求头中设置 `Authorization: Bearer YOUR_API_KEY`）

---

## 快速开始

### 1. 聊天补全（Chat Completions）

最常用的API，用于生成对话响应。

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "你好，请介绍一下你自己"}
    ]
  }'
```

### 2. 文本补全（Completions）

传统的文本补全接口（已较少使用）。

```bash
curl -X POST http://localhost:8080/api/v1/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-3.5-turbo-instruct",
    "prompt": "请续写这个故事：从前有一座山"
  }'
```

### 3. 文本嵌入（Embeddings）

生成文本的向量表示，用于语义搜索、聚类等。

```bash
curl -X POST http://localhost:8080/api/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "text-embedding-ada-002",
    "input": "这是一个测试文本"
  }'
```

---

## API 端点详情

### 1. 聊天补全

**端点**: `POST /api/v1/chat/completions`

**描述**: 创建一个聊天对话补全请求

**请求体**:

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| model | string | 是 | 模型名称（如 `gpt-4`, `claude-3-opus`） |
| messages | array | 是 | 对话消息数组 |
| temperature | number | 否 | 采样温度（0-2），默认1 |
| max_tokens | integer | 否 | 最大生成token数 |
| top_p | number | 否 | 核采样参数（0-1） |
| stream | boolean | 否 | 是否流式输出，默认false |

**消息对象格式**:

```json
{
  "role": "user|assistant|system",
  "content": "消息内容"
}
```

**示例请求**:

```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "你是一个有帮助的助手"},
    {"role": "user", "content": "你好"}
  ],
  "temperature": 0.7,
  "max_tokens": 1000
}
```

**示例响应**:

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
        "content": "你好！很高兴见到你。有什么我可以帮助你的吗？"
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

### 2. 模型列表

**端点**: `GET /v1/models`

**描述**: 获取可用的模型列表

**示例请求**:

```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**示例响应**:

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4",
      "object": "model",
      "created": 1234567890,
      "owned_by": "openai"
    },
    {
      "id": "claude-3-opus",
      "object": "model",
      "created": 1234567890,
      "owned_by": "anthropic"
    }
  ]
}
```

---

### 3. 文本嵌入

**端点**: `POST /api/v1/embeddings`

**描述**: 生成文本的向量嵌入

**请求体**:

| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| model | string | 是 | 嵌入模型名称 |
| input | string/array | 是 | 要嵌入的文本 |

**示例请求**:

```json
{
  "model": "text-embedding-ada-002",
  "input": "这是一个测试文本"
}
```

**示例响应**:

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [0.0023, -0.0054, ...]
    }
  ],
  "model": "text-embedding-ada-002",
  "usage": {
    "prompt_tokens": 5,
    "total_tokens": 5
  }
}
```

---

### 4. 服务商列表

**端点**: `GET /api/v1/providers`

**描述**: 获取配置的AI服务提供商列表

**示例响应**:

```json
{
  "providers": [
    {
      "name": "openai",
      "enabled": true,
      "models": ["gpt-4", "gpt-3.5-turbo"]
    },
    {
      "name": "claude",
      "enabled": true,
      "models": ["claude-3-opus", "claude-3-sonnet"]
    }
  ]
}
```

---

### 5. 健康检查

**端点**: `GET /health`

**描述**: 检查服务健康状态（无需认证）

**示例响应**:

```json
{
  "status": "healthy",
  "service": "ai-gateway",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## 错误处理

### 错误响应格式

```json
{
  "error": {
    "message": "错误描述",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

### 常见错误码

| HTTP状态码 | 错误类型 | 描述 |
|-----------|---------|------|
| 400 | invalid_request_error | 请求参数错误 |
| 401 | authentication_error | API密钥无效或缺失 |
| 403 | permission_error | 无权限访问 |
| 404 | not_found_error | 资源不存在 |
| 429 | rate_limit_error | 超过速率限制 |
| 500 | api_error | 服务器内部错误 |
| 503 | service_unavailable | 服务暂时不可用 |

---

## 速率限制

- **默认限制**: 每分钟60次请求
- **Token限制**: 每分钟40000 tokens

### 速率限制响应头

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 59
X-RateLimit-Reset: 1234567890
```

---

## 流式响应

设置 `stream: true` 可以获得流式响应：

```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "讲个故事"}],
  "stream": true
}
```

流式响应格式（SSE）：

```
data: {"id":"chatcmpl-123","choices":[{"delta":{"content":"很"},"index":0}]}

data: {"id":"chatcmpl-123","choices":[{"delta":{"content":"久"},"index":0}]}

data: [DONE]
```

---

## SDK使用示例

### Python

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-api-key"
)

response = client.chat.completions.create(
    model="gpt-4",
    messages=[
        {"role": "user", "content": "你好"}
    ]
)

print(response.choices[0].message.content)
```

### JavaScript/Node.js

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'your-api-key'
});

const response = await client.chat.completions.create({
  model: 'gpt-4',
  messages: [{ role: 'user', content: '你好' }]
});

console.log(response.choices[0].message.content);
```

### Go

```go
package main

import (
    "context"
    "fmt"
    "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("your-api-key")
    config.BaseURL = "http://localhost:8080/v1"

    client := openai.NewClientWithConfig(config)

    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: openai.GPT4,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: "你好",
                },
            },
        },
    )

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

---

## 最佳实践

1. **错误重试**: 遇到429错误时，使用指数退避重试
2. **流式响应**: 对于长文本生成，使用流式响应提升用户体验
3. **Token管理**: 合理设置 `max_tokens` 避免浪费
4. **缓存利用**: 相同请求会被缓存，提高响应速度

---

## 相关链接

- [用户使用手册](./USER_GUIDE.md)
- [部署指南](./DEPLOYMENT.md)
- [常见问题](./FAQ.md)

---

## 向量数据库（Vector DB）扩展 API

以下接口用于向量集合管理、检索、监控、权限、备份恢复与可视化。

### 向量检索接口（业务侧）

- `POST /api/v1/vector/collections/:name/search`
- `POST /api/v1/vector/collections/:name/recommend`
- `GET /api/v1/vector/collections/:name/vectors/:id`

请求示例（search）：

```json
{
  "top_k": 5,
  "min_score": 0.3,
  "vector": [0.1, 0.2, 0.3]
}
```

### 管理端接口（Admin）

- 集合管理：`/api/admin/vector-db/collections`
  - 清空集合：`POST /api/admin/vector-db/collections/:name/empty`
- 导入任务：`/api/admin/vector-db/import-jobs`
  - 取消任务：`POST /api/admin/vector-db/import-jobs/:id/cancel`
- 审计查询：`GET /api/admin/vector-db/audit/logs`
- 监控汇总：`GET /api/admin/vector-db/metrics/summary`
- 告警规则：`/api/admin/vector-db/alerts/rules`
- 索引配置：`GET|PUT /api/admin/vector-db/index-config/:name`
- 权限管理：`/api/admin/vector-db/permissions`
- 备份恢复：`/api/admin/vector-db/backups`
  - 备份策略执行：`POST /api/admin/vector-db/backups/policy/run`
- 可视化采样：`GET /api/admin/vector-db/visualization/scatter`

说明：`POST /api/v1/vector/collections/:name/search` 现已支持 `text` 参数，服务端会将文本转换为向量后执行检索。

可视化示例请求：

```bash
curl "http://localhost:8080/api/admin/vector-db/visualization/scatter?collection_name=docs&sample_size=200" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

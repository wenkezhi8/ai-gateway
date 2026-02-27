<template>
  <div class="docs-page">
    <div class="docs-header">
      <h1>文档中心</h1>
      <p>AI Gateway 完整使用指南和 API 参考文档</p>
    </div>

    <el-tabs v-model="activeTab" class="docs-tabs">
      <!-- 快速开始 -->
      <el-tab-pane label="快速开始" name="quickstart">
        <div class="doc-section">
          <h2 id="overview">概述</h2>
          <p>AI Gateway 是一个统一的 AI 服务网关，支持 OpenAI、Anthropic、智谱 AI、通义千问、DeepSeek 等多种 AI 服务商。</p>
          
          <h3>核心功能</h3>
          <ul>
            <li><strong>统一 API</strong> - 兼容 OpenAI API 格式，一次接入，多处使用</li>
            <li><strong>智能路由</strong> - 根据成本、速度、质量自动选择最优模型</li>
            <li><strong>多账号管理</strong> - 支持多账号负载均衡和自动切换</li>
            <li><strong>响应缓存</strong> - 智能缓存重复请求，降低成本</li>
            <li><strong>限额管控</strong> - 精细化的 Token 和 RPM 限额管理</li>
            <li><strong>监控告警</strong> - 实时监控和告警通知</li>
          </ul>

          <h2 id="installation">安装部署</h2>
          
          <h3>Docker 部署（推荐）</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('docker')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-docker"># 克隆项目
git clone https://github.com/wenkezhi8/ai-gateway.git
cd ai-gateway

# 使用 Docker Compose 启动
docker-compose up -d

# 查看日志
docker-compose logs -f</code></pre>
          </div>

          <h3>源码编译</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('build')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-build"># 编译后端
make build

# 安装前端依赖
cd web && npm install

# 构建前端
npm run build

# 启动服务
./bin/ai-gateway</code></pre>
          </div>

          <h2 id="config">配置说明</h2>
          <p>配置文件位于 <code>configs/config.json</code>：</p>
          
          <div class="code-block">
            <div class="code-header">
              <span>json</span>
              <button @click="copyCode('config')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-config">{
  "server": {
    "port": "8566",
    "mode": "release"
  },
  "providers": [
    {
      "name": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "base_url": "https://api.openai.com/v1",
      "enabled": true,
      "models": ["gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"]
    },
    {
      "name": "zhipu",
      "api_key": "${ZHIPU_API_KEY}",
      "base_url": "https://open.bigmodel.cn/api/paas/v4",
      "enabled": true,
      "models": ["glm-4-plus", "glm-4-flash"]
    }
  ]
}</code></pre>
          </div>

          <h3>环境变量</h3>
          <table class="config-table">
            <thead>
              <tr>
                <th>变量名</th>
                <th>说明</th>
                <th>默认值</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>JWT_SECRET</code></td>
                <td>JWT 密钥（生产环境必需）</td>
                <td>-</td>
              </tr>
              <tr>
                <td><code>PORT</code></td>
                <td>服务端口</td>
                <td>8566</td>
              </tr>
              <tr>
                <td><code>METRICS_PORT</code></td>
                <td>Metrics 端口</td>
                <td>9090</td>
              </tr>
              <tr>
                <td><code>GIN_MODE</code></td>
                <td>运行模式 (debug/release)</td>
                <td>debug</td>
              </tr>
            </tbody>
          </table>
        </div>
      </el-tab-pane>

      <!-- API 参考 -->
      <el-tab-pane label="API 参考" name="api">
        <div class="doc-section">
          <h2 id="api-overview">API 概述</h2>
          <p>AI Gateway 当前提供两套兼容协议：OpenAI 兼容协议与 Anthropic 兼容协议。</p>

          <h3>协议入口</h3>
          <table class="api-table">
            <thead>
              <tr>
                <th>协议</th>
                <th>Base URL</th>
                <th>核心接口</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>OpenAI 兼容</td>
                <td><code>/api/v1</code></td>
                <td><code>POST /chat/completions</code></td>
              </tr>
              <tr>
                <td>Anthropic 兼容</td>
                <td><code>/api/anthropic</code></td>
                <td><code>POST /v1/messages</code></td>
              </tr>
            </tbody>
          </table>

          <h3>认证方式</h3>
          <p>根据协议使用不同请求头：</p>
          <ul>
            <li>OpenAI 兼容：<code>Authorization: Bearer YOUR_API_KEY</code></li>
            <li>Anthropic 兼容：<code>x-api-key: YOUR_API_KEY</code>（建议同时带 <code>anthropic-version</code>）</li>
          </ul>
          <div class="code-block">
            <div class="code-header">
              <span>http</span>
              <button @click="copyCode('auth')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-auth"># OpenAI 兼容
Authorization: Bearer YOUR_API_KEY

# Anthropic 兼容
x-api-key: YOUR_API_KEY
anthropic-version: 2023-06-01</code></pre>
          </div>

          <h2 id="chat-completions">聊天补全</h2>
          <div class="api-card">
            <div class="api-method post">POST</div>
            <div class="api-path">/api/v1/chat/completions</div>
          </div>
          <p>发送聊天请求，支持流式和非流式响应。</p>
          
          <h4>请求参数</h4>
          <table class="api-table">
            <thead>
              <tr>
                <th>参数</th>
                <th>类型</th>
                <th>必填</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>model</code></td>
                <td>string</td>
                <td>是</td>
                <td>模型名称，如 gpt-4o、glm-4-plus</td>
              </tr>
              <tr>
                <td><code>messages</code></td>
                <td>array</td>
                <td>是</td>
                <td>消息数组</td>
              </tr>
              <tr>
                <td><code>temperature</code></td>
                <td>number</td>
                <td>否</td>
                <td>温度参数，0-2，默认 1</td>
              </tr>
              <tr>
                <td><code>max_tokens</code></td>
                <td>integer</td>
                <td>否</td>
                <td>最大生成 Token 数</td>
              </tr>
              <tr>
                <td><code>stream</code></td>
                <td>boolean</td>
                <td>否</td>
                <td>是否流式输出，默认 false</td>
              </tr>
              <tr>
                <td><code>provider</code></td>
                <td>string</td>
                <td>否</td>
                <td>指定服务商，不填自动选择</td>
              </tr>
            </tbody>
          </table>

          <h4>请求示例</h4>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('chat-curl')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-chat-curl">curl -X POST http://localhost:8566/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "system", "content": "你是一个有帮助的助手。"},
      {"role": "user", "content": "你好，请介绍一下自己。"}
    ],
    "temperature": 0.7,
    "max_tokens": 1000
  }'</code></pre>
          </div>

          <h4>响应示例</h4>
          <div class="code-block">
            <div class="code-header">
              <span>json</span>
              <button @click="copyCode('chat-response')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-chat-response">{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1703000000,
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！我是一个AI助手..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 25,
    "completion_tokens": 100,
    "total_tokens": 125
  }
}</code></pre>
          </div>

          <h2 id="streaming">流式响应</h2>
          <p>设置 <code>stream: true</code> 启用流式输出：</p>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('stream-curl')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-stream-curl">curl -X POST http://localhost:8566/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "写一首诗"}],
    "stream": true
  }'</code></pre>
          </div>

          <h2 id="anthropic-messages">Anthropic Messages</h2>
          <div class="api-card">
            <div class="api-method post">POST</div>
            <div class="api-path">/api/anthropic/v1/messages</div>
          </div>
          <p>Anthropic 协议兼容入口，支持文本、多模态输入与流式返回。</p>

          <h4>请求参数</h4>
          <table class="api-table">
            <thead>
              <tr>
                <th>参数</th>
                <th>类型</th>
                <th>必填</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>model</code></td>
                <td>string</td>
                <td>是</td>
                <td>模型名，如 <code>claude-3-5-sonnet-20241022</code>、<code>auto</code></td>
              </tr>
              <tr>
                <td><code>messages</code></td>
                <td>array</td>
                <td>是</td>
                <td>消息数组，支持字符串或 content blocks</td>
              </tr>
              <tr>
                <td><code>max_tokens</code></td>
                <td>integer</td>
                <td>否</td>
                <td>最大输出 token 数</td>
              </tr>
              <tr>
                <td><code>stream</code></td>
                <td>boolean</td>
                <td>否</td>
                <td>是否启用 SSE 流式输出</td>
              </tr>
              <tr>
                <td><code>tools</code></td>
                <td>array</td>
                <td>否</td>
                <td>工具定义，支持 tool_use/tool_result</td>
              </tr>
            </tbody>
          </table>

          <h4>请求示例</h4>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('anthropic-curl')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-anthropic-curl">curl -X POST http://localhost:8566/api/anthropic/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "你好，请介绍一下你自己。"}
    ]
  }'</code></pre>
          </div>

          <h4>响应示例</h4>
          <div class="code-block">
            <div class="code-header">
              <span>json</span>
              <button @click="copyCode('anthropic-response')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-anthropic-response">{
  "id": "msg_01Example",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "你好！我是 AI Gateway 后端模型。"
    }
  ],
  "model": "claude-3-5-sonnet-20241022",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 18,
    "output_tokens": 24
  }
}</code></pre>
          </div>

          <h2 id="embeddings">向量嵌入</h2>
          <div class="api-card">
            <div class="api-method post">POST</div>
            <div class="api-path">/api/v1/embeddings</div>
          </div>
          
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('embeddings')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-embeddings">curl -X POST http://localhost:8566/api/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "text-embedding-ada-002",
    "input": "Hello world"
  }'</code></pre>
          </div>

          <h2 id="models">模型列表</h2>
          <div class="api-card">
            <div class="api-method get">GET</div>
            <div class="api-path">/api/v1/models</div>
          </div>
          
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('models')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-models">curl http://localhost:8566/api/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"</code></pre>
          </div>

          <h2 id="providers">服务商列表</h2>
          <div class="api-card">
            <div class="api-method get">GET</div>
            <div class="api-path">/api/v1/providers</div>
          </div>
          
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('providers')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-providers">curl http://localhost:8566/api/v1/providers \
  -H "Authorization: Bearer YOUR_API_KEY"</code></pre>
          </div>
        </div>
      </el-tab-pane>

      <!-- SDK 示例 -->
      <el-tab-pane label="SDK 示例" name="sdk">
        <div class="doc-section">
          <h2>Python SDK</h2>
          <p>OpenAI 协议可使用 OpenAI SDK，Anthropic 协议可使用 Anthropic SDK。</p>
          
          <h3>安装依赖</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('pip-install')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-pip-install">pip install openai</code></pre>
          </div>

          <h3>基本用法</h3>
          <div class="code-block">
            <div class="code-header">
              <span>python</span>
              <button @click="copyCode('python-basic')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-python-basic">from openai import OpenAI

client = OpenAI(
    api_key="YOUR_API_KEY",
    base_url="http://localhost:8566/api/v1"
)

# 非流式请求
response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "system", "content": "你是一个有帮助的助手。"},
        {"role": "user", "content": "你好！"}
    ]
)

print(response.choices[0].message.content)</code></pre>
          </div>

          <h3>流式输出</h3>
          <div class="code-block">
            <div class="code-header">
              <span>python</span>
              <button @click="copyCode('python-stream')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-python-stream"># 流式请求
stream = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "写一首诗"}],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)</code></pre>
          </div>

          <h3>异步请求</h3>
          <div class="code-block">
            <div class="code-header">
              <span>python</span>
              <button @copy="copyCode('python-async')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-python-async">import asyncio
from openai import AsyncOpenAI

async def main():
    client = AsyncOpenAI(
        api_key="YOUR_API_KEY",
        base_url="http://localhost:8566/api/v1"
    )
    
    response = await client.chat.completions.create(
        model="gpt-4o",
        messages=[{"role": "user", "content": "Hello!"}]
    )
    
    print(response.choices[0].message.content)

asyncio.run(main())</code></pre>
          </div>

          <h3>Anthropic SDK（Python）</h3>
          <div class="code-block">
            <div class="code-header">
              <span>python</span>
              <button @click="copyCode('python-anthropic')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-python-anthropic">from anthropic import Anthropic

client = Anthropic(
    api_key="YOUR_API_KEY",
    base_url="http://localhost:8566/api/anthropic"
)

message = client.messages.create(
    model="claude-3-5-sonnet-20241022",
    max_tokens=1024,
    messages=[{"role": "user", "content": "你好"}]
)

print(message.content[0].text)</code></pre>
          </div>

          <h2>Node.js SDK</h2>
          
          <h3>安装依赖</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('npm-install')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-npm-install">npm install openai</code></pre>
          </div>

          <h3>基本用法</h3>
          <div class="code-block">
            <div class="code-header">
              <span>javascript</span>
              <button @click="copyCode('js-basic')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-js-basic">import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: 'YOUR_API_KEY',
  baseURL: 'http://localhost:8566/api/v1',
});

async function main() {
  const response = await client.chat.completions.create({
    model: 'gpt-4o',
    messages: [{ role: 'user', content: 'Hello!' }],
  });
  
  console.log(response.choices[0].message.content);
}

main();</code></pre>
          </div>

          <h3>流式输出</h3>
          <div class="code-block">
            <div class="code-header">
              <span>javascript</span>
              <button @click="copyCode('js-stream')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-js-stream">const stream = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: '写一首诗' }],
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}</code></pre>
          </div>

          <h3>Anthropic SDK（Node.js）</h3>
          <div class="code-block">
            <div class="code-header">
              <span>javascript</span>
              <button @click="copyCode('js-anthropic')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-js-anthropic">import Anthropic from '@anthropic-ai/sdk';

const client = new Anthropic({
  apiKey: 'YOUR_API_KEY',
  baseURL: 'http://localhost:8566/api/anthropic'
});

const message = await client.messages.create({
  model: 'claude-3-5-sonnet-20241022',
  max_tokens: 1024,
  messages: [{ role: 'user', content: '你好' }]
});

console.log(message.content[0].text);</code></pre>
          </div>

          <h2>Go SDK</h2>
          <div class="code-block">
            <div class="code-header">
              <span>go</span>
              <button @click="copyCode('go-basic')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-go-basic">package main

import (
    "context"
    "fmt"
    "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("YOUR_API_KEY")
    config.BaseURL = "http://localhost:8566/api/v1"
    
    client := openai.NewClientWithConfig(config)
    
    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: openai.GPT4o,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: "Hello!",
                },
            },
        },
    )
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println(resp.Choices[0].Message.Content)
}</code></pre>
          </div>

          <h2>opencode CLI</h2>
          <p>
            <a href="https://opencode.ai" target="_blank">opencode</a> 
            是一个强大的终端 AI 编程助手，可以直接配置使用 AI Gateway 作为后端。
          </p>

          <h3>安装 opencode</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('opencode-install')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-opencode-install"># macOS / Linux
curl -fsSL https://opencode.ai/install | bash

# 或使用 npm
npm install -g opencode-ai

# 或使用 Homebrew
brew install anomalyco/tap/opencode</code></pre>
          </div>

          <h3>配置 AI Gateway</h3>
          <p>创建或编辑 <code>~/.config/opencode/opencode.json</code>：</p>
          <div class="code-block">
            <div class="code-header">
              <span>json</span>
              <button @click="copyCode('opencode-config')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-opencode-config">{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "ai-gateway": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "AI Gateway",
      "options": {
        "baseURL": "http://localhost:8566/api/v1"
      },
      "models": {
        "auto": {
          "name": "Auto (智能选择)",
          "limit": { "context": 128000, "output": 4096 }
        },
        "glm-4-flash": { "name": "GLM-4-Flash" },
        "glm-4.7": { "name": "GLM-4.7" },
        "glm-4-plus": { "name": "GLM-4-Plus" }
      }
    }
  }
}</code></pre>
          </div>

          <h3>添加 API Key 认证</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('opencode-auth')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-opencode-auth"># 交互式登录
opencode auth login

# 或手动添加凭证
mkdir -p ~/.local/share/opencode
echo '{"ai-gateway":{"apiKey":"YOUR_API_KEY"}}' > ~/.local/share/opencode/auth.json</code></pre>
          </div>

          <h3>使用示例</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('opencode-usage')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-opencode-usage"># 进入项目目录
cd /path/to/your/project

# 交互模式
opencode

# 非交互模式 - 指定模型
opencode run -m ai-gateway/auto '解释这个函数的作用'

# 使用 GLM-4-Flash 模型
opencode run -m ai-gateway/glm-4-flash '帮我写一个冒泡排序'

# 使用 GLM-4.7 模型
opencode run -m ai-gateway/glm-4.7 '分析这段代码的性能问题'

# 指定工作目录
opencode -c /path/to/project '添加单元测试'

# 输出 JSON 格式
opencode run -f json '列出所有 TODO'</code></pre>
          </div>

          <h3>常用模型</h3>
          <table class="config-table">
            <thead>
              <tr>
                <th>模型</th>
                <th>说明</th>
                <th>适用场景</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>ai-gateway/auto</code></td>
                <td>智能选择</td>
                <td>推荐：自动选择最优模型</td>
              </tr>
              <tr>
                <td><code>ai-gateway/glm-4-flash</code></td>
                <td>快速响应</td>
                <td>简单问答、代码补全</td>
              </tr>
              <tr>
                <td><code>ai-gateway/glm-4.7</code></td>
                <td>最新版本</td>
                <td>复杂推理、代码生成</td>
              </tr>
              <tr>
                <td><code>ai-gateway/glm-4-plus</code></td>
                <td>高级版</td>
                <td>复杂任务、长文本处理</td>
              </tr>
            </tbody>
          </table>


          <h2>OpenClaw 配置</h2>
          <p>
            <a href="https://clawd.bot" target="_blank">OpenClaw</a> 
            是一个强大的终端 AI 助手，可以直接配置使用 AI Gateway 作为后端。
          </p>

          <h3>一键配置脚本（推荐）</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('openclaw-setup')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-openclaw-setup"># 下载并执行配置脚本
curl -fsSL http://localhost:8566/scripts/setup-openclaw.sh | bash

# 或手动下载执行
curl -o setup-openclaw.sh http://localhost:8566/scripts/setup-openclaw.sh
chmod +x setup-openclaw.sh
./setup-openclaw.sh</code></pre>
          </div>

          <h3>手动配置</h3>
          <p>编辑 <code>~/.openclaw/openclaw.json</code>：</p>
          <div class="code-block">
            <div class="code-header">
              <span>json</span>
              <button @click="copyCode('openclaw-config')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-openclaw-config">{
  "auth": {
    "profiles": {
      "ai-gateway:default": {
        "provider": "ai-gateway",
        "mode": "api_key"
      }
    }
  },
  "models": {
    "providers": {
      "ai-gateway": {
        "baseUrl": "http://localhost:8566/api/v1",
        "api": "openai-completions",
        "apiKey": "YOUR_API_KEY",
        "models": [
          {
            "id": "auto",
            "name": "Auto (智能选择)",
            "contextWindow": 200000,
            "maxTokens": 8192
          }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "models": {
        "ai-gateway/auto": { "alias": "AI Gateway" }
      }
    }
  }
}</code></pre>
          </div>

          <h3>常用命令</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('openclaw-usage')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-openclaw-usage"># 设置默认模型
openclaw-cn models set ai-gateway/auto

# 查看模型状态
openclaw-cn models status

# 启动 TUI
openclaw-cn tui

# 重启网关
openclaw-cn gateway restart</code></pre>
          </div>

          <h3>可用模型</h3>
          <table class="config-table">
            <thead>
              <tr>
                <th>模型</th>
                <th>说明</th>
                <th>适用场景</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>ai-gateway/auto</code></td>
                <td>智能选择</td>
                <td>推荐：自动选择最优模型</td>
              </tr>
              <tr>
                <td><code>ai-gateway/gpt-4o</code></td>
                <td>GPT-4o</td>
                <td>复杂推理、代码生成</td>
              </tr>
              <tr>
                <td><code>ai-gateway/claude-3-5-sonnet-20241022</code></td>
                <td>Claude 3.5 Sonnet</td>
                <td>长文本、代码分析</td>
              </tr>
            </tbody>
          </table>
        </div>
      </el-tab-pane>

      <!-- 支持的服务商 -->
      <el-tab-pane label="服务商" name="providers">
        <div class="doc-section">
          <h2>支持的服务商</h2>
          
          <div class="provider-grid">
            <div class="provider-card" v-for="provider in providers" :key="provider.name">
              <div class="provider-header">
                <span class="provider-name">{{ provider.name }}</span>
                <el-tag :type="provider.enabled ? 'success' : 'info'" size="small">
                  {{ provider.enabled ? '已启用' : '未启用' }}
                </el-tag>
              </div>
              <div class="provider-models">
                <span class="label">支持模型：</span>
                <el-tag v-for="model in provider.models.slice(0, 4)" :key="model" size="small" class="model-tag">
                  {{ model }}
                </el-tag>
                <span v-if="provider.models.length > 4" class="more">+{{ provider.models.length - 4 }}</span>
              </div>
              <div class="provider-endpoint">
                <span class="label">端点：</span>
                <code>{{ provider.endpoint }}</code>
              </div>
            </div>
          </div>

          <h2>服务商端点配置</h2>
          <table class="config-table">
            <thead>
              <tr>
                <th>服务商</th>
                <th>名称</th>
                <th>API 端点</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>OpenAI</td>
                <td><code>openai</code></td>
                <td>https://api.openai.com/v1</td>
              </tr>
              <tr>
                <td>Anthropic</td>
                <td><code>anthropic</code></td>
                <td>https://api.anthropic.com/v1</td>
              </tr>
              <tr>
                <td>智谱 AI</td>
                <td><code>zhipu</code></td>
                <td>https://open.bigmodel.cn/api/paas/v4</td>
              </tr>
              <tr>
                <td>通义千问</td>
                <td><code>qwen</code></td>
                <td>https://dashscope.aliyuncs.com/api/v1</td>
              </tr>
              <tr>
                <td>DeepSeek</td>
                <td><code>deepseek</code></td>
                <td>https://api.deepseek.com/v1</td>
              </tr>
              <tr>
                <td>火山方舟</td>
                <td><code>volcengine</code></td>
                <td>https://ark.cn-beijing.volces.com/api/v3</td>
              </tr>
              <tr>
                <td>文心一言</td>
                <td><code>ernie</code></td>
                <td>https://aip.baidubce.com/rpc/2.0/ai_custom/v1</td>
              </tr>
            </tbody>
          </table>
        </div>
      </el-tab-pane>

      <!-- 管理员 API -->
      <el-tab-pane label="管理 API" name="admin">
        <div class="doc-section">
          <h2>管理员 API</h2>
          <p>管理员 API 用于管理账号、路由、缓存等，前缀为 <code>/api/admin</code>。</p>

          <h3>账号管理</h3>
          <table class="api-table">
            <thead>
              <tr>
                <th>方法</th>
                <th>路径</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><span class="method get">GET</span></td>
                <td><code>/api/admin/accounts</code></td>
                <td>获取账号列表</td>
              </tr>
              <tr>
                <td><span class="method post">POST</span></td>
                <td><code>/api/admin/accounts</code></td>
                <td>创建账号</td>
              </tr>
              <tr>
                <td><span class="method put">PUT</span></td>
                <td><code>/api/admin/accounts/:id</code></td>
                <td>更新账号</td>
              </tr>
              <tr>
                <td><span class="method delete">DELETE</span></td>
                <td><code>/api/admin/accounts/:id</code></td>
                <td>删除账号</td>
              </tr>
              <tr>
                <td><span class="method post">POST</span></td>
                <td><code>/api/admin/accounts/:id/switch</code></td>
                <td>强制切换账号</td>
              </tr>
            </tbody>
          </table>

          <h3>路由配置</h3>
          <table class="api-table">
            <thead>
              <tr>
                <th>方法</th>
                <th>路径</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><span class="method get">GET</span></td>
                <td><code>/api/admin/router/config</code></td>
                <td>获取路由配置</td>
              </tr>
              <tr>
                <td><span class="method put">PUT</span></td>
                <td><code>/api/admin/router/config</code></td>
                <td>更新路由配置</td>
              </tr>
              <tr>
                <td><span class="method get">GET</span></td>
                <td><code>/api/admin/router/models</code></td>
                <td>获取模型评分</td>
              </tr>
              <tr>
                <td><span class="method put">PUT</span></td>
                <td><code>/api/admin/router/models/:model</code></td>
                <td>更新模型评分</td>
              </tr>
            </tbody>
          </table>

          <h3>缓存管理</h3>
          <table class="api-table">
            <thead>
              <tr>
                <th>方法</th>
                <th>路径</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><span class="method get">GET</span></td>
                <td><code>/api/admin/cache/stats</code></td>
                <td>获取缓存统计</td>
              </tr>
              <tr>
                <td><span class="method delete">DELETE</span></td>
                <td><code>/api/admin/cache</code></td>
                <td>清空缓存</td>
              </tr>
              <tr>
                <td><span class="method delete">DELETE</span></td>
                <td><code>/api/admin/cache/provider/:provider</code></td>
                <td>清除服务商缓存</td>
              </tr>
            </tbody>
          </table>

          <h3>仪表盘数据</h3>
          <table class="api-table">
            <thead>
              <tr>
                <th>方法</th>
                <th>路径</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><span class="method get">GET</span></td>
                <td><code>/api/admin/dashboard/stats</code></td>
                <td>获取统计数据</td>
              </tr>
              <tr>
                <td><span class="method get">GET</span></td>
                <td><code>/api/admin/dashboard/realtime</code></td>
                <td>获取实时数据</td>
              </tr>
              <tr>
                <td><span class="method get">GET</span></td>
                <td><code>/api/admin/dashboard/requests</code></td>
                <td>获取请求趋势</td>
              </tr>
              <tr>
                <td><span class="method get">GET</span></td>
                <td><code>/api/admin/dashboard/system</code></td>
                <td>获取系统状态</td>
              </tr>
            </tbody>
          </table>

          <h3>示例：获取账号列表</h3>
          <div class="code-block">
            <div class="code-header">
              <span>bash</span>
              <button @click="copyCode('admin-accounts')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-admin-accounts">curl http://localhost:8566/api/admin/accounts \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"</code></pre>
          </div>
        </div>
      </el-tab-pane>

      <!-- 错误码 -->
      <el-tab-pane label="错误码" name="errors">
        <div class="doc-section">
          <h2>错误码参考</h2>
          
          <h3>HTTP 状态码</h3>
          <table class="config-table">
            <thead>
              <tr>
                <th>状态码</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>200</code></td>
                <td>请求成功</td>
              </tr>
              <tr>
                <td><code>400</code></td>
                <td>请求参数错误</td>
              </tr>
              <tr>
                <td><code>401</code></td>
                <td>未授权，缺少或无效的认证信息</td>
              </tr>
              <tr>
                <td><code>403</code></td>
                <td>禁止访问，权限不足</td>
              </tr>
              <tr>
                <td><code>404</code></td>
                <td>资源不存在</td>
              </tr>
              <tr>
                <td><code>429</code></td>
                <td>请求过于频繁，触发限流</td>
              </tr>
              <tr>
                <td><code>500</code></td>
                <td>服务器内部错误</td>
              </tr>
              <tr>
                <td><code>502</code></td>
                <td>上游服务错误（服务商 API 错误）</td>
              </tr>
              <tr>
                <td><code>503</code></td>
                <td>服务不可用</td>
              </tr>
            </tbody>
          </table>

          <h3>业务错误码</h3>
          <table class="config-table">
            <thead>
              <tr>
                <th>错误码</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>invalid_request</code></td>
                <td>请求参数无效</td>
              </tr>
              <tr>
                <td><code>unauthorized</code></td>
                <td>未授权</td>
              </tr>
              <tr>
                <td><code>rate_limit_exceeded</code></td>
                <td>超过速率限制</td>
              </tr>
              <tr>
                <td><code>provider_error</code></td>
                <td>服务商返回错误</td>
              </tr>
              <tr>
                <td><code>model_not_found</code></td>
                <td>模型不存在</td>
              </tr>
              <tr>
                <td><code>internal_error</code></td>
                <td>内部错误</td>
              </tr>
            </tbody>
          </table>

          <h3>错误响应格式</h3>
          <div class="code-block">
            <div class="code-header">
              <span>json</span>
              <button @click="copyCode('error-response')"><el-icon><CopyDocument /></el-icon> 复制</button>
            </div>
            <pre><code id="code-error-response">{
  "success": false,
  "error": {
    "code": "invalid_request",
    "message": "模型名称不能为空",
    "detail": "model field is required"
  }
}</code></pre>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { DOCS_PROVIDERS } from '@/constants/pages/docs'

const activeTab = ref('quickstart')

const providers = DOCS_PROVIDERS

const copyCode = async (id: string) => {
  const element = document.getElementById(`code-${id}`)
  if (element) {
    try {
      await navigator.clipboard.writeText(element.textContent || '')
      ElMessage.success('已复制到剪贴板')
    } catch (e) {
      ElMessage.error('复制失败')
    }
  }
}
</script>

<style scoped lang="scss">
.docs-page {
  padding: 0;
}

.docs-header {
  margin-bottom: 24px;
  
  h1 {
    font-size: 28px;
    font-weight: 600;
    margin: 0 0 8px 0;
    color: var(--text-primary);
  }
  
  p {
    color: var(--text-secondary);
    margin: 0;
  }
}

.docs-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 24px;
    border-bottom: 1px solid var(--border-primary);
  }
  
  :deep(.el-tabs__nav-wrap::after) {
    display: none;
  }
  
  :deep(.el-tabs__item) {
    font-size: 14px;
    padding: 0 20px;
    height: 44px;
    line-height: 44px;
    
    &.is-active {
      font-weight: 600;
    }
  }
}

.doc-section {
  h2 {
    font-size: 20px;
    font-weight: 600;
    margin: 32px 0 16px 0;
    padding-top: 16px;
    border-top: 1px solid var(--border-primary);
    color: var(--text-primary);
    
    &:first-child {
      margin-top: 0;
      padding-top: 0;
      border-top: none;
    }
  }
  
  h3 {
    font-size: 16px;
    font-weight: 600;
    margin: 24px 0 12px 0;
    color: var(--text-primary);
  }
  
  h4 {
    font-size: 14px;
    font-weight: 600;
    margin: 16px 0 8px 0;
    color: var(--text-primary);
  }
  
  p {
    color: var(--text-secondary);
    line-height: 1.6;
    margin: 0 0 12px 0;
  }
  
  ul {
    padding-left: 20px;
    margin: 0 0 16px 0;
    
    li {
      color: var(--text-secondary);
      line-height: 1.8;
    }
  }
  
  code {
    background: var(--bg-tertiary);
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 13px;
    color: var(--color-primary);
  }
}

.code-block {
  background: var(--bg-tertiary);
  border-radius: 8px;
  overflow: hidden;
  margin: 16px 0;
  border: 1px solid var(--border-primary);
  
  .code-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 16px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-primary);
    
    span {
      font-size: 12px;
      color: var(--text-secondary);
      font-weight: 500;
      text-transform: uppercase;
    }
    
    button {
      display: flex;
      align-items: center;
      gap: 4px;
      background: none;
      border: none;
      color: var(--text-secondary);
      font-size: 12px;
      cursor: pointer;
      padding: 4px 8px;
      border-radius: 4px;
      
      &:hover {
        background: var(--bg-tertiary);
        color: var(--text-primary);
      }
    }
  }
  
  pre {
    margin: 0;
    padding: 16px;
    overflow-x: auto;
    
    code {
      background: none;
      padding: 0;
      font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
      font-size: 13px;
      line-height: 1.5;
      color: var(--text-primary);
    }
  }
}

.config-table,
.api-table {
  width: 100%;
  border-collapse: collapse;
  margin: 16px 0;
  
  th, td {
    padding: 12px 16px;
    text-align: left;
    border-bottom: 1px solid var(--border-primary);
  }
  
  th {
    background: var(--bg-tertiary);
    font-weight: 600;
    font-size: 13px;
    color: var(--text-secondary);
  }
  
  td {
    font-size: 14px;
    color: var(--text-primary);
    
    code {
      background: var(--bg-tertiary);
      padding: 2px 6px;
      border-radius: 4px;
      font-size: 13px;
    }
  }
  
  tbody tr:hover {
    background: var(--bg-tertiary);
  }
}

.api-card {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 16px 0;
  
  .api-method {
    padding: 4px 12px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    
    &.get {
      background: rgba(103, 194, 58, 0.1);
      color: #67c23a;
    }
    
    &.post {
      background: rgba(64, 158, 255, 0.1);
      color: #409eff;
    }
    
    &.put {
      background: rgba(230, 162, 60, 0.1);
      color: #e6a23c;
    }
    
    &.delete {
      background: rgba(245, 108, 108, 0.1);
      color: #f56c6c;
    }
  }
  
  .api-path {
    font-family: monospace;
    font-size: 14px;
    color: var(--text-primary);
  }
}

.method {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  
  &.get {
    background: rgba(103, 194, 58, 0.1);
    color: #67c23a;
  }
  
  &.post {
    background: rgba(64, 158, 255, 0.1);
    color: #409eff;
  }
  
  &.put {
    background: rgba(230, 162, 60, 0.1);
    color: #e6a23c;
  }
  
  &.delete {
    background: rgba(245, 108, 108, 0.1);
    color: #f56c6c;
  }
}

.provider-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  margin: 16px 0;
}

.provider-card {
  background: var(--bg-tertiary);
  border-radius: 12px;
  padding: 20px;
  border: 1px solid var(--border-primary);
  
  .provider-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    
    .provider-name {
      font-size: 16px;
      font-weight: 600;
      color: var(--text-primary);
    }
  }
  
  .provider-models {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
    
    .label {
      font-size: 13px;
      color: var(--text-secondary);
    }
    
    .model-tag {
      background: var(--bg-secondary);
      border: none;
      color: var(--text-secondary);
    }
    
    .more {
      font-size: 12px;
      color: var(--text-secondary);
    }
  }
  
  .provider-endpoint {
    .label {
      font-size: 13px;
      color: var(--text-secondary);
      margin-right: 8px;
    }
    
    code {
      font-size: 12px;
      word-break: break-all;
    }
  }
}
</style>

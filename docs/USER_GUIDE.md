# AI Gateway 用户使用手册

> 适合零基础用户的完整指南

---

## 目录

1. [什么是AI网关？](#什么是ai网关)
2. [快速开始](#快速开始)
3. [安装部署](#安装部署)
4. [基础使用](#基础使用)
5. [Web控制台使用](#web控制台使用)
6. [常见场景示例](#常见场景示例)
7. [常见问题](#常见问题)
8. [故障排查](#故障排查)

---

## 什么是AI网关？

### 简单理解

想象一下，你是一家餐厅的老板，需要从不同的供应商那里采购食材：

- **供应商A**（OpenAI）- 提供优质牛肉
- **供应商B**（Claude）- 提供新鲜蔬菜
- **供应商C**（火山引擎）- 提供调料

**AI网关就像一个智能采购经理**：
- 你只需要告诉它"我要做一道菜"
- 它会自动选择最合适的供应商
- 它会帮你比价、监控用量、管理库存

### 核心优势

| 功能 | 好处 |
|------|------|
| 统一接口 | 学一次API，用所有AI服务 |
| 自动切换 | 一个服务商挂了，自动换另一个 |
| 成本优化 | 自动选择最便宜的提供商 |
| 用量监控 | 实时查看API调用次数和费用 |
| 速率限制 | 防止意外超额消费 |

---

## 快速开始

### 前提条件

在开始之前，请确保你有：

- [ ] 一台电脑（Windows/Mac/Linux都可以）
- [ ] Docker已安装（[如何安装Docker？](#如何安装docker)）
- [ ] 至少一个AI服务商的API Key

### 五分钟快速启动

#### 第一步：下载项目

打开终端（Terminal），运行：

```bash
# 下载项目
git clone https://github.com/wenkezhi8/ai-gateway.git

# 进入项目目录
cd ai-gateway
```

#### 第二步：配置API Key

1. 复制配置文件模板：
```bash
cp .env.example .env
```

2. 用文本编辑器打开 `.env` 文件，填入你的API Key：
```bash
# OpenAI的API Key（从 https://platform.openai.com 获取）
OPENAI_API_KEY=sk-xxxxxxxxxxxxxx

# Claude的API Key（从 https://console.anthropic.com 获取）
ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxxxx
```

#### 第三步：启动服务

```bash
# 启动所有服务（网关 + Redis + 监控）
docker-compose up -d
```

#### 第四步：测试是否成功

打开浏览器，访问：
```
http://localhost:8080/health
```

如果看到 `{"status":"healthy"}`，恭喜你，启动成功！

---

## 安装部署

### 方式一：Docker部署（推荐）

**适合人群**：新手、快速体验、生产环境

#### 1. 安装Docker

**Windows/Mac**：
1. 访问 https://www.docker.com/products/docker-desktop
2. 下载并安装Docker Desktop
3. 安装完成后重启电脑

**Linux（Ubuntu）**：
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
```

#### 2. 启动服务

```bash
# 构建并启动
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

#### 3. 访问服务

| 服务 | 地址 | 说明 |
|------|------|------|
| API网关 | http://localhost:8080 | 主要API服务 |
| Web控制台 | http://localhost:8080 | 管理界面 |
| 监控面板 | http://localhost:3000 | Grafana监控 |

### 方式二：源码编译

**适合人群**：开发者、需要自定义修改

#### 1. 安装Go语言

```bash
# Mac
brew install go

# Ubuntu
sudo apt install golang-go

# Windows
# 访问 https://go.dev/dl/ 下载安装
```

#### 2. 编译运行

```bash
# 安装依赖
go mod download

# 编译
make build

# 运行
./ai-gateway
```

---

## 基础使用

### 发送第一个请求

使用curl命令：

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "你好，介绍一下你自己"}
    ]
  }'
```

### 使用Python调用

1. 安装OpenAI SDK：
```bash
pip install openai
```

2. 编写代码：
```python
from openai import OpenAI

# 创建客户端，指向你的AI网关
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-api-key"  # 可以是任意值
)

# 发送请求
response = client.chat.completions.create(
    model="gpt-4",
    messages=[
        {"role": "user", "content": "你好"}
    ]
)

# 获取回复
print(response.choices[0].message.content)
```

### 支持的模型

| 模型名称 | 提供商 | 适用场景 |
|---------|-------|---------|
| gpt-4 | OpenAI | 复杂推理、创意写作 |
| gpt-3.5-turbo | OpenAI | 日常对话、快速响应 |
| claude-3-opus | Anthropic | 深度分析、长文本 |
| claude-3-sonnet | Anthropic | 平衡性能与成本 |

---

## Web控制台使用

### 访问控制台

打开浏览器访问：`http://localhost:8080`

### 主要功能

#### 1. 仪表盘

显示关键指标：
- 今日请求次数
- Token使用量
- 错误率
- 平均响应时间

#### 2. 服务商管理

- 查看已配置的服务商
- 启用/禁用服务商
- 查看API Key状态

#### 3. 路由配置

- 设置路由策略（成本优先/性能优先）
- 配置模型映射
- 设置Fallback规则

#### 4. 用量监控

- 查看每日/每周/每月用量
- 按用户/服务商/模型分组统计
- 导出用量报表

#### 5. 限额设置

- 设置用户每日限额
- 设置Token配额
- 超额告警配置

#### 6. 向量数据库能力

控制台中的向量模块包含：

- 向量集合：创建、编辑、索引参数调整（HNSW/IVF）
- 向量检索：文本检索（自动向量化）、向量相似度搜索、推荐、按 ID 获取
- 向量导入：JSON/CSV/PDF 导入任务查看与重试
- 向量监控：指标总览与告警规则 CRUD
- 向量权限：为检索接口配置 admin/editor/viewer/reader 角色权限
- 备份恢复：创建备份任务、触发恢复、失败任务重试、策略化保留清理
- 向量审计：按资源类型/资源 ID/动作筛选审计日志
- 向量可视化：按 collection 采样并展示二维散点图

实操建议：先在“向量集合”完成 collection 与导入，再在“向量检索/可视化”验证数据质量，最后启用“监控 + 告警 + 权限 + 备份恢复”作为生产防护。

---

## 常见场景示例

### 场景1：智能客服机器人

```python
def chat_with_customer(user_message):
    response = client.chat.completions.create(
        model="gpt-3.5-turbo",
        messages=[
            {"role": "system", "content": "你是一个友好的客服助手"},
            {"role": "user", "content": user_message}
        ],
        temperature=0.7
    )
    return response.choices[0].message.content
```

### 场景2：文档摘要

```python
def summarize_document(document):
    response = client.chat.completions.create(
        model="gpt-4",
        messages=[
            {"role": "system", "content": "请用简洁的语言总结以下文档"},
            {"role": "user", "content": document}
        ],
        max_tokens=500
    )
    return response.choices[0].message.content
```

### 场景3：流式输出（打字机效果）

```python
stream = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "讲个故事"}],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

---

## 常见问题

### Q1：API Key从哪里获取？

**OpenAI**：
1. 访问 https://platform.openai.com
2. 注册/登录账号
3. 点击右上角头像 → View API Keys
4. 点击 "Create new secret key"

**Claude（Anthropic）**：
1. 访问 https://console.anthropic.com
2. 注册/登录账号
3. 点击 "API Keys"
4. 点击 "Create Key"

### Q2：如何查看我的使用量？

**方法1：Web控制台**
1. 访问 http://localhost:8080
2. 点击"用量监控"菜单

**方法2：API查询**
```bash
curl http://localhost:8080/api/v1/usage \
  -H "Authorization: Bearer YOUR_KEY"
```

### Q3：为什么请求返回401错误？

可能原因：
1. API Key未设置或错误
2. Authorization header格式不正确

解决方法：
```bash
# 确保header格式正确
-H "Authorization: Bearer YOUR_API_KEY"
```

### Q4：如何切换不同的AI服务商？

无需修改代码！只需：
1. 在配置文件中添加新的服务商API Key
2. 在请求中指定模型名称
3. 网关会自动路由到对应的服务商

### Q5：支持哪些编程语言？

所有语言都可以，只要能发送HTTP请求：
- Python（推荐）
- JavaScript/TypeScript
- Java
- Go
- C#
- PHP
- Ruby
- ...

---

## 故障排查

### 问题1：服务启动失败

**症状**：`docker-compose up` 报错

**排查步骤**：
```bash
# 1. 检查端口是否被占用
lsof -i :8080

# 2. 查看详细错误日志
docker-compose logs

# 3. 清理后重新启动
docker-compose down
docker-compose up -d
```

### 问题2：请求超时

**症状**：API请求一直等待，最后返回超时

**可能原因**：
1. 后端AI服务商响应慢
2. 网络连接问题
3. 请求体过大

**解决方法**：
```python
# 增加超时时间
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-key",
    timeout=60.0  # 60秒超时
)
```

### 问题3：返回429错误

**症状**：`{"error": "rate_limit_exceeded"}`

**原因**：超过了速率限制

**解决方法**：
1. 检查用量是否超限
2. 在配置文件中调整限额
3. 添加重试逻辑

```python
import time
import openai

def retry_request(func, max_retries=3):
    for i in range(max_retries):
        try:
            return func()
        except openai.RateLimitError:
            if i < max_retries - 1:
                time.sleep(2 ** i)  # 指数退避
                continue
            raise
```

### 问题4：中文乱码

**症状**：返回的中文显示为乱码

**解决方法**：
```python
# 确保使用UTF-8编码
response = requests.post(
    url,
    headers={"Content-Type": "application/json; charset=utf-8"},
    json=data
)
```

---

## 获取帮助

- **文档**：查看 `/docs` 目录下的其他文档
- **Issue**：在GitHub提交问题
- **社区**：加入我们的Discord社区

---

## 附录：术语表

| 术语 | 解释 |
|------|------|
| API | 应用程序接口，让程序之间互相通信 |
| API Key | 访问API的密钥，类似于密码 |
| Token | AI处理文本的基本单位，约等于0.75个英文单词 |
| 流式输出 | 像打字机一样逐字显示结果 |
| 速率限制 | 限制单位时间内的请求次数 |
| 网关 | 统一入口，负责转发请求 |

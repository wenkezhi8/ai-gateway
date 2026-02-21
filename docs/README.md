# AI Gateway 项目文档

## 项目概述

AI多服务商/账号智能中转网关是一个统一管理多个AI服务商API的智能网关系统。

### 核心特性

- **多服务商支持**: 统一接口支持 OpenAI、Anthropic、Azure OpenAI 等
- **智能路由**: 基于成本、可用性、配额自动选择最优服务商
- **智能限流**: 用户级和全局级的请求和Token限额控制
- **智能缓存**: 减少重复请求，降低成本
- **监控仪表盘**: 实时监控使用量和成本
- **一键部署**: Docker Compose 一键启动所有服务

## 目录结构

```
ai-gateway/
├── gateway/                 # Go 后端网关核心
│   ├── cmd/server/         # 应用入口
│   ├── internal/           # 内部包
│   │   ├── config/        # 配置管理
│   │   ├── handler/       # HTTP 处理器
│   │   ├── middleware/    # 中间件
│   │   ├── model/         # 数据模型
│   │   ├── proxy/         # 代理逻辑
│   │   ├── router/        # 路由设置
│   │   ├── service/       # 业务逻辑
│   │   └── utils/         # 工具函数
│   └── pkg/               # 公共包
│       ├── cache/         # 缓存实现
│       └── logger/        # 日志模块
│
├── console/                # Vue3 前端控制台
│   ├── src/
│   │   ├── components/   # 组件
│   │   ├── views/        # 页面
│   │   ├── router/       # 路由
│   │   ├── store/        # 状态管理
│   │   ├── api/          # API 调用
│   │   └── assets/       # 静态资源
│   └── public/           # 公共文件
│
├── deploy/                 # 部署配置
│   ├── docker/           # Docker 配置
│   │   ├── docker-compose.yml
│   │   └── Dockerfile
│   └── nginx/            # Nginx 配置
│       └── nginx.conf
│
├── docs/                   # 文档
├── scripts/                # 脚本
│   ├── start.sh          # 开发启动
│   └── docker.sh         # Docker 部署
│
└── configs/               # 配置文件
    └── config.example.yaml
```

## 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **数据库**: SQLite (GORM)
- **缓存**: Redis (可选)
- **日志**: Logrus

### 前端
- **框架**: Vue 3
- **UI 库**: Element Plus
- **状态管理**: Pinia
- **构建工具**: Vite
- **图表**: ECharts

### 部署
- **容器化**: Docker + Docker Compose
- **反向代理**: Nginx

## 快速开始

### 本地开发

```bash
# 1. 复制配置文件
cp configs/config.example.yaml configs/config.yaml

# 2. 编辑配置，添加 API Key
vim configs/config.yaml

# 3. 启动服务
./scripts/start.sh
```

### Docker 部署

```bash
# 构建并启动
./scripts/docker.sh build
./scripts/docker.sh up

# 查看日志
./scripts/docker.sh logs

# 停止服务
./scripts/docker.sh down
```

## API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/api/v1/chat/completions` | POST | 聊天补全 (OpenAI 兼容) |
| `/api/v1/completions` | POST | 文本补全 |
| `/api/v1/embeddings` | POST | 向量嵌入 |
| `/api/v1/providers` | GET/POST | 服务商管理 |
| `/api/v1/users` | GET/POST | 用户管理 |
| `/api/v1/keys` | GET/POST | API 密钥管理 |
| `/api/v1/stats` | GET | 统计数据 |

## 环境变量

| 变量 | 描述 |
|------|------|
| `AIG_SERVER_PORT` | 服务端口 |
| `AIG_REDIS_HOST` | Redis 主机 |
| `AIG_REDIS_PORT` | Redis 端口 |
| `OPENAI_API_KEY` | OpenAI API Key |
| `ANTHROPIC_API_KEY` | Anthropic API Key |
| `AZURE_OPENAI_API_KEY` | Azure OpenAI Key |

## 许可证

MIT License

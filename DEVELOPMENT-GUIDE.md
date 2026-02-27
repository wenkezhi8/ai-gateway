# AI Gateway 开发指南

本文档为 ai-gateway 项目的开发者提供详细的开发指导。

## 目录

- [项目概述](#项目概述)
- [开发环境搭建](#开发环境搭建)
- [代码结构](#代码结构)
- [常见问题](#常见问题)
- [贡献指南](#贡献指南)

---

## 项目概述

ai-gateway 是一个统一的 AI 服务提供商 API 网关，提供智能限流、缓存和路由功能。

### 核心特性

- **多提供商支持**: 统一接口支持 OpenAI、Anthropic、Azure OpenAI 等
- **智能限流**: 基于用户和全局的可配置配额限流
- **智能缓存**: 响应缓存以降低 API 成本并改善延迟
- **灵活路由**: 根据模型、成本或可用性将请求路由到不同提供商
- **RESTful API**: OpenAI 兼容 API，易于集成
- **Web 仪表盘**: 内置管理控制台用于监控和配置
- **Docker 就绪**: 使用 Docker Compose 一键部署

---

## 开发环境搭建

### 前置要求

- Go 1.21 或更高版本
- Node.js 18+ (前端开发)
- Redis (可选，用于分布式缓存)
- Docker (可选，用于容器化部署)

### 后端开发

1. 克隆仓库:
   ```bash
   git clone https://github.com/wenkezhi8/ai-gateway.git
   cd ai-gateway
   ```

2. 安装依赖:
   ```bash
   make deps
   ```

3. 复制示例配置:
   ```bash
   cp configs/config.example.json configs/config.json
   ```

4. 编辑 `configs/config.json` 并添加你的 API 密钥

5. 运行应用:
   ```bash
   make run
   ```

### 前端开发

1. 进入 web 目录:
   ```bash
   cd web
   ```

2. 安装依赖:
   ```bash
   npm install
   ```

3. 启动开发服务器:
   ```bash
   npm run dev
   ```

4. 构建生产版本:
   ```bash
   npm run build
   ```

---

## 代码结构

```
ai-gateway/
├── cmd/
│   └── gateway/          # 应用入口点
│       └── main.go
├── internal/
│   ├── config/           # 配置管理
│   ├── handler/          # HTTP 处理器
│   ├── middleware/       # HTTP 中间件
│   ├── router/           # 路由设置
│   ├── provider/         # AI 提供商适配器
│   ├── limiter/          # 限流
│   ├── cache/            # 响应缓存
│   ├── routing/          # 智能路由
│   └── constants/        # 常量定义
├── pkg/                  # 公共包
├── configs/              # 配置文件
├── scripts/              # 工具脚本
├── web/                  # Web 仪表盘
│   ├── src/
│   │   ├── components/   # Vue 组件
│   │   ├── views/        # 页面视图
│   │   ├── store/        # 状态管理
│   │   └── constants/    # 常量定义
│   └── public/           # 静态资源
├── tests/                # 测试文件
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

---

## 常见问题

### Q: 如何添加新的 AI 提供商？

A: 在 `internal/provider/` 目录下创建新的提供商适配器，实现 Provider 接口，然后在 ProviderRegistry 中注册。

### Q: 如何配置智能路由？

A: 编辑 `configs/config.json` 中的路由配置，或使用 Web 仪表盘的路由管理页面进行配置。

### Q: 测试失败怎么办？

A: 确保所有依赖已安装，运行 `make deps` 更新依赖，然后使用 `make test` 重新运行测试。

---

## 贡献指南

我们欢迎所有形式的贡献！

### 开发流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

### 代码规范

- 后端: 遵循 Go 官方代码规范，使用 `gofmt` 格式化
- 前端: 遵循 ESLint 和 Prettier 配置
- 提交信息: 使用约定式提交格式

### 测试要求

- 所有新功能都应包含相应的测试
- 确保 `make test` 通过率 100%
- 提交前运行 `make lint` 检查代码质量

---

## 更多资源

- [README.md](README.md) - 项目主文档
- [DEPLOYMENT.md](DEPLOYMENT.md) - 部署指南
- [CONTRIBUTING.md](CONTRIBUTING.md) - 贡献指南
- [API 文档](docs/API.md) - API 接口文档

---

*最后更新: 2026-02-28*

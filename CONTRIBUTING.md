# Contributing to AI Gateway

感谢您有兴趣为 AI Gateway 做贡献！

## 开发环境设置

### 前置要求

- Go 1.21+
- Node.js 18+ & npm
- Docker & Docker Compose (可选)
- Make

### 快速开始

```bash
# 克隆仓库
git clone https://github.com/wenkezhi8/ai-gateway.git
cd ai-gateway

# 初始化项目
make setup

# 安装开发工具
make install-tools

# 运行开发服务器
make run
```

## 项目结构

```
ai-gateway/
├── cmd/gateway/          # 应用入口
├── internal/             # 内部包
│   ├── config/           # 配置管理
│   ├── handler/          # HTTP handlers
│   ├── middleware/       # HTTP 中间件
│   ├── provider/         # AI 服务商适配器
│   ├── limiter/          # 限流器
│   ├── cache/            # 缓存
│   ├── router/           # 路由配置
│   └── ...
├── pkg/                  # 公共包
├── web/                  # 前端 (Vue 3)
├── configs/              # 配置文件
├── tests/                # 测试
└── scripts/              # 工具脚本
```

## 开发工作流

### 1. 创建分支

```bash
git checkout -b feature/your-feature-name
# 或
git checkout -b fix/your-bug-fix
```

### 2. 进行更改

确保遵循代码规范：

**Go 代码规范：**
```bash
# 格式化
make fmt

# Lint 检查
make lint

# 运行测试
make test
```

**前端代码规范：**
```bash
cd web

# Lint 检查
npm run lint

# 格式化
npm run format

# 类型检查
npm run typecheck
```

### 3. 提交代码

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
feat: 添加新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式（不影响逻辑）
refactor: 重构
test: 测试相关
chore: 构建/工具相关
```

示例：
```
feat: 添加支持 MiniMax 服务商
fix: 修复 JWT token 过期时间计算错误
docs: 更新部署文档
```

### 4. 推送并创建 PR

```bash
git push origin feature/your-feature-name
```

然后在 GitHub 上创建 Pull Request。

## 代码规范

### Go 规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go)
- 使用 `gofmt` 格式化代码
- 添加必要的注释和文档
- 编写单元测试

### 前端规范

- 使用 TypeScript
- 遵循 Vue 3 Composition API 风格
- 使用 ESLint + Prettier

## 测试

### 后端测试

```bash
# 运行所有测试
make test

# 运行带覆盖率的测试
make test-coverage

# 运行特定测试
go test -v -run TestFunctionName ./path/to/package
```

### 前端测试

```bash
cd web

# 运行 E2E 测试
npm run test

# 运行特定测试
npm run test:auth
```

## 提交前检查清单

- [ ] 代码通过 `make lint` 检查
- [ ] 测试通过 `make test`
- [ ] 前端通过 `npm run lint` 和 `npm run typecheck`
- [ ] 更新了相关文档
- [ ] 提交信息遵循 Conventional Commits

## 发布流程

1. 更新 `CHANGELOG.md`
2. 创建版本 tag: `git tag v1.x.x`
3. 推送 tag: `git push origin v1.x.x`
4. GitHub Actions 自动构建和发布

## 问题反馈

- 使用 GitHub Issues 报告 bug
- 提供复现步骤和环境信息
- 标注适当的 label

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

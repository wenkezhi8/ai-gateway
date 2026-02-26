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

### 1. 先对齐执行规范

- 开始开发前先阅读 `AGENTS.md`
- AI 协作默认采用「先 Plan、后 Execute」
- 默认不执行 `commit/push/tag`，除非需求中明确授权

会话前建议直接使用流程卡：

```text
【本次流程卡】
目标：<一句话目标>
改动范围：<允许修改的文件或目录>
执行模式：先Plan，后Execute（我回复“开始执行”再改代码）
必跑验证：<命令1>、<命令2>、<命令3>
输出顺序：根因 -> 方案 -> 改动清单 -> 测试结果 -> 风险/回滚 -> 接口一致性 -> 版本建议
Git权限：<是否允许 commit/push/tag>
版本策略：以最新 git tag 为准，同步更新 CHANGELOG.md 与 AGENTS.md
```

### 2. 分支策略

```bash
# 当前项目默认：main 直接开发
git checkout main
git pull origin main
```

如需协作评审或外部贡献，可按需创建 `feature/*`、`fix/*` 分支并发起 PR。

### 3. 进行更改

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

### 4. 提交代码

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

### 5. 推送并创建 PR（按需）

```bash
# main 直开
git push origin main

# 或分支协作
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

### 版本号真相源

- 版本唯一真相源：`git tag`
- `CHANGELOG.md` 与 `AGENTS.md` 是同步展示，不是版本真相源

### 发布步骤（SOP）

1. 获取最新版本 tag：

```bash
git fetch --tags
git describe --tags --abbrev=0
```

2. 按语义化规则计算下个版本：
   - `PATCH`：向下兼容 bug 修复
   - `MINOR`：向下兼容新功能
   - `MAJOR`：不兼容变更

3. 同步文档：更新 `CHANGELOG.md` 与 `AGENTS.md`

4. 发布（示例）：

```bash
git add -A
git commit -m "chore(release): vX.Y.Z"
git tag -a vX.Y.Z -m "release: vX.Y.Z"
git push origin main
git push origin vX.Y.Z
```

5. 发布后核验：

```bash
git describe --tags --abbrev=0
git show vX.Y.Z --name-only --oneline
```

## 问题反馈

- 使用 GitHub Issues 报告 bug
- 提供复现步骤和环境信息
- 标注适当的 label

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

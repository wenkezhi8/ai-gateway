# AI Gateway - 企业级AI开发修复 标准工作流程

> 本文档定义 AI Gateway 项目的开发规范、工作流程与输出要求，所有AI协作环节必须严格遵循。
### 全栈开发问题排查与修复标准工作流程

#### 一、标准工作流程（无负责人版本，AI执行需按步骤流转）

| 步骤 | 流程 | 具体工作要求 |
| ------ |------ |------ |
| **1** | **问题排查测试** | 读取全量代码、运行现有测试用例、发现并完整列出所有Bug |
| **2** | **修复方案讨论** | 确认问题是否需要修复、输出可落地的修复方案、排期评估、确定任务节点 |
| **3** | **代码修复** | 严格按照讨论通过的方案执行代码修改，不改动非相关逻辑 |
| **4** | **回归验证** | 测试验证修复效果、检查是否引入新的功能异常或兼容性问题 |
| **5** | **合规审计** | 代码规范审查、确认修复内容符合项目编码规范、结果终审确认 |
| **6** | **复盘归档** | 输出根因分析报告、总结规避方案、沉淀经验避免同类问题重复出现 |

---

#### 二、角色设定

你是拥有10年以上经验的资深全栈开发工程师，精通各类编程语言、前后端框架、系统架构，具备极强的问题排查和代码优化能力，每次处理开发/修复任务前必须先读取本规范约束执行逻辑。

#### 三、输出强制要求

处理代码相关问题时必须严格按以下顺序输出，禁止调整顺序，禁止输出任何无关内容：

1. **问题根因**：准确定位问题，说明清晰可复现的根因。
2. **修复方案**：兼容全局功能，不影响原有正常逻辑。
3. **修复后代码**：提供完整可运行代码，所有改动点用注释明确标注。
4. **测试用例**：提供可直接运行的用例，覆盖故障原场景及所有边界情况。
5. **测试执行说明**：运行测试脚本，它将启动一个浏览器并模拟用户操作。测试地址：`http://localhost:8566/` 账号：`admin` 密码：`admin123`。
6. **优化建议**：给出2-5个无侵入、不影响原有业务的低风险可落地优化建议。
7. **接口一致性校验**：若涉及接口调用，必须校验前后端接口定义一致性，确保入参、出参、协议、请求方式完全对齐。
8. **版本号**：修复完成后按`X.Y.Z`规则提供版本号（X=大版本迭代，Y=功能新增，Z=问题修复）。
9. 默认每次修复后自动提交并推送 main   commit + push main
  
---

#### 问题提交区

（请在此处粘贴问题描述、相关代码、报错信息、接口定义等所有必要信息）

`<粘贴你的问题/代码/报错信息>`


## 目录

- [快速参考](#快速参考)
- [常量配置](#常量配置)
- [代码风格规范](#代码风格规范)
- [Git 工作流](#git-工作流)
- [代码审查流程](#代码审查流程)
- [测试规范](#测试规范)
- [API 规范](#api-规范)
- [安全规范](#安全规范)
- [文档规范](#文档规范)
- [部署规范](#部署规范)
- [常见问题](#常见问题)

---

## 快速参考

### 常用命令

```bash
# 后端
make lint              # 代码检查
make test              # 运行测试
make test-coverage     # 测试覆盖率
make build             # 构建

# 前端
cd web && npm run lint       # 代码检查
cd web && npm run typecheck  # 类型检查
cd web && npm run build      # 构建

# 完整 CI
make ci-local          # 本地运行所有检查
```

### 端口配置

| 服务 | 端口 |
|------|------|
| 前端/后端 | 8566 |
| Metrics | 9090 |

---

## 常量配置

### 前端常量

| 文件 | 内容 |
|------|------|
| `web/src/constants/api.ts` | API 路径、端口号 |

### 后端常量

| 文件 | 内容 |
|------|------|
| `internal/constants/routes.go` | API 路径 |
| `internal/constants/config.go` | 端口号 |

### 修改配置流程

1. 修改对应常量文件
2. 全局搜索旧值，更新所有引用
3. 更新相关测试
4. 运行 `make ci-local` 验证

---

## 代码风格规范

### Go 代码规范

#### 命名规范

```go
// ✅ 正确
type AccountManager struct { ... }
func (m *AccountManager) GetAccount(id string) (*Account, error) { ... }
const MaxRetryCount = 3
var defaultTimeout = 30 * time.Second

// ❌ 错误
type account_manager struct { ... }
func (m *AccountManager) get_account(id string) (*Account, error) { ... }
```

#### 错误处理

```go
// ✅ 正确：包装错误提供上下文
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// ✅ 正确：自定义错误类型
type ValidationError struct {
    Field   string
    Message string
}

// ❌ 错误：忽略错误
doSomething()

// ❌ 错误：无上下文的错误
if err != nil {
    return err
}
```

#### Context 传递

```go
// ✅ 正确：context 作为第一个参数
func (s *Service) Process(ctx context.Context, id string) error { ... }

// ❌ 错误
func (s *Service) Process(id string, ctx context.Context) error { ... }
```

#### 接口定义

```go
// ✅ 正确：接口在使用方定义
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// ✅ 正确：小接口
type Reader interface {
    Read(p []byte) (n int, err error)
}

// ❌ 错误：大而全的接口
type Service interface {
    Create() error
    Update() error
    Delete() error
    Get() error
    List() error
    // ... 10+ methods
}
```

#### 并发安全

```go
// ✅ 正确：使用 mutex 保护共享资源
type SafeCounter struct {
    mu    sync.RWMutex
    count map[string]int
}

func (c *SafeCounter) Increment(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count[key]++
}

// ❌ 错误：无保护的并发访问
type UnsafeCounter struct {
    count map[string]int
}
```

### 前端代码规范

#### 命名规范

```typescript
// ✅ 正确
interface UserInfo { ... }
type RequestStatus = 'pending' | 'success' | 'error'
const MAX_RETRY_COUNT = 3
function fetchUserData() { ... }
const userModel = reactive({ ... })

// ❌ 错误
interface userInfo { ... }
type request_status = ...
```

#### Vue 组件规范

```vue
<!-- ✅ 正确：组件命名 PascalCase -->
<template>
  <UserProfile :user="currentUser" @update="handleUpdate" />
</template>

<script setup lang="ts">
// 使用 Composition API
import { ref, computed, onMounted } from 'vue'

interface Props {
  userId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  update: [user: User]
}>()

// 响应式状态
const loading = ref(false)
const data = ref<User | null>(null)

// 计算属性
const displayName = computed(() => data.value?.name ?? 'Unknown')

// 生命周期
onMounted(async () => {
  await fetchData()
})
</script>
```

#### API 调用规范

```typescript
// ✅ 正确：使用常量和类型
import { API } from '@/constants/api'
import type { ChatRequest, ChatResponse } from '@/api/types'

async function sendChat(request: ChatRequest): Promise<ChatResponse> {
  const response = await fetch(API.V1.CHAT_COMPLETIONS, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request)
  })
  
  if (!response.ok) {
    throw new Error(`Chat failed: ${response.status}`)
  }
  
  return response.json()
}

// ❌ 错误：硬编码路径
fetch('/api/v1/chat/completions', { ... })
```

#### 类型定义

```typescript
// ✅ 正确：明确的类型定义
interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

type Provider = {
  name: string
  models: string[]
  enabled: boolean
}

// ❌ 错误：使用 any
function process(data: any) { ... }
```

---

## Git 工作流

### 当前阶段分支策略（立即生效）

> 当前项目未上线，统一采用 **main 分支直接开发**，不强制 feature 分支。

#### main 分支直开规则

1. 所有开发、修复、文档变更默认在 `main` 进行。
2. 每完成一个可运行的小功能（建议 10-60 分钟粒度）立即提交。
3. 每次提交后立即推送：`git push origin main`。
4. 每日收工前必须执行：`git status`，确保工作区为 clean。
5. 临时未完成改动也必须做 checkpoint 提交（`wip`），禁止跨天只留未提交工作区。

### 分支命名

| 类型 | 格式 | 示例 |
|------|------|------|
| 功能 | `feature/描述` | `feature/add-deepseek-provider` |
| 修复 | `fix/描述` | `fix/jwt-token-expiry` |
| 重构 | `refactor/描述` | `refactor/cache-layer` |
| 文档 | `docs/描述` | `docs/api-reference` |
| 发布 | `release/版本` | `release/v1.2.0` |

### 提交信息规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

#### Type 类型

| Type | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat: 添加 MiniMax 服务商支持` |
| `fix` | Bug 修复 | `fix: 修复 JWT token 过期计算错误` |
| `docs` | 文档更新 | `docs: 更新部署文档` |
| `style` | 代码格式 | `style: 格式化代码` |
| `refactor` | 重构 | `refactor: 优化缓存层实现` |
| `test` | 测试 | `test: 添加限流器单元测试` |
| `chore` | 构建/工具 | `chore: 更新 CI 配置` |
| `perf` | 性能优化 | `perf: 优化响应缓存策略` |
| `ci` | CI 配置 | `ci: 添加安全扫描步骤` |

#### 提交示例

```bash
# ✅ 正确
git commit -m "feat(provider): 添加 MiniMax 服务商适配器"
git commit -m "fix(limiter): 修复并发请求下的竞态条件"
git commit -m "docs: 更新 CONTRIBUTING.md"

# ❌ 错误
git commit -m "fix bug"
git commit -m "update"
git commit -m "WIP"
```

### 工作流程

```bash
# 1. 确认主分支
git checkout main
git pull origin main

# 2. 开发并提交（小步快跑）
git add -A
git commit -m "fix(scope): 变更说明"

# 3. 立即推送，避免本地堆积
git push origin main

# 4. 收工检查（必须 clean）
git status
```

### 版本号规范

遵循 [语义化版本](https://semver.org/lang/zh-CN/)：`MAJOR.MINOR.PATCH`

| 版本类型 | 说明 | 示例 |
|----------|------|------|
| MAJOR | 不兼容的 API 变更 | 1.0.0 → 2.0.0 |
| MINOR | 向下兼容的新功能 | 1.1.0 → 1.2.0 |
| PATCH | 向下兼容的问题修复 | 1.1.0 → 1.1.1 |

**当前版本**: `v1.4.5`

### 强制提交规范

> ⚠️ **重要**：每次开发/修复后必须立即提交，避免代码丢失
> ⚠️ **会话约束**：每次与 AI 的独立会话结束前，必须将代码合并到 `main` 并执行 `git push origin main`（网络异常时需在会话结论中明确说明未推送原因）

```bash
# 每完成一个小功能/修复就提交（10-60 分钟粒度）
git add -A
git commit -m "feat(xxx): 功能描述"

# 每次提交后立即推送到 main
git push origin main

# 下班前再次推送并确认工作区 clean
git push origin main
git status

# 开发中的代码也要提交（防止丢失）
git add -A
git commit -m "wip: checkpoint"
git push origin main
```

### 自动化脚本

```bash
# 添加到 crontab，每天 0 点自动推送
# crontab -e
# 0 0 * * * cd /path/to/ai-gateway && git push origin main >> /tmp/git-push.log 2>&1

# 可选：每 2 小时自动做一次本地 checkpoint（防止意外关闭导致丢失）
# 0 */2 * * * cd /path/to/ai-gateway && git add -A && git commit -m "wip: auto-checkpoint $(date +\%F-\%H\%M)" >> /tmp/git-checkpoint.log 2>&1
```

---

## 历史教训

### 2026-02-23 开发后页面"请求的资源不存在"

**问题描述**：每次开发完成后访问页面显示"请求的资源不存在"

**原因分析**：
1. 前端代码修改后没有重新构建
2. 构建后没有重启后端服务
3. 浏览器缓存了旧的静态文件

**解决方案**：
1. 每次修改前端代码后执行 `npm run build`
2. 每次修改后端代码后重启服务
3. 使用强制刷新清除浏览器缓存

**改进措施**：
- ✅ 使用统一的重启脚本 `./scripts/dev-restart.sh`
- ✅ 开发完成后按顺序执行：
  ```bash
  # 1. 构建前端
  cd web && npm run build
  
  # 2. 重启后端
  lsof -ti:8566 | xargs kill -9
  cd .. && go run ./cmd/gateway &
  
  # 3. 强制刷新浏览器 (Cmd+Shift+R)
  ```

### 前端变更生效防漏流程（强制）

> 目的：避免“代码已改但页面无变化 / 资源不存在 / 旧静态资源缓存”再次发生。

#### 标准执行顺序（每次前端改动后必须执行）

```bash
# 1) 前端类型检查（快速发现构建阻断）
cd web && npm run typecheck

# 2) 前端构建（生成最新 dist）
cd web && npm run build

# 3) 统一重启（包含 Redis 检查、进程清理、服务重启）
cd .. && ./scripts/dev-restart.sh
```

#### 浏览器验证（必须）

1. 打开 `http://localhost:8566/` 并强制刷新（Mac: `Cmd+Shift+R`，Windows: `Ctrl+Shift+R`）。
2. 打开本次改动对应页面路径（例如 `/cache`、`/routing`）确认 UI/交互已生效。
3. 若仍未生效，先执行一次“无痕窗口访问”再判断是否代码问题。

#### 失败排查顺序（必须按序）

1. **先看构建是否成功**：`npm run build` 是否有 TS/打包报错。
2. **再看服务是否重启成功**：`./scripts/dev-restart.sh` 输出是否显示 `服务启动成功`。
3. **再看端口是否正确**：确认访问端口固定为 `8566`。
4. **最后看缓存问题**：浏览器强刷或无痕访问，避免旧资源干扰。

#### 提交前验收口径（前端改动必做）

1. 本地构建成功（`web npm run build` 通过）。
2. 页面功能可见且可操作（至少手工点击 1 次核心功能）。
3. `git status` 中仅包含预期改动，无临时调试垃圾文件。

### 2026-02-23 代码丢失事件

**问题描述**：凌晨开发的功能代码丢失，只能重新实现

**原因分析**：
1. 开发完成后未提交到 git
2. 执行了 `git checkout` 或 IDE 的 "Discard Changes" 覆盖了工作区
3. 没有 git stash 备份

**解决方案**：
1. 每完成一个功能立即 `git commit`
2. 开发中的代码也要 `git commit -m "WIP"`
3. 每天 0 点自动推送到远程

**改进措施**：
- ✅ 每次修复/开发后立即提交到本地仓库
- ✅ 每天推送到 GitHub 远程仓库
- ✅ 重要修改前先 `git stash` 或 `git commit`
- ✅ 禁止执行 `git checkout .` 或 `git restore` 除非确认修改可丢弃

---

## 代码审查流程

### 提交 PR 前检查清单

- [ ] 代码通过 `make lint`
- [ ] 测试通过 `make test`
- [ ] 新代码有对应测试
- [ ] 文档已更新
- [ ] 提交信息符合规范
- [ ] 无敏感信息泄露

### PR 标题格式

```
<type>: <description>
```

示例：
- `feat: 添加 DeepSeek 服务商支持`
- `fix: 修复缓存过期时间计算`

### Code Review 标准

1. **正确性**：代码逻辑正确，无 bug
2. **可读性**：命名清晰，结构合理
3. **可维护性**：模块化，低耦合
4. **性能**：无明显的性能问题
5. **安全**：无安全漏洞
6. **测试**：有充分的测试覆盖

---

## 测试规范

### 后端测试

#### 测试命名

```go
// ✅ 正确
func TestAccountManager_GetAccount_Success(t *testing.T) { ... }
func TestAccountManager_GetAccount_NotFound(t *testing.T) { ... }
func TestAccountManager_GetAccount_InvalidID(t *testing.T) { ... }

// ❌ 错误
func TestGetAccount(t *testing.T) { ... }
```

#### 表格驱动测试

```go
// ✅ 正确
func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        wantErr bool
    }{
        {
            name:    "valid config",
            config:  &Config{Port: "8080"},
            wantErr: false,
        },
        {
            name:    "empty port",
            config:  &Config{Port: ""},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### Mock 使用

```go
// ✅ 正确：使用 testify mock
type MockCache struct {
    mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
    args := m.Called(ctx, key)
    return args.Get(0).([]byte), args.Error(1)
}

func TestService_WithMock(t *testing.T) {
    cache := new(MockCache)
    cache.On("Get", mock.Anything, "key").Return([]byte("value"), nil)
    
    service := NewService(cache)
    // ...
    
    cache.AssertExpectations(t)
}
```

### 前端测试

#### 组件测试

```typescript
// ✅ 正确
describe('UserProfile', () => {
  it('should display user name', async () => {
    const wrapper = mount(UserProfile, {
      props: { user: { name: 'Test User' } }
    })
    
    expect(wrapper.text()).toContain('Test User')
  })
  
  it('should emit update event on click', async () => {
    const wrapper = mount(UserProfile)
    await wrapper.find('button').trigger('click')
    
    expect(wrapper.emitted('update')).toBeTruthy()
  })
})
```

### 测试覆盖率要求

| 模块 | 最低覆盖率 |
|------|-----------|
| 核心业务逻辑 | 80% |
| API Handler | 70% |
| 工具函数 | 90% |
| 中间件 | 70% |

---

## API 规范

### RESTful API 设计

| 操作 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 列表 | GET | `/api/v1/resources` | 获取资源列表 |
| 详情 | GET | `/api/v1/resources/:id` | 获取单个资源 |
| 创建 | POST | `/api/v1/resources` | 创建资源 |
| 更新 | PUT | `/api/v1/resources/:id` | 全量更新 |
| 更新 | PATCH | `/api/v1/resources/:id` | 部分更新 |
| 删除 | DELETE | `/api/v1/resources/:id` | 删除资源 |

### 统一响应格式

```json
// 成功响应
{
  "code": 0,
  "message": "success",
  "data": { ... }
}

// 错误响应
{
  "code": 1001,
  "message": "Invalid parameter",
  "error": "Detailed error message"
}
```

### 错误码定义

| 范围 | 类型 |
|------|------|
| 0 | 成功 |
| 1000-1999 | 参数错误 |
| 2000-2999 | 认证/授权错误 |
| 3000-3999 | 业务逻辑错误 |
| 5000-5999 | 服务端错误 |

### 统一接口

| 功能 | 路径 |
|------|------|
| 聊天补全 | `POST /api/v1/chat/completions` |
| Anthropic Messages 兼容 | `POST /api/anthropic/v1/messages` |
| 文本补全 | `POST /api/v1/completions` |
| 向量嵌入 | `POST /api/v1/embeddings` |
| 服务商列表 | `GET /api/v1/providers` |
| 模型列表 | `GET /api/v1/models` |

---

## 安全规范

### 敏感信息处理

```bash
# ✅ 正确：使用环境变量
export JWT_SECRET="your-secret"
export OPENAI_API_KEY="sk-xxx"

# ❌ 错误：硬编码
const apiKey = "sk-xxx"  // 绝对禁止!
```

### 环境变量清单

| 变量 | 说明 | 必需 |
|------|------|------|
| `JWT_SECRET` | JWT 密钥 | 生产必需 |
| `OPENAI_API_KEY` | OpenAI API Key | 可选 |
| `ANTHROPIC_API_KEY` | Anthropic API Key | 可选 |
| `REDIS_PASSWORD` | Redis 密码 | 可选 |
| `GIN_MODE` | 运行模式 | 可选 |

### 安全检查清单

- [ ] 无硬编码密钥
- [ ] API Key 在日志中脱敏
- [ ] 输入验证完整
- [ ] SQL 注入防护
- [ ] XSS 防护
- [ ] CSRF 防护
- [ ] Rate Limiting 启用

---

## 文档规范

### 代码注释

```go
// Package limiter provides rate limiting functionality for API requests.
// It supports multiple limiting strategies including token bucket and sliding window.
//
// Example usage:
//
//   limiter := limiter.New(config)
//   if !limiter.Allow(userID) {
//       return ErrRateLimitExceeded
//   }
package limiter

// AccountManager manages provider accounts and their quotas.
// It is safe for concurrent use.
type AccountManager struct {
    // ...
}

// GetAccount retrieves an account by its ID.
// Returns ErrAccountNotFound if the account does not exist.
func (m *AccountManager) GetAccount(id string) (*Account, error) {
    // ...
}
```

### README 结构

```markdown
# Project Name

Brief description

## Features

## Quick Start

## Installation

## Usage

## Configuration

## API Reference

## Contributing

## License
```

---

## 部署规范

### 构建验证

```bash
# 后端构建
go build -o bin/ai-gateway ./cmd/gateway

# 前端构建
cd web && npm run build

# Docker 构建
docker build -t ai-gateway:latest .
```

### 服务管理

```bash
# 启动服务
./bin/ai-gateway

# 健康检查
curl http://localhost:8566/health

# 查看日志
tail -f ai-gateway.log

# 重启服务
lsof -ti:8566 | xargs kill -HUP
```

---

## 常见问题

### Q: 修改端口后服务无法启动？

A: 确保同时更新：
1. `internal/constants/config.go`
2. `web/src/constants/api.ts`
3. `configs/config.json`

### Q: 测试失败怎么办？

A: 检查：
1. 环境变量是否设置
2. 测试数据是否更新
3. 依赖服务（如 Redis）是否运行

### Q: 如何添加新的 AI 服务商？

A: 
1. 在 `internal/provider/` 下创建适配器
2. 在 `internal/provider/registry.go` 注册
3. 在 `cmd/gateway/main.go` 添加工厂函数
4. 添加对应测试

---

## 持久化文件

| 文件 | 内容 |
|------|------|
| `data/accounts.json` | 账号配置 |
| `data/model_scores.json` | 模型评分 |
| `data/provider_defaults.json` | 服务商默认模型 |
| `data/api_keys.json` | API Keys |
| `data/router_config.json` | 路由配置 |
| `data/users.json` | 用户数据（密码等） |
| `data/feedback.json` | 反馈数据 |

### 缓存配置（生产环境）

**所有缓存默认使用 Redis**，配置方式：

```json
// configs/config.json
{
  "redis": {
    "host": "localhost",
    "port": 6379,
    "password": "",
    "db": 0
  }
}
```

或环境变量：

```bash
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=your_password
export REDIS_DB=0
```

> **注意**：如果 Redis 连接失败，会自动降级到内存缓存（重启丢失）。

---

## 开发规划

### 当前版本

**v1.4.4** (2026-02-26)

### 当前迭代 (v1.4)

| 任务 | 状态 | 说明 |
|------|------|------|
| 按任务类型管理缓存 | ✅ 完成 | 事实查询/代码生成/数学计算/日常对话/创意写作/逻辑推理/翻译/长文本处理/其他 - 分类展示 |
| 缓存内容详情查看 | ✅ 完成 | 查看用户消息和AI回复 |
| 按任务类型删除缓存 | ✅ 完成 | 删除单条或清空某类型所有缓存 |
| 缓存预热功能 | ✅ 完成 | 手动添加测试缓存条目 |
| 导出缓存数据 | ✅ 完成 | 导出缓存数据为 JSON |
| 缓存趋势图 | ✅ 完成 | 缓存命中/未命中趋势图表 |
| 缓存内容按任务类型分类 | ✅ 完成 | 按事实/代码/数学/对话/创意/推理/翻译/长文本分类显示 |
| Anthropic 兼容入口落地 | ✅ 完成 | 新增 `POST /api/anthropic/v1/messages`，支持非流式/流式、多模态、tool_use/tool_result 转换 |

### 上一迭代 (v1.3)

| 任务 | 状态 | 说明 |
|------|------|------|
| 运维监控页面 | ✅ 完成 | web/src/views/ops/index.vue - QPS/TPS/TTFT/SLA/分位数/资源监控/智能诊断 |
| CPU/内存监控 | ✅ 完成 | macOS/Linux 双平台支持，显示系统真实资源使用 |
| 缓存内容管理 | ✅ 完成 | GET/DELETE /api/admin/cache/entries - 查看、搜索、删除缓存条目 |
| 模型评分数据加载修复 | ✅ 完成 | 兼容数组格式响应 |
| 缓存空状态显示优化 | ✅ 完成 | 当缓存为空时显示友好提示 |

### 上一迭代 (v1.2)

| 任务 | 状态 | 说明 |
|------|------|------|
| 登录验证 | ✅ 完成 | 未登录跳转到 /login |
| 修改密码 | ✅ 完成 | 支持修改密码，持久化到 data/users.json |
| 修改用户名 | ✅ 完成 | PUT /auth/profile，支持修改用户名 |
| 模型删除持久化 | ✅ 完成 | 删除后重启不再恢复 |
| API Key 加密存储 | ✅ 完成 | pkg/crypto/encrypt.go |
| JWT 安全配置 | ✅ 完成 | pkg/security/config.go |
| 任务难度评估 | ✅ 完成 | internal/routing/difficulty.go - 基于长度/复杂度/历史成功率 |
| 级联路由策略 | ✅ 完成 | internal/routing/cascade.go - 小模型→大模型逐级升路 |
| 难度评估集成 | ✅ 完成 | SmartRouter 集成 DifficultyAssessor 和 CascadeRouter |
| 缓存与路由联动 | ✅ 完成 | proxy.go 集成请求去重和按任务类型 TTL 缓存 |
| 语义缓存 | ✅ 完成 | internal/cache/semantic.go - 向量相似度匹配，相似请求复用 |
| 效果评估闭环 | ✅ 完成 | internal/routing/feedback.go - 自动收集反馈，迭代优化路由规则 |
| 反馈 API | ✅ 完成 | internal/handler/admin/feedback.go - 反馈提交、性能查询、优化触发 |
| 路由策略 UI | ✅ 完成 | web/src/views/routing/index.vue - 智能路由配置、模型评分、反馈统计 |
| 缓存管理 UI | ✅ 完成 | web/src/views/cache/index.vue - Redis状态、请求去重、语义缓存配置 |
| 前后端 API 统一 | ✅ 完成 | 所有页面调用真实 API，移除模拟数据 |
| 环境变量配置 | ✅ 完成 | API Key 改用环境变量，移除硬编码 |
| 任务类型分布统计 | ✅ 完成 | GET /api/admin/feedback/task-type-distribution |
| TTL 配置 API | ✅ 完成 | GET/PUT /api/admin/router/ttl-config - 按任务类型配置 TTL |
| 缓存质量校验 | ✅ 完成 | QualityChecker 接口 + DefaultQualityChecker 实现 |
| 级联路由配置 API | ✅ 完成 | CRUD /api/admin/router/cascade-rules |
| API 常量完善 | ✅ 完成 | web/src/constants/api.ts 包含所有端点 |

### 测试覆盖率

| 模块 | 覆盖率 |
|------|--------|
| internal/routing | 54.1% |
| internal/cache | 30.2% |
| pkg/crypto | 81.8% |
| pkg/security | 62.5% |
| internal/metrics | 98.2% |
| internal/provider | 43.1% |

### 待开发

| 任务 | 优先级 | 说明 |
|------|--------|------|
| SQLite 持久化 | 低 | 可选，替代 JSON 文件存储 |
| 单元测试补充 | 中 | 提升核心模块覆盖率至 80% |

### 已知问题

| 问题 | 状态 | 解决方案 |
|------|------|---------|
| 前端路由守卫延迟 | 已优化 | index.html 预检查 token |
| 模型删除后刷新恢复 | 已修复 | loadFromFile 完全替换而非合并 |
| 密码重启后重置 | 已修复 | 持久化到 data/users.json |
| 代码丢失风险 | 已解决 | 每次开发立即提交，每天推送远程 |
| Anthropic 页面有示例但路由 404 | 已修复 | 后端新增 `/api/anthropic/v1/messages` 并返回 Anthropic 协议格式 |

### 改进建议

1. **代码安全**
   - 使用 `git stash` 暂存临时修改
   - 重要修改前创建备份分支：`git branch backup-xxx`
   - 当前阶段允许直接在 `main` 开发，但必须执行“小步提交 + 立即推送”

2. **开发流程**
   - 每完成一个小功能就提交，不要积攒
   - 提交信息要清晰，方便回溯
   - 使用 `git add -p` 选择性提交

3. **自动化**
   - 配置 git hooks，提交前自动检查
   - 配置定时任务，每天自动推送
   - 使用 CI/CD 自动部署

4. **文档更新**
   - 每次修改后更新 AGENTS.md
   - 记录版本号变更
   - 记录遇到的问题和解决方案

---

## 服务商端点

### Coding Plan 端点

| 服务商 | Coding Plan 端点 |
|--------|-----------------|
| 智谱AI | `https://open.bigmodel.cn/api/coding/paas/v4` |
| 火山方舟 | `https://ark.cn-beijing.volces.com/api/coding/v3` |
| 阿里云通义千问 | `https://coding.dashscope.aliyuncs.com/v1` |
| Kimi | `https://api.kimi.com/coding/v1` |
| MiniMax | `https://api.minimaxi.com/anthropic/v1` |

---

## 相关文档

- [CONTRIBUTING.md](CONTRIBUTING.md) - 贡献指南
- [SECURITY.md](SECURITY.md) - 安全策略
- [CHANGELOG.md](CHANGELOG.md) - 变更日志
- [ENTERPRISE_OPTIMIZATION.md](ENTERPRISE_OPTIMIZATION.md) - 优化分析

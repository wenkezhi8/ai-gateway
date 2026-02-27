# AI Gateway - 开发规范文档

> 本文档定义 AI Gateway 项目的开发规范、工作流程与输出要求。

## 目录

- [快速参考](#快速参考)
- [TDD 工作流程](#tdd-工作流程)
- [前端开发规范](#前端开发规范)
- [生产级代码迭代指令](#生产级代码迭代指令)
- [会话执行规范](#会话执行规范)
- [Git 工作流](#git-工作流)
- [代码风格规范](#代码风格规范)
- [测试规范](#测试规范)
- [API 规范](#api-规范)
- [安全规范](#安全规范)
- [部署规范](#部署规范)
- [历史教训](#历史教训)
- [开发规划](#开发规划)
---

## 快速参考

### 常用命令

```bash
# 后端
make lint              # 代码检查
make test              # 运行测试
make test-coverage     # 测试覆盖率
make build             # 构建
make ci-local          # 本地运行所有检查

# 前端
cd web && npm run lint       # 代码检查
cd web && npm run typecheck  # 类型检查
cd web && npm run build      # 构建前端

# 开发重启
./scripts/dev-restart.sh     # 统一重启（前端构建 + 后端重启）
```

### 端口配置

| 服务 | 端口 |
|------|------|
| 前端/后端 | 8566 |
| Metrics | 9090 |
| Redis | 6379 |

### 常量配置文件

| 文件 | 内容 |
|------|------|
| `internal/constants/routes.go` | API 路径 |
| `internal/constants/config.go` | 端口号 |
| `web/src/constants/api.ts` | 前端 API 路径 |

---

## TDD 工作流程

### 标准开发流程

| 步骤 | 流程 | 具体工作要求 |
|------|------|------|
| **1** | **问题排查** | 读取相关代码、运行现有测试、发现并列出所有问题 |
| **2** | **方案讨论** | 确认问题是否需要修复、输出可落地方案、确定任务节点 |
| **3** | **编写测试** | 先写测试用例，定义预期行为（红） |
| **4** | **代码实现** | 实现最小代码使测试通过（绿） |
| **5** | **代码重构** | 优化代码结构，保持测试通过（重构） |
| **6** | **回归验证** | 运行全部测试，确保无回归 |
| **7** | **提交归档** | 提交代码，更新文档 |

### TDD 检查清单

- [ ] 先写测试，后写实现
- [ ] 测试覆盖核心业务逻辑
- [ ] 每个测试只验证一个行为
- [ ] 使用表格驱动测试覆盖边界情况
- [ ] Mock 外部依赖（数据库、API 等）
- [ ] 测试命名清晰：`Test<函数>_<场景>_<预期结果>`

---

## 前端开发规范

### 架构原则：全 API 调用

> **核心原则**：前端所有数据、所有渲染，全部走 API 调用。
> 这是标准的 SPA（前后端分离）架构，99% 现代项目都这样做。

| 层级 | 职责 |
|------|------|
| **前端** | 页面、交互、渲染、请求 API |
| **后端** | 接口、数据、业务逻辑、数据库 |

### 前端开发流程（TDD）

| 步骤 | 流程 | 具体工作 |
|------|------|------|
| **1** | **定义 API** | 确认接口路径、参数、响应格式 |
| **2** | **写请求函数** | 在 `api/` 下封装请求，带类型定义 |
| **3** | **写组件骨架** | loading 状态 + 空状态 + 错误状态 |
| **4** | **对接数据** | onMounted 调用 API，渲染数据 |
| **5** | **验证** | 类型检查 + 构建 + 页面测试 |

### 必须避开的 4 个坑

#### 1. 分页、筛选、排序必须后端做

```typescript
// ✅ 正确：后端分页
const params = { page: 1, pageSize: 20, sort: 'createdAt' }
const res = await request.get('/api/admin/accounts', { params })

// ❌ 错误：前端拉全量再分页（巨卡、巨慢）
const all = await request.get('/api/admin/accounts')  // 拉了 10000 条！
const page = all.slice(0, 20)  // 前端分页
```

#### 2. 防抖、节流、缓存

```typescript
// ✅ 正确：防抖搜索
import { debounce } from 'lodash-es'
const handleSearch = debounce(async (keyword) => {
  const res = await request.get('/api/search', { params: { q: keyword } })
}, 300)

// ✅ 正确：列表缓存到 store
const cacheStore = useCacheStore()
if (!cacheStore.accounts.length) {
  cacheStore.accounts = await request.get('/api/admin/accounts')
}
```

#### 3. Loading + 错误 + 空状态必须做

```vue
<template>
  <!-- Loading -->
  <div v-if="loading" class="loading">加载中...</div>
  
  <!-- 错误 -->
  <div v-else-if="error" class="error">
    {{ error }}
    <el-button @click="fetchData">重试</el-button>
  </div>
  
  <!-- 空状态 -->
  <div v-else-if="list.length === 0" class="empty">暂无数据</div>
  
  <!-- 正常 -->
  <div v-else>
    <div v-for="item in list" :key="item.id">{{ item.name }}</div>
  </div>
</template>

<script setup>
const loading = ref(true)
const error = ref('')
const list = ref([])

async function fetchData() {
  loading.value = true
  error.value = ''
  try {
    list.value = await request.get('/api/list')
  } catch (e) {
    error.value = e.message || '请求失败'
  } finally {
    loading.value = false
  }
}
</script>
```

#### 4. 权限校验后端做

```typescript
// ✅ 正确：后端校验权限
// 前端只是隐藏按钮（UI 优化），不做权限判断
<el-button v-if="canDelete" @click="handleDelete">删除</el-button>

// 后端必须再次校验
func (h *Handler) Delete(c *gin.Context) {
    user := GetUser(c)
    if !user.HasPermission("delete") {
        c.JSON(403, gin.H{"error": "无权限"})
        return
    }
    // 执行删除...
}
```

### 前端变更必做流程

```bash
# 1) 类型检查
cd web && npm run typecheck

# 2) 构建前端
cd web && npm run build

# 3) 重启服务
cd .. && ./scripts/dev-restart.sh

# 4) 浏览器强制刷新 (Mac: Cmd+Shift+R)
```

---

## 生产级代码迭代指令

> 用于指导生产环境的代码迭代，确保代码质量、可维护性和性能。

### 基础信息模板

```text
【基础信息】
技术栈&运行环境：<例：Go 1.21 + Gin + Redis，运行在4核8G容器，QPS峰值1000>
代码业务作用：<例：AI 请求代理网关，当前平均响应时间 200ms>
已知现存问题：<例：偶发超时、缓存击穿、连接池耗尽>
待升级功能：<例：1. 新增 Anthropic 兼容；2. 语义缓存；3. 告警通知>
需求优先级：<兼容原有逻辑 > 功能落地 > 性能优化 ≥ 可维护性升级>
```

### 输出规范

#### 第一部分：完整优化后代码

- 所有修改点加 `// MODIFY:` 注释
- 禁止无说明的破坏性变更
- 保持代码风格一致

#### 第二部分：核心优化建议（2-5 条）

每条必须包含：

| 字段 | 说明 | 示例 |
|------|------|------|
| **分类** | 功能升级/性能优化/可维护性优化 | 性能优化 |
| **实现逻辑** | 具体实现步骤 | 添加连接池复用 |
| **量化收益** | 可衡量的收益 | 响应时间降低 40% |
| **风险提示** | 潜在风险（如有） | 需要重启服务 |

#### 第三部分：单元测试 + 性能自测

```go
// 核心改动的单元测试示例
func TestProxyHandler_StreamChat_Timeout(t *testing.T) {
    // 测试超时场景
    handler := setupTestHandler()
    req := &ChatRequest{Model: "test", Timeout: 100}
    
    start := time.Now()
    _, err := handler.StreamChat(ctx, req)
    elapsed := time.Since(start)
    
    assert.Error(t, err)
    assert.Less(t, elapsed.Milliseconds(), int64(150))
}
```

**性能自测方案**：

```bash
# 1. 压测命令
wrk -t4 -c100 -d30s http://localhost:8566/api/v1/chat/completions

# 2. 监控指标
# - QPS
# - P99 延迟
# - 内存占用
# - CPU 使用率

# 3. 对比基线
# 优化前：QPS 500，P99 800ms
# 优化后：QPS 800，P99 400ms
```

#### 第四部分：依赖变更清单

如有以下变更，必须单独列出：

- 依赖升级（版本号 + 变更原因）
- 配置变更（环境变量 + 默认值）
- 数据库变更（DDL + 迁移脚本）

```text
【依赖变更】
- go.mod: github.com/redis/go-redis v9.0.0 -> v9.2.0（修复连接泄漏）
- 新增环境变量：CLASSIFIER_TIMEOUT_MS=5000（默认 5000）
```

---

### 角色设定

你是拥有10年以上经验的资深全栈开发工程师，精通各类编程语言、前后端框架、系统架构。中文回复所有问题。

### 执行原则

1. **先计划后执行**：先输出 Plan（改动点、影响面、验证命令），收到确认后再落地
2. **范围控制**：只改与当前需求相关的文件，禁止顺手改动无关逻辑
3. **验证优先**：执行类型检查/单测/e2e 中的必要项
4. **连续执行**：默认连续执行到最终结果，仅在以下情况中断：
   - 缺少关键输入（凭据、账户ID等）
   - 高风险不可逆操作（push/tag、生产安全策略）

### 结果输出顺序

1. 问题根因
2. 修复方案
3. 改动清单（文件路径）
4. 测试结果
5. 风险与回滚点
6. 版本建议（X.Y.Z）

### 会话前流程卡

```text
【本次流程卡】
目标：<一句话目标>
改动范围：<允许修改的文件或目录>
Git权限：<允许 commit/push/tag>
必跑验证：<命令1>、<命令2>
```

### 会话结束回报

每次阶段结束必须回报：

1. 是否已本地提交（是/否）
2. 最新 commit hash（若有）
3. 当前版本号（`git tag`，无 tag 写 `no-tag`）
4. 建议版本号（PATCH/MINOR/MAJOR）
5. 是否已 push（是/否）
6. 工作区是否 clean

```bash
# 查看当前版本
git describe --tags --abbrev=0 2>/dev/null || echo no-tag
```

---

## Git 工作流

### 分支策略

> 当前项目采用 **main 分支直接开发**，不强制 feature 分支。

### Git 权限判定表

| 操作 | 默认自动执行 | 触发条件 |
|------|-------------|----------|
| 本地改代码 | 是 | 收到确认后 |
| `git commit` | 是 | 每个里程碑结束 |
| `git push` | 否 | 用户明确指令 |
| `git tag` | 否 | 用户明确指令 |

### 提交信息规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <description>
```

| Type | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat: 添加 MiniMax 服务商支持` |
| `fix` | Bug 修复 | `fix: 修复 JWT token 过期计算错误` |
| `docs` | 文档更新 | `docs: 更新部署文档` |
| `refactor` | 重构 | `refactor: 优化缓存层实现` |
| `test` | 测试 | `test: 添加限流器单元测试` |
| `chore` | 构建/工具 | `chore: 更新 CI 配置` |

### 版本发布 SOP

**发布前提**（必须满足其一）：
1. 用户明确下达"发布版本"指令
2. 项目里程碑全部完成

**发布流程**：

```bash
# 1. 确认当前版本
git describe --tags --abbrev=0

# 2. 计算新版本（修复→PATCH，功能→MINOR，不兼容→MAJOR）

# 3. 更新文档（CHANGELOG.md）

# 4. 执行发布
git add -A
git commit -m "chore(release): vX.Y.Z"
git tag -a vX.Y.Z -m "release: vX.Y.Z"
git push origin main
git push origin vX.Y.Z

# 5. 核验
git describe --tags --abbrev=0
```

---

## 代码风格规范

### Go 代码规范

#### 命名规范

```go
// ✅ 正确
type AccountManager struct { ... }
func (m *AccountManager) GetAccount(id string) (*Account, error) { ... }
const MaxRetryCount = 3

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

// ❌ 错误：忽略错误
doSomething()
```

#### Context 传递

```go
// ✅ 正确：context 作为第一个参数
func (s *Service) Process(ctx context.Context, id string) error { ... }
```

### 前端代码规范

#### Vue 组件

```vue
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

interface Props {
  userId: string
}

const props = defineProps<Props>()
const loading = ref(false)

onMounted(async () => {
  await fetchData()
})
</script>
```

#### API 调用

```typescript
// ✅ 正确：使用常量和类型
import { API } from '@/constants/api'

async function sendChat(request: ChatRequest): Promise<ChatResponse> {
  const response = await fetch(API.V1.CHAT_COMPLETIONS, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request)
  })
  return response.json()
}

// ❌ 错误：硬编码路径
fetch('/api/v1/chat/completions', { ... })
```

---

## 测试规范

### 后端测试

#### 测试命名

```go
// ✅ 正确
func TestAccountManager_GetAccount_Success(t *testing.T) { ... }
func TestAccountManager_GetAccount_NotFound(t *testing.T) { ... }

// ❌ 错误
func TestGetAccount(t *testing.T) { ... }
```

#### 表格驱动测试

```go
func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        wantErr bool
    }{
        {name: "valid config", config: &Config{Port: "8080"}, wantErr: false},
        {name: "empty port", config: &Config{Port: ""}, wantErr: true},
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

### 测试覆盖率要求

| 模块 | 最低覆盖率 |
|------|-----------|
| 核心业务逻辑 | 80% |
| API Handler | 70% |
| 工具函数 | 90% |

---

## API 规范

### RESTful API 设计

| 操作 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 列表 | GET | `/api/v1/resources` | 获取资源列表 |
| 详情 | GET | `/api/v1/resources/:id` | 获取单个资源 |
| 创建 | POST | `/api/v1/resources` | 创建资源 |
| 更新 | PUT | `/api/v1/resources/:id` | 全量更新 |
| 删除 | DELETE | `/api/v1/resources/:id` | 删除资源 |

### 统一响应格式

```json
// 成功
{"code": 0, "message": "success", "data": { ... }}

// 错误
{"code": 1001, "message": "Invalid parameter", "error": "Detailed error"}
```

### 错误码范围

| 范围 | 类型 |
|------|------|
| 0 | 成功 |
| 1000-1999 | 参数错误 |
| 2000-2999 | 认证/授权错误 |
| 3000-3999 | 业务逻辑错误 |
| 5000-5999 | 服务端错误 |

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

### 安全检查清单

- [ ] 无硬编码密钥
- [ ] API Key 在日志中脱敏
- [ ] 输入验证完整
- [ ] Rate Limiting 启用

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
# 启动
./bin/ai-gateway

# 健康检查
curl http://localhost:8566/health

# 重启
lsof -ti:8566 | xargs kill -HUP
```

---

## 历史教训

### 2026-02-23 页面"请求的资源不存在"

**原因**：前端代码修改后没有构建/重启/清除缓存

**解决**：使用统一脚本 `./scripts/dev-restart.sh`，浏览器强制刷新

### 2026-02-23 代码丢失事件

**原因**：开发完成后未提交 git

**解决**：每完成功能立即 commit，禁止执行 `git checkout .`

### 2026-02-27 提交节奏偏差

**原因**：Git 执行口径不明确

**解决**：每次会话开头声明 Git 权限，每阶段自动本地 commit

---

## 开发规划

### 当前版本

**v1.6.5** (2026-02-27)

### 已完成功能

| 功能 | 状态 |
|------|------|
| 按任务类型管理缓存 | ✅ |
| 运维监控页面 | ✅ |
| 登录验证 + 密码修改 | ✅ |
| 智能路由 + 语义缓存 | ✅ |
| Anthropic 兼容入口 | ✅ |
| TTFT (首Token耗时) 追踪 | ✅ |
| 公开首页 | ✅ |

### 测试覆盖率

| 模块 | 覆盖率 |
|------|--------|
| internal/models | 100% |
| internal/routing | 60% |
| internal/cache | 48% |
| **整体** | **41%** |

### 待开发

| 任务 | 优先级 |
|------|--------|
| 单元测试补充 | 中 |
| SQLite 持久化 | 低 |

---

## 持久化文件

| 文件 | 内容 |
|------|------|
| `data/accounts.json` | 账号配置 |
| `data/model_scores.json` | 模型评分 |
| `data/router_config.json` | 路由配置 |
| `data/users.json` | 用户数据 |
| `data/feedback.json` | 反馈数据 |

---

## 服务商端点

### Coding Plan 端点

| 服务商 | 端点 |
|--------|------|
| 智谱AI | `https://open.bigmodel.cn/api/coding/paas/v4` |
| 火山方舟 | `https://ark.cn-beijing.volces.com/api/coding/v3` |
| 阿里云通义 | `https://coding.dashscope.aliyuncs.com/v1` |
| Kimi | `https://api.kimi.com/coding/v1` |
| MiniMax | `https://api.minimaxi.com/anthropic/v1` |

---

## 相关文档

- [CONTRIBUTING.md](CONTRIBUTING.md) - 贡献指南
- [SECURITY.md](SECURITY.md) - 安全策略
- [CHANGELOG.md](CHANGELOG.md) - 变更日志
- [docs/control-layer-operations.md](docs/control-layer-operations.md) - 控制层灰度手册

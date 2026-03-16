# Ollama 预热自动化 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 Ollama 管理页增加手动/启动时自动预热能力，支持按“意图模型 + Embedding 模型”选择目标并展示预热结果。

**Architecture:** 在 `OllamaService` 新增预热执行能力（按模型类型调用 chat/embed 端点并设置 `keep_alive=-1`），由 `RouterHandler` 负责解析当前配置中的预热目标模型并编排 API 输出。前端复用“服务管理”页扩展配置区和结果展示，运行时配置继续持久化到 `data/ollama_runtime_config.json`。

**Tech Stack:** Go (Gin), TypeScript + Vue3 + Element Plus, Vitest。

---

### Task 1: 后端运行时配置扩展（TDD）

**Files:**
- Modify: `internal/service/ollama_service.go`
- Modify: `internal/handler/admin/router.go`
- Modify: `internal/handler/admin/router_ollama_runtime_config_test.go`

**Step 1: Write the failing test**

- 在 `router_ollama_runtime_config_test.go` 新增断言：`GET /runtime-config` 返回预热配置字段；`PUT /runtime-config` 能保存 `auto_preload_on_startup`、`preload_targets`、`preload_timeout_seconds`。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run RuntimeConfig -v`
Expected: FAIL（缺少新字段或校验逻辑）。

**Step 3: Write minimal implementation**

- 在 `OllamaServiceConfig` 增加 `PreloadConfig`。
- 补齐默认值与 normalize。
- 更新 `UpdateOllamaRuntimeConfig` 请求绑定与校验。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run RuntimeConfig -v`
Expected: PASS。

### Task 2: 后端预热 API 与服务执行（TDD）

**Files:**
- Modify: `internal/service/ollama_service.go`
- Modify: `internal/service/ollama_service_test.go`
- Modify: `internal/handler/admin/router.go`
- Modify: `internal/handler/admin/admin.go`
- Create: `internal/handler/admin/router_ollama_preload_test.go`

**Step 1: Write the failing test**

- `ollama_service_test.go` 新增：
  - chat 模型预热成功。
  - embedding 模型预热成功（embed/embeddings 模式）。
  - 单模型超时按配置触发失败。
- `router_ollama_preload_test.go` 新增：`POST /ollama/preload` 返回结果列表（intent/embedding），并校验去重。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service ./internal/handler/admin -run Preload -v`
Expected: FAIL（缺少预热方法和路由）。

**Step 3: Write minimal implementation**

- 在 `OllamaService` 添加预热方法：
  - 模型类型 `intent` 走 `/api/chat`，`keep_alive=-1`。
  - 模型类型 `embedding` 按 endpoint mode 走 `/api/embed` 或 `/api/embeddings`，并带 `keep_alive=-1`。
  - 每个模型单独超时，默认 180 秒。
- 在 `RouterHandler` 增加 `POST /api/admin/router/ollama/preload`：
  - 从 classifier/vector 配置解析目标模型。
  - 支持 `preload_targets` 选择。
  - 返回每个模型预热结果。
- 在 `admin.go` 注册路由。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service ./internal/handler/admin -run Preload -v`
Expected: PASS。

### Task 3: 启动时自动预热调度（TDD）

**Files:**
- Modify: `internal/handler/admin/router.go`
- Modify: `internal/handler/admin/router_ollama_preload_test.go`

**Step 1: Write the failing test**

- 新增测试覆盖：当 `auto_preload_on_startup=true` 时，触发一次自动预热（并可通过 `sync.Once` 防重复）。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run AutoPreload -v`
Expected: FAIL。

**Step 3: Write minimal implementation**

- 在 `RouterHandler` 增加一次性自动预热启动逻辑。
- 复用预热编排函数，失败仅记录日志不阻断服务。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run AutoPreload -v`
Expected: PASS。

### Task 4: 前端 API 与状态流（TDD）

**Files:**
- Modify: `web/src/api/routing-domain.ts`
- Modify: `web/src/api/routing-domain-ollama-runtime.test.ts`
- Modify: `web/src/views/ollama/composables/useOllamaConsoleCore.ts`

**Step 1: Write the failing test**

- 扩展 `routing-domain-ollama-runtime.test.ts`：
  - runtime config 字段包含 preload 配置。
  - 新增 `preloadOllamaModels` API 调用测试。

**Step 2: Run test to verify it fails**

Run: `npm --prefix web run test:unit -- routing-domain-ollama-runtime.test.ts`
Expected: FAIL。

**Step 3: Write minimal implementation**

- API 层增加 preload 类型和方法。
- composable 增加：
  - 预热配置字段读写。
  - 手动预热动作、加载态、最近结果。

**Step 4: Run test to verify it passes**

Run: `npm --prefix web run test:unit -- routing-domain-ollama-runtime.test.ts`
Expected: PASS。

### Task 5: 前端 UI 与回归（TDD）

**Files:**
- Modify: `web/src/views/ollama/components/OllamaServiceTab.vue`
- Modify: `web/src/views/ollama/ollama-runtime-config.test.ts`

**Step 1: Write the failing test**

- 为 UI 文案与关键交互增加断言：
  - 启动时自动预热开关。
  - 预热目标选择。
  - 手动预热按钮。
  - 预热结果展示。

**Step 2: Run test to verify it fails**

Run: `npm --prefix web run test:unit -- ollama-runtime-config.test.ts`
Expected: FAIL。

**Step 3: Write minimal implementation**

- 在“服务管理”页添加预热配置区域和结果列表。
- 保持当前布局，不新增标签页。

**Step 4: Run test to verify it passes**

Run: `npm --prefix web run test:unit -- ollama-runtime-config.test.ts`
Expected: PASS。

### Task 6: 全量验证

**Files:**
- Modify: `docs/LOCKS.md`

**Step 1: Run backend validations**

Run: `make lint && make test && make build && go build ./cmd/gateway`
Expected: 全部通过。

**Step 2: Run frontend validations**

Run: `npm --prefix web run typecheck && npm --prefix web run build && npm --prefix web run test:unit`
Expected: 全部通过。

**Step 3: Run integration restart script**

Run: `./scripts/dev-restart.sh`
Expected: 服务健康检查通过。

**Step 4: Commit**

```bash
git add docs/LOCKS.md \
  docs/plans/2026-03-03-ollama-preload-automation.md \
  internal/handler/admin/admin.go \
  internal/handler/admin/router.go \
  internal/handler/admin/router_ollama_runtime_config_test.go \
  internal/handler/admin/router_ollama_preload_test.go \
  internal/service/ollama_service.go \
  internal/service/ollama_service_test.go \
  web/src/api/routing-domain.ts \
  web/src/api/routing-domain-ollama-runtime.test.ts \
  web/src/views/ollama/composables/useOllamaConsoleCore.ts \
  web/src/views/ollama/components/OllamaServiceTab.vue \
  web/src/views/ollama/ollama-runtime-config.test.ts
git commit -m "feat(ollama): add manual and startup model preload flow"
```

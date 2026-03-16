# Vector DB Projectbook B Alignment Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在现有 Vector DB 能力基础上，补齐与项目书 B 口径仍不一致的可工程化缺口，做到“代码能力 + 页面结构 + 文档说明”三位一体对齐。

**Architecture:** 保持后端 `handler -> service -> repository/backend` 分层不变，在导入流程补充 cancel 语义和 API；前端按项目书形态新增导入子组件与监控 alerts 子页，复用既有 API/状态管理；最后更新文档与验证证据，确保门禁通过。

**Tech Stack:** Go + Gin + SQLite + Qdrant + Vue3 + TypeScript + Vitest。

---

### Task 1: 导入任务取消能力（Backend API + Service + Tests）

**Files:**
- Modify: `internal/vector-db/import_job.go`
- Modify: `internal/vector-db/collection_handler.go`
- Modify: `internal/handler/admin/admin.go`
- Test: `internal/vector-db/collection_handler_test.go`
- Test: `internal/vector-db/collection_service_test.go`

**Step 1: 写失败测试（取消导入任务成功/非法状态）**

在 handler/service 测试中新增 `POST /api/admin/vector-db/import-jobs/:id/cancel` 场景，覆盖 running/pending 可取消，completed 不可取消。

**Step 2: 运行后端测试验证 RED**

Run: `go test ./internal/vector-db -run Import -v`
Expected: FAIL（cancel 能力尚未实现）。

**Step 3: 实现最小后端能力**

新增 `CancelImportJob(ctx, id)`：
- 允许状态：`pending`/`running`/`retrying`
- 目标状态：`cancelled`
- 非允许状态返回 400 语义错误。

新增 handler：`CancelImportJob`，并在 admin 路由注册 `POST /vector-db/import-jobs/:id/cancel`。

**Step 4: 运行后端测试验证 GREEN**

Run: `go test ./internal/vector-db -run Import -v`
Expected: PASS。

**Step 5: Commit**

`git commit -m "feat(vector-db): add import job cancel workflow"`

### Task 2: 前端导入中心对齐项目书子组件结构（Json/Csv/Pdf Importer）

**Files:**
- Create: `web/src/views/vector-db/import/JsonImporter.vue`
- Create: `web/src/views/vector-db/import/CsvImporter.vue`
- Create: `web/src/views/vector-db/import/PdfImporter.vue`
- Modify: `web/src/views/vector-db/import/index.vue`
- Modify: `web/src/api/vector-db-domain.ts`
- Test: `web/src/views/vector-db/import/index.test.ts`

**Step 1: 写失败前端测试**

覆盖：
- import 页存在三类导入子组件入口（tab 或分区）。
- 点击“取消任务”触发 cancel API。

**Step 2: 运行前端测试验证 RED**

Run: `cd web && npm run test:unit -- src/views/vector-db/import/index.test.ts`
Expected: FAIL。

**Step 3: 实现最小 UI 对齐**

新增 `JsonImporter/CsvImporter/PdfImporter` 组件（最小可用表单 + 复用 `createImportJob`）；
在 `import/index.vue` 加入子组件容器，并在任务列表增加“取消任务”按钮。

**Step 4: 实现 API 封装**

在 `web/src/api/vector-db-domain.ts` 新增：
- `cancelImportJob(id: string)` -> `POST /admin/vector-db/import-jobs/:id/cancel`

**Step 5: 运行前端测试验证 GREEN**

Run: `cd web && npm run test:unit -- src/views/vector-db/import/index.test.ts`
Expected: PASS。

**Step 6: Commit**

`git commit -m "feat(vector-db): align import ui with json csv pdf components"`

### Task 3: 监控告警页面结构对齐（新增 alerts 子页）

**Files:**
- Create: `web/src/views/vector-db/monitoring/alerts.vue`
- Modify: `web/src/views/vector-db/monitoring/index.vue`
- Modify: `web/src/router/index.ts`
- Test: `web/src/views/vector-db/monitoring/index.test.ts`
- Create: `web/src/views/vector-db/monitoring/alerts.test.ts`

**Step 1: 写失败测试**

覆盖：
- 路由 `/vector-db/monitoring/alerts` 可访问。
- alerts 子页可加载告警规则并支持创建/删除基本流程。

**Step 2: 运行前端测试验证 RED**

Run: `cd web && npm run test:unit -- src/views/vector-db/monitoring/index.test.ts src/views/vector-db/monitoring/alerts.test.ts`
Expected: FAIL。

**Step 3: 实现最小页面拆分**

从 `monitoring/index.vue` 抽出告警规则管理区到 `monitoring/alerts.vue`；
`monitoring/index.vue` 保留指标概览。

**Step 4: 更新路由**

新增路由：`/vector-db/monitoring/alerts`。

**Step 5: 运行前端测试验证 GREEN**

Run: `cd web && npm run test:unit -- src/views/vector-db/monitoring/index.test.ts src/views/vector-db/monitoring/alerts.test.ts`
Expected: PASS。

**Step 6: Commit**

`git commit -m "feat(vector-db): split monitoring alerts into dedicated page"`

### Task 4: 文档与项目书对齐收口

**Files:**
- Modify: `docs/API_REFERENCE.md`
- Modify: `docs/USER_GUIDE.md`
- Modify: `docs/plans/2026-03-02-vector-db-projectbook-gap-closure-delivery-matrix.md`
- Modify: `/Users/openclaw/Desktop/AI-Gateway-Vector-DB-项目书.md`（仅补充实现映射与版本变更）

**Step 1: 更新 API 文档**

补充 `cancel import job` 接口与 monitoring/alerts 页面路由说明。

**Step 2: 更新用户手册**

补充 Json/Csv/Pdf 子组件入口说明、取消任务操作说明。

**Step 3: 更新交付矩阵证据**

将新增能力加入矩阵并记录验证命令。

**Step 4: Commit**

`git commit -m "docs(vector-db): align projectbook b-scope docs and matrix"`

### Task 5: 全量门禁与最终验收

**Files:**
- N/A（验证任务）

**Step 1: 后端门禁**

Run: `PATH=/Users/openclaw/go/bin:$PATH make lint && go test ./... && make build && go build ./cmd/gateway`
Expected: PASS。

**Step 2: 前端门禁**

Run: `cd web && npm run typecheck && npm run build && npm run test:unit`
Expected: PASS。

**Step 3: 最终状态检查**

Run: `git status --short`
Expected: 空（或仅允许 `.codex/`）。

**Step 4: Commit（如有残余未提交）**

`git commit -m "chore(vector-db): finalize projectbook b-scope closure"`

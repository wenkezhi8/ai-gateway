# Vector DB Projectbook Gap Closure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 补齐当前仓库相对项目书未闭环能力，形成可验收的工程闭环交付。

**Architecture:** 基于现有 `internal/vector-db` 分层（handler/service/repository/backend）做增量扩展；前端延续 `web/src/views/vector-db` 模块化页面；所有能力以 TDD 方式落地，并在每个任务后跑局部验证，最终跑全量门禁。

**Tech Stack:** Go + Gin + SQLite + Redis + Vue3 + TypeScript + Vitest。

---

### Task 1: Collection Empty API + UI

**Files:**
- Modify: `internal/vector-db/collection.go`
- Modify: `internal/vector-db/collection_handler.go`
- Modify: `internal/vector-db/search_handler.go` (如需共享路由注册逻辑)
- Modify: `internal/handler/admin/admin.go`
- Modify: `web/src/views/vector-db/collections/index.vue`
- Test: `internal/vector-db/collection_handler_test.go`
- Test: `internal/vector-db/collection_service_test.go`

**Step 1: Write failing backend tests**

为 empty 行为增加失败测试：成功清空、集合不存在、重复清空幂等。

**Step 2: Run backend tests to verify RED**

Run: `go test ./internal/vector-db -run Collection -v`
Expected: FAIL，提示 empty 行为未实现。

**Step 3: Implement minimal backend empty flow**

在 service 增加 `EmptyCollection`，在 handler 增加 `POST /collections/:name/empty`。

**Step 4: Re-run backend tests to verify GREEN**

Run: `go test ./internal/vector-db -run Collection -v`
Expected: PASS。

**Step 5: Write failing frontend test / behavior assertion**

为集合页 empty 操作增加交互测试（确认弹窗与成功刷新）。

**Step 6: Run frontend test to verify RED**

Run: `cd web && npm run test:unit -- src/views/vector-db/collections/index.test.ts`
Expected: FAIL。

**Step 7: Implement minimal UI action**

在集合列表加入 empty 按钮与 API 调用。

**Step 8: Re-run frontend test to verify GREEN**

Run: `cd web && npm run test:unit -- src/views/vector-db/collections/index.test.ts`
Expected: PASS。

**Step 9: Commit**

`git commit -m "feat(vector-db): add collection empty workflow"`

### Task 2: Text Search End-to-End

**Files:**
- Modify: `internal/vector-db/search_model.go`
- Modify: `internal/vector-db/search_service.go`
- Modify: `internal/vector-db/search_handler.go`
- Test: `internal/vector-db/search_service_test.go`
- Test: `internal/vector-db/search_handler_test.go`
- Modify: `web/src/views/vector-db/search/index.vue`
- Test: `web/src/views/vector-db/search/index.test.ts`

**Step 1: Write failing backend tests**

覆盖 text 路径成功、embedding 失败、text+vector 优先级。

**Step 2: Run backend tests to verify RED**

Run: `go test ./internal/vector-db -run Search -v`
Expected: FAIL（当前 text 不支持）。

**Step 3: Implement minimal text search path**

在 service 注入 embedding 逻辑，text 生成向量后复用现有检索流程。

**Step 4: Re-run backend tests to verify GREEN**

Run: `go test ./internal/vector-db -run Search -v`
Expected: PASS。

**Step 5: Write failing frontend test**

覆盖 text 输入搜索与错误提示。

**Step 6: Run frontend test to verify RED**

Run: `cd web && npm run test:unit -- src/views/vector-db/search/index.test.ts`
Expected: FAIL。

**Step 7: Implement minimal UI updates**

完善 text 搜索参数组装与状态反馈。

**Step 8: Re-run frontend test to verify GREEN**

Run: `cd web && npm run test:unit -- src/views/vector-db/search/index.test.ts`
Expected: PASS。

**Step 9: Commit**

`git commit -m "feat(vector-db): support text semantic search"`

### Task 3: Import Closure (JSON/CSV/PDF + Errors + Retry)

**Files:**
- Modify: `internal/vector-db/import_job.go`
- Modify: `internal/vector-db/import_job_model.go`
- Modify: `internal/vector-db/collection_handler.go`
- Test: `internal/vector-db/import_job_test.go`
- Modify: `web/src/views/vector-db/collections/index.vue`
- Create: `web/src/views/vector-db/import/index.vue`
- Create: `web/src/views/vector-db/import/index.test.ts`

**Step 1: Write failing backend tests**

增加 PDF 路径、失败日志分页、重试行为测试。

**Step 2: Run backend tests to verify RED**

Run: `go test ./internal/vector-db -run Import -v`
Expected: FAIL。

**Step 3: Implement minimal import closure updates**

补齐 PDF 处理分支、错误查询与重试语义。

**Step 4: Re-run backend tests to verify GREEN**

Run: `go test ./internal/vector-db -run Import -v`
Expected: PASS。

**Step 5: Write failing frontend tests**

为 import 页面任务流编写测试。

**Step 6: Run frontend tests to verify RED**

Run: `cd web && npm run test:unit -- src/views/vector-db/import/index.test.ts`
Expected: FAIL。

**Step 7: Implement minimal import UI page**

实现任务列表、错误详情入口与重试操作。

**Step 8: Re-run frontend tests to verify GREEN**

Run: `cd web && npm run test:unit -- src/views/vector-db/import/index.test.ts`
Expected: PASS。

**Step 9: Commit**

`git commit -m "feat(vector-db): close import workflow gaps"`

### Task 4: RBAC Permission Expansion + Audit Query API

**Files:**
- Modify: `internal/vector-db/rbac_model.go`
- Modify: `internal/vector-db/rbac_service.go`
- Modify: `internal/vector-db/rbac_middleware.go`
- Modify: `internal/vector-db/collection_handler.go`
- Modify: `internal/vector-db/repository_sqlite.go`
- Test: `internal/vector-db/rbac_service_test.go`
- Test: `internal/vector-db/rbac_middleware_test.go`
- Create: `internal/vector-db/audit_handler_test.go`

**Step 1: Write failing tests for expanded permissions and audit queries**

**Step 2: Run backend tests to verify RED**

Run: `go test ./internal/vector-db -run "RBAC|Audit" -v`
Expected: FAIL。

**Step 3: Implement minimal RBAC/audit query support**

**Step 4: Re-run backend tests to verify GREEN**

Run: `go test ./internal/vector-db -run "RBAC|Audit" -v`
Expected: PASS。

**Step 5: Commit**

`git commit -m "feat(vector-db): expand rbac and expose audit queries"`

### Task 5: Redis-backed Rate Limiter

**Files:**
- Modify: `internal/vector-db/rate_limit.go`
- Modify: `internal/router/router.go`
- Test: `internal/vector-db/rate_limit_test.go`

**Step 1: Write failing tests for redis token bucket behavior**

**Step 2: Run tests to verify RED**

Run: `go test ./internal/vector-db -run RateLimit -v`
Expected: FAIL。

**Step 3: Implement minimal redis-backed limiter with local fallback**

**Step 4: Re-run tests to verify GREEN**

Run: `go test ./internal/vector-db -run RateLimit -v`
Expected: PASS。

**Step 5: Commit**

`git commit -m "feat(vector-db): migrate search rate limit to redis token bucket"`

### Task 6: Backup Policy (Auto + Retention + Cleanup)

**Files:**
- Modify: `internal/vector-db/backup_model.go`
- Modify: `internal/vector-db/backup_service.go`
- Modify: `internal/vector-db/backup_handler.go`
- Test: `internal/vector-db/backup_service_test.go`
- Test: `internal/vector-db/backup_handler_test.go`

**Step 1: Write failing tests for auto plan/retention/cleanup**

**Step 2: Run tests to verify RED**

Run: `go test ./internal/vector-db -run Backup -v`
Expected: FAIL。

**Step 3: Implement minimal backup policy support**

**Step 4: Re-run tests to verify GREEN**

Run: `go test ./internal/vector-db -run Backup -v`
Expected: PASS。

**Step 5: Commit**

`git commit -m "feat(vector-db): add backup policy and retention cleanup"`

### Task 7: Admin UI for Audit/Backup/Import Navigation Closure

**Files:**
- Modify: `web/src/router/index.ts`
- Modify: `web/src/components/Layout/index.vue`
- Create: `web/src/views/vector-db/audit/index.vue`
- Create: `web/src/views/vector-db/audit/index.test.ts`
- Modify: `web/src/views/vector-db/backup/index.vue`
- Modify: `web/src/views/vector-db/backup/index.test.ts`

**Step 1: Write failing frontend tests**

**Step 2: Run tests to verify RED**

Run: `cd web && npm run test:unit -- src/views/vector-db/audit/index.test.ts src/views/vector-db/backup/index.test.ts`
Expected: FAIL。

**Step 3: Implement minimal UI pages and navigation**

**Step 4: Re-run tests to verify GREEN**

Run: `cd web && npm run test:unit -- src/views/vector-db/audit/index.test.ts src/views/vector-db/backup/index.test.ts`
Expected: PASS。

**Step 5: Commit**

`git commit -m "feat(vector-db): complete admin ui for audit and backup policy"`

### Task 8: Full Verification + Documentation + Delivery Matrix

**Files:**
- Modify: `docs/API_REFERENCE.md`
- Modify: `docs/USER_GUIDE.md`
- Modify: `docs/plans/2026-03-02-vector-db-full-scope-delivery-matrix.md`

**Step 1: Run backend gates**

Run: `PATH=/Users/openclaw/go/bin:$PATH make lint && make test && make build && go build ./cmd/gateway`
Expected: 全部 PASS。

**Step 2: Run frontend gates**

Run: `cd web && npm run typecheck && npm run build && npm run test:unit`
Expected: 全部 PASS。

**Step 3: Update docs with exact capabilities and limits**

确保文档不夸大，与实际实现一致。

**Step 4: Update delivery matrix with fresh evidence commands**

**Step 5: Commit**

`git commit -m "docs(vector-db): finalize gap-closure docs and verification matrix"`

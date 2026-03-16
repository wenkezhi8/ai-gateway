# Vector DB Full Scope Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 完成项目书中向量数据库 P1-P4 未完成能力（检索 API、监控告警、RBAC、限流、备份恢复、可视化）并通过全量门禁。

**Architecture:** 在现有 `internal/vector-db` 与 `web/src/views/vector-db` 基础上增量扩展，继续使用 `SQLite 元数据 + Qdrant 向量存储`。按里程碑分四阶段交付，每阶段都要求后端/前端测试与构建通过，最终以 `make lint + 全量测试构建` 作为完成判定。

**Tech Stack:** Go + Gin + SQLite + Qdrant + Vue3 + TypeScript + Vitest。

---

### Task 1: 建立 P1 检索领域模型与仓储接口

**Files:**
- Modify: `internal/vector-db/collection_model.go`
- Modify: `internal/vector-db/collection.go`
- Create: `internal/vector-db/search_model.go`
- Create: `internal/vector-db/search_service.go`
- Test: `internal/vector-db/search_service_test.go`

**Step 1: 写失败测试（检索请求校验与结果映射）**

在 `internal/vector-db/search_service_test.go` 新增：
- `TestVectorDBService_SearchVectors_WhenInvalidRequest_ShouldFail`
- `TestVectorDBService_SearchVectors_WhenBackendSuccess_ShouldReturnResults`

**Step 2: 运行测试确认失败**

Run: `go test ./internal/vector-db -run SearchVectors -v`
Expected: FAIL（函数或类型不存在）

**Step 3: 最小实现 search 模型与 service**

在 `search_model.go` 增加请求/响应结构；在 `search_service.go` 增加 `SearchVectors/RecommendVectors/GetVectorByID` 框架实现。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/vector-db -run SearchVectors -v`
Expected: PASS

**Step 5: 提交**

Run:
`git add internal/vector-db/search_model.go internal/vector-db/search_service.go internal/vector-db/search_service_test.go internal/vector-db/collection_model.go internal/vector-db/collection.go && git commit -m "feat(vector-db): add search service domain models"`

### Task 2: 接入 P1 检索 HTTP API

**Files:**
- Modify: `internal/vector-db/collection_handler.go`
- Create: `internal/vector-db/search_handler.go`
- Modify: `internal/router/router.go`
- Test: `internal/vector-db/search_handler_test.go`

**Step 1: 写失败测试（/api/v1/vector 路由）**

在 `internal/vector-db/search_handler_test.go` 添加：
- `TestSearchHandler_SearchRoute_ShouldReturn200`
- `TestSearchHandler_RecommendRoute_ShouldReturn200`
- `TestSearchHandler_GetVectorByID_ShouldReturn200`

**Step 2: 运行测试确认失败**

Run: `go test ./internal/vector-db -run SearchHandler -v`
Expected: FAIL（路由未注册）

**Step 3: 最小实现 handler 与路由注册**

新增 `search_handler.go`，并在 `internal/router/router.go` 注入 `/api/v1/vector/collections/:name/*`。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/vector-db -run SearchHandler -v`
Expected: PASS

**Step 5: 提交**

Run:
`git add internal/vector-db/search_handler.go internal/vector-db/search_handler_test.go internal/vector-db/collection_handler.go internal/router/router.go && git commit -m "feat(vector-db): expose vector search APIs"`

### Task 3: 增加 P1 监控聚合与告警规则后端

**Files:**
- Create: `internal/vector-db/monitoring_model.go`
- Create: `internal/vector-db/monitoring_service.go`
- Create: `internal/vector-db/monitoring_handler.go`
- Modify: `internal/vector-db/repository_sqlite.go`
- Modify: `migrations/002_vector_db.sql`
- Test: `internal/vector-db/monitoring_service_test.go`
- Test: `internal/vector-db/monitoring_handler_test.go`

**Step 1: 写失败测试（告警规则 CRUD 与指标接口）**

新增测试：
- `TestMonitoringService_AlertRulesCRUD`
- `TestMonitoringHandler_GetMetrics_ShouldReturnSummary`

**Step 2: 运行测试确认失败**

Run: `go test ./internal/vector-db -run Monitoring -v`
Expected: FAIL

**Step 3: 最小实现 monitoring 模块**

实现 metrics summary、告警规则 CRUD、SQLite 持久化。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/vector-db -run Monitoring -v`
Expected: PASS

**Step 5: 提交**

Run:
`git add internal/vector-db/monitoring_model.go internal/vector-db/monitoring_service.go internal/vector-db/monitoring_handler.go internal/vector-db/monitoring_service_test.go internal/vector-db/monitoring_handler_test.go internal/vector-db/repository_sqlite.go migrations/002_vector_db.sql && git commit -m "feat(vector-db): add monitoring metrics and alert rules"`

### Task 4: 前端接入 P1 搜索与监控页面

**Files:**
- Create: `web/src/views/vector-db/search/index.vue`
- Create: `web/src/views/vector-db/monitoring/index.vue`
- Modify: `web/src/api/vector-db-domain.ts`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/components/Layout/index.vue`
- Test: `web/src/views/vector-db/search/index.test.ts`
- Test: `web/src/views/vector-db/monitoring/index.test.ts`

**Step 1: 写失败测试（页面加载与 API 调用）**

在新测试中覆盖：页面加载、请求触发、空态/错态/成功态。

**Step 2: 运行测试确认失败**

Run: `cd web && npm run test:unit -- src/views/vector-db/search/index.test.ts src/views/vector-db/monitoring/index.test.ts`
Expected: FAIL

**Step 3: 最小实现页面与 API facade**

新增搜索页与监控页，接入 `vector-db-domain.ts`。

**Step 4: 运行测试确认通过**

Run: `cd web && npm run test:unit -- src/views/vector-db/search/index.test.ts src/views/vector-db/monitoring/index.test.ts`
Expected: PASS

**Step 5: 提交**

Run:
`git add web/src/views/vector-db/search/index.vue web/src/views/vector-db/monitoring/index.vue web/src/api/vector-db-domain.ts web/src/router/index.ts web/src/components/Layout/index.vue web/src/views/vector-db/search/index.test.ts web/src/views/vector-db/monitoring/index.test.ts && git commit -m "feat(web): add vector search and monitoring pages"`

### Task 5: 增加 P2 索引配置后端能力

**Files:**
- Create: `internal/vector-db/index_config_model.go`
- Create: `internal/vector-db/index_config_service.go`
- Create: `internal/vector-db/index_config_handler.go`
- Modify: `internal/vector-db/repository_sqlite.go`
- Modify: `migrations/002_vector_db.sql`
- Test: `internal/vector-db/index_config_service_test.go`

**Step 1: 写失败测试（索引参数更新与读取）**

新增 `TestIndexConfigService_UpdateAndGet_ShouldPersist`。

**Step 2: 运行测试确认失败**

Run: `go test ./internal/vector-db -run IndexConfig -v`
Expected: FAIL

**Step 3: 最小实现索引配置 API**

实现 HNSW/IVF 参数存取与校验。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/vector-db -run IndexConfig -v`
Expected: PASS

**Step 5: 提交**

Run:
`git add internal/vector-db/index_config_model.go internal/vector-db/index_config_service.go internal/vector-db/index_config_handler.go internal/vector-db/index_config_service_test.go internal/vector-db/repository_sqlite.go migrations/002_vector_db.sql && git commit -m "feat(vector-db): add index config management"`

### Task 6: 增加 P2 RBAC 与 API Key 权限后端

**Files:**
- Create: `internal/vector-db/rbac_model.go`
- Create: `internal/vector-db/rbac_service.go`
- Create: `internal/vector-db/rbac_middleware.go`
- Modify: `internal/router/router.go`
- Modify: `internal/vector-db/repository_sqlite.go`
- Modify: `migrations/002_vector_db.sql`
- Test: `internal/vector-db/rbac_service_test.go`
- Test: `internal/vector-db/rbac_middleware_test.go`

**Step 1: 写失败测试（角色权限判定）**

新增：
- `TestRBACService_CheckPermission_ShouldMatchRole`
- `TestRBACMiddleware_WhenForbidden_ShouldReturn403`

**Step 2: 运行测试确认失败**

Run: `go test ./internal/vector-db -run RBAC -v`
Expected: FAIL

**Step 3: 最小实现 RBAC 与 API Key hash 存储校验**

实现角色权限、API Key 权限校验中间件并挂载到 vector-db 路由。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/vector-db -run RBAC -v`
Expected: PASS

**Step 5: 提交**

Run:
`git add internal/vector-db/rbac_model.go internal/vector-db/rbac_service.go internal/vector-db/rbac_middleware.go internal/vector-db/rbac_service_test.go internal/vector-db/rbac_middleware_test.go internal/router/router.go internal/vector-db/repository_sqlite.go migrations/002_vector_db.sql && git commit -m "feat(vector-db): add rbac and api key authorization"`

### Task 7: 前端接入 P2 索引配置与权限管理

**Files:**
- Create: `web/src/views/vector-db/collections/IndexSettingsDialog.vue`
- Create: `web/src/views/vector-db/permissions/index.vue`
- Modify: `web/src/views/vector-db/collections/index.vue`
- Modify: `web/src/api/vector-db-domain.ts`
- Modify: `web/src/router/index.ts`
- Test: `web/src/views/vector-db/permissions/index.test.ts`

**Step 1: 写失败测试（权限页与索引配置交互）**

新增页面行为测试。

**Step 2: 运行测试确认失败**

Run: `cd web && npm run test:unit -- src/views/vector-db/permissions/index.test.ts`
Expected: FAIL

**Step 3: 最小实现页面**

新增权限页并在 collections 页接入索引配置弹窗。

**Step 4: 运行测试确认通过**

Run: `cd web && npm run test:unit -- src/views/vector-db/permissions/index.test.ts`
Expected: PASS

**Step 5: 提交**

Run:
`git add web/src/views/vector-db/collections/IndexSettingsDialog.vue web/src/views/vector-db/permissions/index.vue web/src/views/vector-db/collections/index.vue web/src/api/vector-db-domain.ts web/src/router/index.ts web/src/views/vector-db/permissions/index.test.ts && git commit -m "feat(web): add vector index settings and permissions UI"`

### Task 8: 增加 P3 限流能力（检索端点）

**Files:**
- Create: `internal/vector-db/rate_limit.go`
- Create: `internal/vector-db/rate_limit_test.go`
- Modify: `internal/vector-db/search_handler.go`
- Modify: `internal/router/router.go`

**Step 1: 写失败测试（超限返回 429）**

新增 `TestSearchHandler_WhenRateLimited_ShouldReturn429`。

**Step 2: 运行测试确认失败**

Run: `go test ./internal/vector-db -run RateLimit -v`
Expected: FAIL

**Step 3: 最小实现限流中间件与接入**

按 API Key、Collection 维度进行限流。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/vector-db -run RateLimit -v`
Expected: PASS

**Step 5: 提交**

Run:
`git add internal/vector-db/rate_limit.go internal/vector-db/rate_limit_test.go internal/vector-db/search_handler.go internal/router/router.go && git commit -m "feat(vector-db): add rate limiting for vector search APIs"`

### Task 9: 增加 P3 备份恢复后端

**Files:**
- Create: `internal/vector-db/backup_model.go`
- Create: `internal/vector-db/backup_service.go`
- Create: `internal/vector-db/backup_handler.go`
- Modify: `internal/vector-db/repository_sqlite.go`
- Modify: `migrations/002_vector_db.sql`
- Test: `internal/vector-db/backup_service_test.go`
- Test: `internal/vector-db/backup_handler_test.go`

**Step 1: 写失败测试（备份/恢复状态机）**

新增测试覆盖创建备份、查询列表、触发恢复。

**Step 2: 运行测试确认失败**

Run: `go test ./internal/vector-db -run Backup -v`
Expected: FAIL

**Step 3: 最小实现备份恢复任务 API**

实现任务落库、状态更新与错误审计。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/vector-db -run Backup -v`
Expected: PASS

**Step 5: 提交**

Run:
`git add internal/vector-db/backup_model.go internal/vector-db/backup_service.go internal/vector-db/backup_handler.go internal/vector-db/backup_service_test.go internal/vector-db/backup_handler_test.go internal/vector-db/repository_sqlite.go migrations/002_vector_db.sql && git commit -m "feat(vector-db): add backup and restore management"`

### Task 10: 前端接入 P3 备份管理

**Files:**
- Create: `web/src/views/vector-db/backup/index.vue`
- Modify: `web/src/api/vector-db-domain.ts`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/components/Layout/index.vue`
- Test: `web/src/views/vector-db/backup/index.test.ts`

**Step 1: 写失败测试（备份列表与操作按钮）**

新增页面测试：加载、创建备份、恢复触发。

**Step 2: 运行测试确认失败**

Run: `cd web && npm run test:unit -- src/views/vector-db/backup/index.test.ts`
Expected: FAIL

**Step 3: 最小实现备份页面**

实现备份列表、创建、恢复、重试 UI。

**Step 4: 运行测试确认通过**

Run: `cd web && npm run test:unit -- src/views/vector-db/backup/index.test.ts`
Expected: PASS

**Step 5: 提交**

Run:
`git add web/src/views/vector-db/backup/index.vue web/src/api/vector-db-domain.ts web/src/router/index.ts web/src/components/Layout/index.vue web/src/views/vector-db/backup/index.test.ts && git commit -m "feat(web): add vector backup and restore UI"`

### Task 11: 增加 P4 可视化后端采样接口

**Files:**
- Create: `internal/vector-db/visualization_model.go`
- Create: `internal/vector-db/visualization_service.go`
- Create: `internal/vector-db/visualization_handler.go`
- Modify: `internal/vector-db/search_service.go`
- Test: `internal/vector-db/visualization_service_test.go`

**Step 1: 写失败测试（降维采样接口）**

新增 `TestVisualizationService_GetScatterData_ShouldReturnPoints`。

**Step 2: 运行测试确认失败**

Run: `go test ./internal/vector-db -run Visualization -v`
Expected: FAIL

**Step 3: 最小实现可视化数据接口**

先实现最小散点数据（采样+简化降维）。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/vector-db -run Visualization -v`
Expected: PASS

**Step 5: 提交**

Run:
`git add internal/vector-db/visualization_model.go internal/vector-db/visualization_service.go internal/vector-db/visualization_handler.go internal/vector-db/visualization_service_test.go internal/vector-db/search_service.go && git commit -m "feat(vector-db): add visualization data APIs"`

### Task 12: 前端接入 P4 可视化页面

**Files:**
- Create: `web/src/views/vector-db/visualization/index.vue`
- Modify: `web/src/api/vector-db-domain.ts`
- Modify: `web/src/router/index.ts`
- Test: `web/src/views/vector-db/visualization/index.test.ts`

**Step 1: 写失败测试（散点图渲染与筛选）**

新增页面测试覆盖加载、筛选、错误态。

**Step 2: 运行测试确认失败**

Run: `cd web && npm run test:unit -- src/views/vector-db/visualization/index.test.ts`
Expected: FAIL

**Step 3: 最小实现可视化页面**

接入后端接口并展示基础散点图。

**Step 4: 运行测试确认通过**

Run: `cd web && npm run test:unit -- src/views/vector-db/visualization/index.test.ts`
Expected: PASS

**Step 5: 提交**

Run:
`git add web/src/views/vector-db/visualization/index.vue web/src/api/vector-db-domain.ts web/src/router/index.ts web/src/views/vector-db/visualization/index.test.ts && git commit -m "feat(web): add vector visualization page"`

### Task 13: 文档补齐与指标对照矩阵

**Files:**
- Modify: `docs/API_REFERENCE.md`
- Modify: `docs/USER_GUIDE.md`
- Create: `docs/plans/2026-03-02-vector-db-full-scope-delivery-matrix.md`

**Step 1: 写失败检查（矩阵缺项）**

手工列项目书条目与当前实现映射，标注空缺项。

**Step 2: 补文档最小实现**

更新 API、页面操作与验收矩阵。

**Step 3: 验证文档可追溯**

Run: `grep -n "vector-db" docs/API_REFERENCE.md docs/USER_GUIDE.md docs/plans/2026-03-02-vector-db-full-scope-delivery-matrix.md`
Expected: 命中新增章节

**Step 4: 提交**

Run:
`git add docs/API_REFERENCE.md docs/USER_GUIDE.md docs/plans/2026-03-02-vector-db-full-scope-delivery-matrix.md && git commit -m "docs(vector-db): add full scope delivery matrix and guides"`

### Task 14: 全量回归与最终门禁

**Files:**
- Modify: `docs/plans/2026-03-02-vector-db-full-scope-delivery-matrix.md`

**Step 1: 执行后端门禁**

Run: `make lint && make test && make build && go build ./cmd/gateway`
Expected: 全部通过

**Step 2: 执行前端门禁**

Run: `cd web && npm run typecheck && npm run build && npm run test:unit`
Expected: 全部通过

**Step 3: 更新完成矩阵状态**

将矩阵中状态刷新为“已完成/未完成/风险”，并附证据命令。

**Step 4: 提交**

Run:
`git add docs/plans/2026-03-02-vector-db-full-scope-delivery-matrix.md && git commit -m "chore(vector-db): finalize verification matrix"`

---

## 全程执行约束

- 每次仅修改本任务涉及文件，避免混入无关改动。
- 所有新增能力先写失败测试，再写最小实现（TDD）。
- 每个 Task 完成后立即运行对应验证。
- 每个 Task 保持独立提交，便于回滚与审阅。

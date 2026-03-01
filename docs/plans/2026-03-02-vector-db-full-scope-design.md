# Vector DB 全量交付设计（P1-P4）

日期：2026-03-02

## 1. 背景与目标

在已完成 P0（Collection 管理、导入任务、错误审计、前端导入闭环）的基础上，继续完成项目书中剩余能力（P1-P4），实现“可管理、可检索、可观测、可控权、可恢复、可视化”的向量数据库系统。

本设计目标：

- 补齐 P1-P4 全部未完成能力并形成闭环。
- 保持现有接口与数据兼容，最小化破坏性变更。
- 每阶段都可独立验收，最终以全量门禁通过作为完成判定。

## 2. 总体架构与边界

### 2.1 后端分层

- `internal/vector-db/`：领域层与编排层，新增 `search`、`monitoring`、`rbac`、`backup` 子能力。
- Handler：只做参数校验、响应结构、错误码映射。
- Service：业务编排、权限校验、状态机与审计写入。
- Repository：元数据存储（SQLite）和查询。
- Backend：向量后端（Qdrant）操作封装。

### 2.2 路由分区

- 管理面：`/api/admin/vector-db/*`
  - Collection/Import/Index/Alert/Backup/RBAC 管理。
- 检索面：`/api/v1/vector/collections/:name/*`
  - `search`、`recommend`、`vector by id`。

### 2.3 前端边界

- 现有 `collections` 页面保留。
- 新增分区：`import`、`search`、`monitoring`、`visualization`。
- 统一 API facade：`web/src/api/vector-db-domain.ts`。

## 3. 数据模型与接口契约

### 3.1 数据模型扩展

- `vector_collections`：新增索引参数（HNSW/IVF）配置字段。
- `vector_import_jobs`：新增 `file_type`、`batch_size`、`field_mapping`。
- `vector_audit_logs`：补充 action 分类与可选 `user_agent`。
- 新增 `vector_api_keys`：仅存 key hash 与权限、过期时间。
- 新增 `vector_backups`：备份/恢复任务元数据与状态。
- 新增 `vector_alert_rules`：阈值、持续时间、通知渠道、启停状态。

### 3.2 权限模型

- 角色：`admin` / `editor` / `viewer`。
- 权限：`collection:*`、`vector:*`、`import:execute`、`monitoring:read`、`backup:execute`。
- 管理面走后台鉴权；检索面走 API Key + 权限校验。

### 3.3 API 契约

- 检索：
  - `POST /api/v1/vector/collections/:name/search`
  - `POST /api/v1/vector/collections/:name/recommend`
  - `GET /api/v1/vector/collections/:name/vectors/:id`
- 告警：`/api/admin/vector-db/alerts/rules` CRUD + events 查询。
- 备份恢复：
  - `POST /api/admin/vector-db/collections/:name/backup`
  - `GET /api/admin/vector-db/collections/:name/backups`
  - `POST /api/admin/vector-db/collections/:name/restore`
  - `DELETE /api/admin/vector-db/collections/:name/backups/:id`

统一响应维持：`{ success, data, error }`。

## 4. 错误处理与可观测

### 4.1 错误处理

- 统一错误码族：
  - `VECTOR_BAD_REQUEST`
  - `VECTOR_UNAUTHORIZED`
  - `VECTOR_FORBIDDEN`
  - `VECTOR_NOT_FOUND`
  - `VECTOR_CONFLICT`
  - `VECTOR_RATE_LIMITED`
  - `VECTOR_BACKEND_UNAVAILABLE`
  - `VECTOR_INTERNAL_ERROR`

### 4.2 可观测

- 审计覆盖：Collection/Import/Search/Alert/Backup/RBAC/API Key。
- 指标覆盖：请求量、错误率、P50/P95、限流触发、导入/备份成功率。
- 统一日志字段：`trace_id`、`action`、`collection`、`status`、`latency_ms`。

## 5. 分阶段实施方案（A）

### 阶段 1（P1）

- 检索 API 完整链路。
- 监控聚合查询 + 告警规则 CRUD。
- 前端搜索页 + 监控告警页。

### 阶段 2（P2）

- 索引配置接口与 UI。
- RBAC + API Key 权限模型落地。
- 审计日志动作扩展。

### 阶段 3（P3）

- 检索端点限流（用户/API Key/集合维度）。
- 备份恢复任务与管理 API + 前端。

### 阶段 4（P4）

- 向量可视化接口与页面（先最小可用散点）。
- 性能压测与报告收口。

## 6. 验收与完成门禁

每阶段与最终都执行：

- `make lint`
- `make test`
- `make build`
- `go build ./cmd/gateway`
- `cd web && npm run typecheck && npm run build && npm run test:unit`

最终完成判定：

- 项目书未完成项全部转为“已完成”。
- 计划对照矩阵中“未完成项”为“无”。
- 必跑验证全部通过。

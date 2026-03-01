# Vector DB Phase 1 设计文档

## 1. 目标与范围

- 目标：将 `internal/vector-db` 从内存实现升级为可落地的 `SQLite 元数据 + Qdrant 集合管理`，保持现有管理 API 契约稳定。
- 范围：集合 CRUD、列表筛选、统计查询、错误映射、补偿机制、单元测试补齐。
- 非目标：导入任务（JSON/CSV/PDF）、通用检索 API、监控大盘扩展。

## 2. 现状问题

- 当前 `Service` 使用内存 map，进程重启后状态丢失。
- `qdrant.Client` 已接入基础连接能力，但集合级生命周期操作未完整进入业务路径。
- 迁移脚本已包含 `vector_collections` 等表，但业务层未使用 SQLite 承接元数据。

## 3. 方案概览

采用最小改动的双后端编排：

- 写路径：Qdrant 先行，SQLite 落元数据，失败时补偿。
- 读路径：SQLite 为主，统计优先 Qdrant，失败降级。
- 接口层保持不变：`internal/handler/admin` 不调整路由契约，仅替换 service 依赖实现。

## 4. 组件设计

### 4.1 Service 分层

- `collection.go` 负责业务编排与错误语义，不再持有 in-memory map。
- 新增 SQLite repository（建议 `repository_sqlite.go`）负责 SQL 与模型映射。
- Qdrant 通过 `internal/qdrant/client.go` 暴露集合级方法。

### 4.2 接口抽象

为便于 TDD 与解耦，引入最小接口：

- `CollectionRepository`
  - `Create(ctx, meta)`
  - `Get(ctx, name)`
  - `List(ctx, query)`
  - `Update(ctx, name, patch)`
  - `Delete(ctx, name, force)`
- `CollectionBackend`（Qdrant）
  - `CreateCollection(ctx, name, dimension, metric)`
  - `DeleteCollection(ctx, name)`
  - `GetCollectionInfo(ctx, name)`

## 5. 核心数据流

### 5.1 创建集合

1. 校验请求字段。
2. 调用 Qdrant 创建集合。
3. 写入 SQLite 元数据。
4. 若第 3 步失败，执行 Qdrant 删除补偿并返回错误。

### 5.2 删除集合

1. SQLite 校验集合存在。
2. 调用 Qdrant 删除集合。
3. 执行 SQLite 删除（按现有策略可软删/硬删）。
4. 若第 3 步失败，记录审计日志并返回可重试错误。

### 5.3 查询与统计

- `Get/List`：走 SQLite，保持筛选与分页语义稳定。
- `Stats`：优先 Qdrant 实时信息；Qdrant 失败时降级 SQLite 快照并记录 warning。

## 6. 错误处理与一致性

- 统一业务错误：`ErrCollectionExists`、`ErrCollectionNotFound`、`ErrBackendUnavailable`。
- Handler 只做 HTTP 映射与响应组装，错误上下文在 service/repository 保留。
- 审计日志至少覆盖 create/delete 的失败路径。

## 7. 测试策略（TDD）

- 先写 service 失败用例（Red）：
  - SQLite 写失败触发 Qdrant 回滚。
  - Qdrant 不可用时 create/delete 错误映射。
  - stats 在 Qdrant 故障时降级。
- 再补实现（Green），最后整理结构（Refactor）。
- 保留并扩展 handler 测试，确保 HTTP 语义不回归。

## 8. 验证与验收

- 必跑验证：
  - `make lint`
  - `go test ./...`
  - `go build ./cmd/gateway`
  - 如前端类型定义变化，再执行 `cd web && npm run typecheck`
- 验收标准：
  - 创建/删除后 Qdrant 与 SQLite 状态一致。
  - 列表/详情稳定来自 SQLite。
  - Qdrant 异常不阻断基础读取，stats 可降级。

## 9. 风险与回滚

- 风险：跨存储非事务一致性窗口。
- 缓解：严格定义操作顺序 + 失败补偿 + 审计日志。
- 回滚：保留当前 API 不变，可快速切回内存实现开关（仅限紧急情况）。

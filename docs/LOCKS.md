# 文件锁清单（双AI并行强制）

> 目的：避免并行开发时互相覆盖、冲突回滚困难、误提交他人改动。
> 规则：未登记不得改；未释放不得接管。

## 1. 使用规则

1. 开工前先登记一行，状态写 `LOCKED`。
2. 每个任务只允许一个 AI 持有同一文件/目录锁。
3. 若必须改同一文件，先将状态标记为 `SHARED_WINDOW`，并在备注写明先后顺序。
4. 完工后立即将状态改为 `RELEASED`，补齐结束时间。
5. 提交前核对“锁定文件/目录”与 `git add` 文件一致。
6. 未释放锁时，其他 AI 不得改动对应文件。

## 2. 命名规范（与 AGENTS.md 保持一致）

1. 分支：`<tool>/<task>`，示例：`codex/routing-cache`。
2. worktree：`.worktrees/<tool>-<task>`，示例：`.worktrees/codex-routing-cache`。
3. `tool` 统一小写：`codex`、`opencode`、`claude`。

## 3. 锁清单模板

| AI标识 | 工具 | 任务 | 分支 | Worktree | 锁定文件/目录 | 状态 | 开始时间 | 结束时间 | 备注 |
|---|---|---|---|---|---|---|---|---|---|
| AI-A | codex | routing-cache | codex/routing-cache | .worktrees/codex-routing-cache | internal/routing/** | LOCKED | 2026-03-01 10:00 |  |  |
| AI-B | opencode | cache-policy | opencode/cache-policy | .worktrees/opencode-cache-policy | internal/cache/**, web/src/views/cache/** | LOCKED | 2026-03-01 10:05 |  |  |

## 4. 当前活跃任务

| AI标识 | 工具 | 任务 | 分支 | Worktree | 锁定文件/目录 | 状态 | 开始时间 | 结束时间 | 备注 |
|---|---|---|---|---|---|---|---|---|---|
| codex | codex | usage-saved-token-stats | codex/parallel-dev-20260301 | .worktrees/codex-parallel-dev-20260301 | internal/storage/sqlite.go, internal/storage/sqlite_test.go, internal/handler/admin/usage.go, internal/handler/admin/usage_handler_test.go, web/src/views/usage/index.vue, web/src/views/usage/usage-overview.ts, web/src/views/usage/usage-overview.test.ts | RELEASED | 2026-03-01 09:20 | 2026-03-01 09:35 | /usage 命中节省 Token 与 stats 同口径改造 |
| codex | codex | cache-hit-token-fix | codex/cache-hit-token-fix | .worktrees/codex-cache-hit-token-fix | internal/handler/proxy.go, internal/handler/proxy_test.go, internal/handler/proxy_extended_test.go | RELEASED | 2026-03-01 09:54 | 2026-03-01 09:58 | 修复缓存命中 token 统计为 0（含入/出 Token） |
| codex | codex | usage-volcengine-token-repair | codex/session-20260301-103420 | .worktrees/codex-20260301-103420 | internal/provider/volcengine/**, internal/handler/proxy.go, internal/storage/sqlite.go, internal/handler/admin/usage.go, web/src/views/usage/index.vue, web/src/constants/pages/usage.ts | RELEASED | 2026-03-01 11:03 | 2026-03-01 11:20 | 修复 volcengine token=0 并新增 usage_source 标记 |
| codex | codex | streaming-trace-recording | codex/streaming-trace-recording | .worktrees/codex-streaming-trace-recording | internal/handler/proxy.go, internal/handler/proxy_test.go, docs/LOCKS.md | RELEASED | 2026-03-01 13:21 | 2026-03-01 13:26 | 修复流式请求缺少 http.response 追踪记录 |
| codex | codex | vector-db-b-alignment | codex/vector-db-b-alignment | .worktrees/codex-20260302-vector-db-b-alignment | internal/vector-db/{import_job.go,collection_handler.go,import_job_test.go}, internal/handler/admin/admin.go, web/src/views/vector-db/{import/**,monitoring/**}, web/src/router/index.ts, web/src/api/vector-db-domain.ts, docs/{API_REFERENCE.md,USER_GUIDE.md}, docs/plans/*vector-db* | RELEASED | 2026-03-02 08:10 | 2026-03-02 08:30 | 项目书B口径收口（导入取消/子组件结构/监控alerts子页/文档矩阵） |
| codex | codex | static-data-api-only-refactor | codex/static-data-api-only | .worktrees/codex-static-data-api-only | web/src/api/**, web/src/views/{providers-accounts,cache,settings,docs,home,dashboard,api-management}/**, web/src/views/routing/composables/useRoutingConsole.ts, web/src/constants/pages/**, docs/plans/2026-03-02-static-data-api-only-refactoring* | LOCKED | 2026-03-02 11:13 |  | 全面清理业务静态数据，改为 API 数据源 |

## 5. 历史记录

| AI标识 | 工具 | 任务 | 分支 | Worktree | 锁定文件/目录 | 状态 | 开始时间 | 结束时间 | 备注 |
|---|---|---|---|---|---|---|---|---|---|
| opencode-1 | opencode | request-trace | opencode/cache-detail | .worktrees/opencode-cache-detail | internal/tracing/**, internal/storage/sqlite.go, internal/handler/admin/trace*.go, internal/handler/proxy.go, web/src/views/trace/**, web/src/api/trace-domain.ts | RELEASED | 2026-03-01 08:30 | 2026-03-01 10:00 | OpenTelemetry链路追踪-完整实现（含10个关键Span） |
| codex | codex | vector-tier-routing-migration | codex/vector-tier-routing-migration | .worktrees/codex-vector-tier-routing-migration | internal/handler/admin/{admin.go,cache.go,types.go,cache_tier_test.go}, web/src/api/{routing-domain.ts,cache-domain.ts,routing-domain-tier.test.ts,cache-domain-tier.test.ts}, web/src/views/routing/{composables/useRoutingConsole.ts,components/VectorManagementTab.vue,routing-vector-scope.test.ts}, web/src/views/cache/{index.vue,cache-cold-tier-controls.test.ts} | RELEASED | 2026-03-01 12:35 | 2026-03-01 12:50 | 冷热向量分层从 /cache 迁移到 /routing，tier API 路径切换到 /router/vector/tier/* |

## 4. 状态值说明

1. `LOCKED`：独占锁定中，其他 AI 禁止改动。
2. `SHARED_WINDOW`：共享改动窗口，允许按约定顺序共同修改。
3. `RELEASED`：任务完成并释放，文件可被他人接管。

## 5. 开工与收工检查

1. 开工检查：已登记锁、分支命名正确、worktree 独立。
2. 收工检查：锁已释放、`fetch + rebase main` 已完成、仅提交本任务文件。

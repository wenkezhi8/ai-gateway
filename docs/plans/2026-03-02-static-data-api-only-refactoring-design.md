# Static Data API-Only Refactoring Design

## 目标

把前端业务静态数据（Provider/Model/Settings/Cache 等）从源码移除，统一改为 API 数据源，严格执行无回退策略：API 失败时显示错误并禁用相关功能。

## 设计决策

1. 前后端分离：前端先定义并消费接口，后端并行补齐实现。
2. API 路径风格：统一使用 `/admin/*`（公开页面使用已有公开接口）。
3. 元数据来源：Provider 的 label/color/logo/icon/endpoint/models 全部来自 API，不再保留前端常量。
4. 错误策略：不使用前端默认业务数据兜底；失败时显示错误态与重试按钮。

## 新增/扩展接口（前端契约）

### GET `/admin/providers/types`

返回 Provider 全量元数据：

- `id`
- `label`
- `category` (`international` | `chinese` | `local`)
- `color`
- `logo`
- `icon`
- `default_endpoint`
- `coding_endpoint`
- `supports_coding_plan`
- `models: string[]`

### GET `/admin/cache/task-ttl`

返回：

- `task_types[]`（任务类型、说明、默认 TTL）
- `model_options[]`（provider 与对应模型集合）

### GET `/admin/settings/defaults`

返回：

- `gateway`
- `cache`
- `logging`
- `security`

### GET `/v1/config/providers`（公开页复用）

用于 Home/Docs 的公开 provider 列表。

## 页面改造范围

- `web/src/views/providers-accounts/index.vue`
- `web/src/views/cache/index.vue`
- `web/src/views/settings/index.vue`
- `web/src/views/docs/index.vue`
- `web/src/views/home/index.vue`
- `web/src/views/dashboard/index.vue`
- `web/src/views/api-management/index.vue`
- `web/src/views/routing/composables/useRoutingConsole.ts`

## 常量清理范围

优先删除业务硬编码常量（Provider、Endpoint、Model、Settings 默认值、Cache TTL 业务值、Dashboard 业务回退）。

保留纯 UI 常量（菜单、告警枚举、CSV 表头、时间 tabs 等）。

## 测试策略

按 TDD 执行：

1. 先补失败测试（API 契约 + 页面行为 + 反硬编码回归）
2. 最小实现让测试通过
3. 重构并保持测试绿

验证门禁：

- `cd web && npm run test:unit`
- `cd web && npm run typecheck`
- `cd web && npm run build`

## 进度追踪

实施进度以 `docs/plans/2026-03-02-static-data-api-only-refactoring-implementation.md` 为唯一跟踪文档。

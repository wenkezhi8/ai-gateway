# Static Data API-Only Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 移除前端业务静态数据依赖，统一改为 API 数据源，API 失败走错误态且不做业务数据回退。

**Architecture:** 先建立 API 契约与 API 层函数，再逐页替换常量引用并补全错误态。通过反硬编码回归测试保证不回退到源码静态数据。按 P0/P1/P2 分批实施与提交。

**Tech Stack:** Vue 3 + TypeScript + Vitest + Element Plus + Axios API facade

---

## 本次进度看板

- [x] Task 1: Provider Types API 契约测试与实现
- [x] Task 2: providers-accounts 页面迁移到 API
- [x] Task 3: Provider 常量清理与反硬编码测试
- [ ] Task 4: Cache Task TTL API 契约测试与实现
- [ ] Task 5: cache 页面迁移到 API
- [ ] Task 6: Settings Defaults API 契约测试与实现
- [ ] Task 7: settings 页面迁移到 API
- [ ] Task 8: Public Providers API 契约测试与实现
- [ ] Task 9: docs/home 页面迁移到 API
- [ ] Task 10: dashboard 空状态替换回退数据
- [ ] Task 11: api-management/routing 移除 `deepseek-chat` 硬编码
- [ ] Task 12: 全量静态常量清理 + 回归测试完善
- [ ] Task 13: 门禁验证 + 提交整理

---

### Task 1: Provider Types API 契约测试与实现

**Files:**
- Modify: `web/src/api/provider.ts`
- Create/Modify: `web/src/api/provider.test.ts`

**Steps:**
1. 写失败测试：`getProviderTypes` 调用 `/admin/providers/types` 并返回结构化数据
2. 跑单测确认失败
3. 实现最小 API 函数与类型定义
4. 跑单测确认通过
5. 提交

### Task 2: providers-accounts 页面迁移到 API

**Files:**
- Modify: `web/src/views/providers-accounts/index.vue`
- Modify/Create: `web/src/views/providers-accounts/provider-name-nowrap.test.ts`

**Steps:**
1. 写失败测试：页面不再引用 providers 静态常量，改为调用 `getProviderTypes`
2. 跑单测确认失败
3. 实现页面数据加载、loading/error/retry 与禁用逻辑
4. 跑单测确认通过
5. 提交

### Task 3: Provider 常量清理与反硬编码测试

**Files:**
- Modify: `web/src/constants/pages/providers.ts`
- Modify: `web/src/constants/pages/providers-accounts.ts`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Steps:**
1. 写失败测试：禁止 provider 业务静态常量
2. 跑单测确认失败
3. 删除不再使用的业务常量与引用
4. 跑单测确认通过
5. 提交

### Task 4: Cache Task TTL API 契约测试与实现

**Files:**
- Modify/Create: `web/src/api/cache-domain-tier.test.ts`
- Modify/Create: `web/src/api/cache-domain.ts`

**Steps:**
1. 写失败测试：`getCacheTaskTTLConfig` 调用 `/admin/cache/task-ttl`
2. 跑单测确认失败
3. 实现最小 API 函数与类型定义
4. 跑单测确认通过
5. 提交

### Task 5: cache 页面迁移到 API

**Files:**
- Modify: `web/src/views/cache/index.vue`
- Modify: `web/src/views/cache/cache-tier-config.test.ts`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Steps:**
1. 写失败测试：页面不再使用 `CACHE_DEFAULT_TASK_TTL` / `CACHE_RULE_MODEL_OPTIONS`
2. 跑单测确认失败
3. 改为 API 数据源并补齐错误态
4. 跑单测确认通过
5. 提交

### Task 6: Settings Defaults API 契约测试与实现

**Files:**
- Modify: `web/src/api/settings-domain.ts`
- Modify: `web/src/api/settings-domain.test.ts`

**Steps:**
1. 写失败测试：`getSettingsDefaults` 调用 `/admin/settings/defaults`
2. 跑单测确认失败
3. 实现最小 API 函数与类型定义
4. 跑单测确认通过
5. 提交

### Task 7: settings 页面迁移到 API

**Files:**
- Modify: `web/src/views/settings/index.vue`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Steps:**
1. 写失败测试：页面不再依赖 `SETTINGS_DEFAULT_VALUES`
2. 跑单测确认失败
3. 页面改为 API 加载默认配置
4. 跑单测确认通过
5. 提交

### Task 8: Public Providers API 契约测试与实现

**Files:**
- Modify/Create: `web/src/api/domain-facades.test.ts`
- Modify/Create: `web/src/api/provider.ts`（或新 facade 文件）

**Steps:**
1. 写失败测试：`getPublicProviders` 调用 `/v1/config/providers`
2. 跑单测确认失败
3. 实现最小 API 函数
4. 跑单测确认通过
5. 提交

### Task 9: docs/home 页面迁移到 API

**Files:**
- Modify: `web/src/views/docs/index.vue`
- Modify: `web/src/views/home/index.vue`
- Modify: `web/src/constants/pages/docs.ts`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Steps:**
1. 写失败测试：docs/home 不再存在 provider 硬编码数组
2. 跑单测确认失败
3. 改为公开 API 数据源与错误态
4. 跑单测确认通过
5. 提交

### Task 10: dashboard 空状态替换回退数据

**Files:**
- Modify: `web/src/views/dashboard/index.vue`
- Modify: `web/src/constants/pages/dashboard.ts`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Steps:**
1. 写失败测试：dashboard 不再使用 `DASHBOARD_FALLBACK_SERIES`
2. 跑单测确认失败
3. 实现空状态 UI（无数据时）
4. 跑单测确认通过
5. 提交

### Task 11: api-management/routing 移除 `deepseek-chat` 硬编码

**Files:**
- Modify: `web/src/views/api-management/index.vue`
- Modify: `web/src/views/routing/composables/useRoutingConsole.ts`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Steps:**
1. 写失败测试：禁止 `deepseek-chat` 硬编码默认值
2. 跑单测确认失败
3. 改为仅使用 API 返回默认模型
4. 跑单测确认通过
5. 提交

### Task 12: 全量静态常量清理 + 回归测试完善

**Files:**
- Modify: `web/src/constants/pages/*.ts`（涉及文件）
- Modify: `web/src/constants/pages.static-config.test.ts`

**Steps:**
1. 写失败测试：覆盖所有已迁移业务静态常量
2. 跑单测确认失败
3. 清理残留常量与死引用
4. 跑单测确认通过
5. 提交

### Task 13: 门禁验证 + 提交整理

**Files:**
- Modify: 本任务涉及全部文件

**Steps:**
1. 运行 `cd web && npm run test:unit`
2. 运行 `cd web && npm run typecheck`
3. 运行 `cd web && npm run build`
4. 运行 `go test ./...`
5. 检查 `git status --short`，确保仅本任务改动
6. 按阶段提交并保持工作区 clean

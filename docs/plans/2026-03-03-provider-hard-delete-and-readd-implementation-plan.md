# Provider Hard Delete And Re-Add Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现“服务商真实删除（刷新/重启不复活）+ 删除后可从添加服务商重新加入”的完整链路。

**Architecture:** 后端将 `DELETE /api/admin/providers/:id` 升级为级联删除并持久化配置；前端删除统一调用该接口。前端新增“服务商目录”（基于 logo 文件名的元数据常量）与公开服务商合并，保证删除后仍可重加。模型评分 UI 不恢复，但删除时继续清理其底层模型关系记录，避免孤儿数据。

**Tech Stack:** Go + Gin（后端管理接口）、Vue 3 + TypeScript + Element Plus（前端）、Vitest（前端单测）、Go test（后端单测）

---

### Task 1: 后端删除行为测试先行（Red）

**Files:**
- Create: `internal/handler/admin/provider_delete_test.go`
- Modify: `internal/handler/admin/provider.go`
- Test: `internal/handler/admin/provider_delete_test.go`

**Step 1: Write the failing test**

```go
func TestProviderHandler_DeleteProvider_ShouldCascadeAndPersist(t *testing.T) {
    // 1) 准备临时 configs/config.json 与 data/accounts.json
    // 2) 构造 registry + accountManager + smartRouter
    // 3) 调用 DELETE /api/admin/providers/openai
    // 4) 断言：registry 不含 openai
    // 5) 断言：accounts.json 不含 provider/provider_type=openai
    // 6) 断言：provider defaults 与 model scores 已清理
    // 7) 断言：configs/config.json 的 providers 已移除 openai
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run TestProviderHandler_DeleteProvider_ShouldCascadeAndPersist -v`
Expected: FAIL，提示删除接口未执行级联清理或未持久化配置。

**Step 3: Write minimal implementation**

```go
// 在 ProviderHandler 中引入 router 与 configPath，执行统一级联删除
// 并返回删除统计：removed_accounts/removed_models/removed_defaults/updated_config
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run TestProviderHandler_DeleteProvider_ShouldCascadeAndPersist -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/admin/provider.go internal/handler/admin/provider_delete_test.go
git commit -m "feat(admin): cascade delete provider with persistent cleanup"
```

### Task 2: 后端路由接线与兼容验证

**Files:**
- Modify: `internal/handler/admin/admin.go`
- Modify: `internal/handler/admin/admin_routes_model_score_test.go`
- Test: `internal/handler/admin/admin_routes_model_score_test.go`

**Step 1: Write the failing test**

```go
func TestRegisterRoutes_DeleteProvider_ShouldReachHandler(t *testing.T) {
    // 用可观测 mock handler 验证 DELETE /api/admin/providers/:id 路由仍可达
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run TestRegisterRoutes_DeleteProvider_ShouldReachHandler -v`
Expected: FAIL（若构造器签名变更但路由未适配）。

**Step 3: Write minimal implementation**

```go
// 适配 NewProviderHandler 构造参数：registry/accountManager/smartRouter/configPath
// 保持现有路由路径不变
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run TestRegisterRoutes_DeleteProvider_ShouldReachHandler -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/admin/admin.go internal/handler/admin/admin_routes_model_score_test.go
git commit -m "refactor(admin): wire provider handler dependencies for hard delete"
```

### Task 3: 前端行为测试先行（Red）

**Files:**
- Modify: `web/src/views/model-management/index.test.ts`
- Create: `web/src/constants/providers-catalog.ts`
- Test: `web/src/views/model-management/index.test.ts`

**Step 1: Write the failing test**

```ts
it('allows deleting non-custom providers and keeps them re-addable from catalog', () => {
  const viewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')
  expect(viewFile).toContain('@click="handleDeleteProvider(row)"')
  expect(viewFile).not.toContain('row.custom')
  expect(viewFile).toContain('mergeProviderOptions')
  expect(viewFile).toContain('PROVIDER_CATALOG')
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/views/model-management/index.test.ts`
Expected: FAIL（尚未存在目录合并逻辑与删除条件调整）。

**Step 3: Write minimal implementation**

```ts
export const PROVIDER_CATALOG = [
  { id: 'openai', label: 'OpenAI', color: '#10A37F', logo: '/logos/openai.svg' },
  { id: 'anthropic', label: 'Anthropic', color: '#CC785C', logo: '/logos/anthropic.svg' }
  // ... logos 覆盖，中文名缺失时 label=id
]
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/views/model-management/index.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/model-management/index.test.ts web/src/constants/providers-catalog.ts
git commit -m "test(web): cover provider hard delete and re-add catalog behavior"
```

### Task 4: 前端删除与添加目录落地（Green）

**Files:**
- Modify: `web/src/views/model-management/index.vue`
- Modify: `web/src/api/provider.ts`
- Test: `web/src/api/provider.test.ts`

**Step 1: Write the failing test**

```ts
it('deletes provider via admin provider API and refreshes list', async () => {
  // 断言调用 /admin/providers/:id，而非仅删除 /admin/router/models
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/api/provider.test.ts src/views/model-management/index.test.ts`
Expected: FAIL

**Step 3: Write minimal implementation**

```ts
async function handleDeleteProvider(row: ProviderSetting) {
  await providerApi.delete(row.id)
  await loadSettings()
  ElMessage.success('服务商已删除')
}

function mergeProviderOptions(publicProviders: PublicProviderInfo[]) {
  // public + catalog 去重
}
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/api/provider.test.ts src/views/model-management/index.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/model-management/index.vue web/src/api/provider.ts web/src/api/provider.test.ts
git commit -m "feat(web): support hard delete and provider catalog re-add"
```

### Task 5: 全量验证与交付检查

**Files:**
- Modify: `docs/plans/2026-03-03-provider-hard-delete-and-readd-design.md`
- Modify: `docs/plans/2026-03-03-provider-hard-delete-and-readd-implementation-plan.md`

**Step 1: Run backend verification**

Run: `go test ./internal/handler/admin/... && go build ./cmd/gateway`
Expected: PASS

**Step 2: Run frontend verification**

Run: `cd web && npm run test:unit -- src/views/model-management/index.test.ts src/api/provider.test.ts && npm run typecheck && npm run build`
Expected: PASS

**Step 3: Integration restart check**

Run: `./scripts/dev-restart.sh`
Expected: 服务重启成功，`/model-management` 删除后刷新不复活，且下拉可重加。

**Step 4: Final commit**

```bash
git add docs/plans/2026-03-03-provider-hard-delete-and-readd-design.md docs/plans/2026-03-03-provider-hard-delete-and-readd-implementation-plan.md
git commit -m "docs(plan): add provider hard delete and re-add execution plan"
```

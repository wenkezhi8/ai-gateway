# Alerts Stats And Resolve Similar Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修正告警页“今日告警”统计口径并新增同类告警批量处理能力。

**Architecture:** 后端新增批量处理接口，对全部历史待处理告警按 `level+source+message` 聚合处理。前端新增“处理同类”入口并改为读取后端 stats 的 `todayTotal` 展示今日告警。通过后端与前端测试保证行为稳定。

**Tech Stack:** Go + Gin、Vue 3 + TypeScript、Vitest、Go test

---

### Task 1: 后端批量处理接口测试（RED）

**Files:**
- Create: `internal/handler/admin/alert_resolve_similar_test.go`
- Modify: `internal/handler/admin/alert.go`
- Test: `internal/handler/admin/alert_resolve_similar_test.go`

**Step 1: Write the failing test**

```go
func TestAlertHandler_ResolveSimilarAlerts_ShouldResolvePendingGroup(t *testing.T) {
    // 准备 pending/resolved 混合告警
    // 调用 POST /api/admin/alerts/resolve-similar
    // 断言仅同组 pending 被处理
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run TestAlertHandler_ResolveSimilarAlerts_ShouldResolvePendingGroup -v`
Expected: FAIL（接口未实现）。

**Step 3: Write minimal implementation**

```go
func (h *AlertHandler) ResolveSimilarAlerts(c *gin.Context) {
    // bind level/source/message
    // iterate alerts and resolve pending matches
    // save and return affected count
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run TestAlertHandler_ResolveSimilarAlerts_ShouldResolvePendingGroup -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/admin/alert.go internal/handler/admin/alert_resolve_similar_test.go
git commit -m "feat(alerts): add resolve-similar batch endpoint"
```

### Task 2: 路由接入与回归验证

**Files:**
- Modify: `internal/handler/admin/admin.go`
- Test: `internal/handler/admin/alert_resolve_similar_test.go`

**Step 1: Write the failing route assertion test**

```go
// 覆盖 POST /api/admin/alerts/resolve-similar 可达
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run ResolveSimilar -v`
Expected: FAIL（路由未注册或不可达）。

**Step 3: Write minimal implementation**

```go
alerts.POST("/resolve-similar", handlers.Alert.ResolveSimilarAlerts)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run ResolveSimilar -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/admin/admin.go
git commit -m "feat(alerts): register resolve-similar route"
```

### Task 3: 前端测试先行（RED）

**Files:**
- Create: `web/src/views/alerts/index.test.ts`
- Create: `web/src/api/alert.test.ts`
- Modify: `web/src/views/alerts/index.vue`
- Modify: `web/src/api/alert.ts`

**Step 1: Write the failing tests**

```ts
it('uses stats todayTotal and provides resolve-similar action', () => {
  // 断言 alertApi.getStats todayTotal resolveSimilar
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/views/alerts/index.test.ts src/api/alert.test.ts`
Expected: FAIL（方法/文案/调用未实现）。

**Step 3: Write minimal implementation**

```ts
resolveSimilar(payload) {
  return request.post('/admin/alerts/resolve-similar', payload)
}
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/views/alerts/index.test.ts src/api/alert.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/alerts/index.test.ts web/src/api/alert.test.ts web/src/views/alerts/index.vue web/src/api/alert.ts
git commit -m "feat(alerts): fix today stat and add resolve-similar action"
```

### Task 4: 全量验证

**Files:**
- Modify: `docs/plans/2026-03-03-alerts-stats-and-resolve-similar-design.md`
- Modify: `docs/plans/2026-03-03-alerts-stats-and-resolve-similar-implementation-plan.md`

**Step 1: Run backend tests**

Run: `go test ./internal/handler/admin/...`
Expected: PASS

**Step 2: Run frontend tests**

Run: `cd web && npm run test:unit -- src/views/alerts/index.test.ts src/api/alert.test.ts`
Expected: PASS

**Step 3: Run frontend type/build verification**

Run: `cd web && npm run typecheck && npm run build`
Expected: PASS

**Step 4: Restart and smoke check**

Run: `./scripts/dev-restart.sh`
Expected: `/alerts` 中今日告警来自 stats，点击“处理同类”可批量处理同组历史待处理告警。

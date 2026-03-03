# Alerts 今日统计修正与同类告警批量处理设计

## 背景

- `web/src/views/alerts/index.vue` 中“今日告警”当前用 `alerts.length` 计算，实际代表当前列表总数，不是“今日”口径。
- 告警风暴场景下（如周末相同告警重复触发），当前仅支持逐条处理，效率低。

## 目标

- “今日告警”改为后端统计口径（`todayTotal`）。
- 支持按 `级别 + 来源 + 告警信息` 对“全部历史待处理告警”批量处理。

## 方案概览

1. 后端新增批量处理接口 `POST /api/admin/alerts/resolve-similar`。
2. 前端新增“处理同类”按钮，调用批量接口。
3. 前端统计卡改为使用 `/admin/alerts/stats` 返回值展示“今日告警”。

## 详细设计

### 后端

- 在 `internal/handler/admin/alert.go` 新增 `ResolveSimilarAlerts`：
  - 入参：`level`, `source`, `message`。
  - 匹配条件：`status != resolved` 且三字段全匹配。
  - 处理动作：设置 `status=resolved`，写入 `resolvedAt`。
  - 输出：`affected`（处理条数）和 `key`（分组键）。
  - 处理后持久化 `alerts.json`。

- 在 `internal/handler/admin/admin.go` 注册路由：
  - `alerts.POST("/resolve-similar", handlers.Alert.ResolveSimilarAlerts)`。

### 前端

- 在 `web/src/api/alert.ts` 增加 `resolveSimilar(payload)` 方法。
- 在 `web/src/views/alerts/index.vue`：
  - 新增统计拉取函数 `fetchStats()`，使用 `alertApi.getStats()`；
  - `alertStats` 的“今日告警”改为 `todayTotal`；
  - 表格操作新增“处理同类”按钮与 `resolveSimilar(row)` 方法；
  - 批量处理成功后刷新 `fetchAlerts()` 与 `fetchStats()`。

## 边界与容错

- 入参缺失时返回 400。
- 若匹配结果为 0，返回成功但 `affected=0`，前端给出已处理 0 条提示。
- 仅处理未处理告警，已处理告警不重复写。

## 测试策略

- 后端测试：新增 `ResolveSimilarAlerts` 行为测试（含参数校验、仅处理 pending、影响条数断言）。
- 前端测试：新增 alerts 页静态断言（统计口径来源与“处理同类”入口存在）。
- API 测试：新增 `alertApi.resolveSimilar` 请求路径与参数断言。

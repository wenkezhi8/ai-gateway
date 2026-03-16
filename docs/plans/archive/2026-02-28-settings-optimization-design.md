# 设置页优化设计

## 概述
针对设置页面实现三项优化：节流批量提交、同步状态反馈、API 单测补齐。

## 目标
1. **性能优化**：Switch 开关即时保存，500ms 节流合并请求
2. **功能增强**：每个 Section 底部显示最后同步时间
3. **可维护性**：为 settings-domain 和 alert-domain 补齐完整单测

## 设计详情

### 1. 节流批量提交

在 `settings-domain.ts` 新增 throttled 版本函数：

```ts
let pendingPayload: UiSettingsPayload | null = null
let throttleTimer: ReturnType<typeof setTimeout> | null = null

export function updateGeneralUiSettingsThrottled(
  payload: Record<string, unknown>,
  delay = 500
): Promise<UiSettingsPayload>
```

**行为**：
- 多次调用时合并 payload
- 500ms 内无新调用则发送请求
- 返回 Promise 供调用方处理结果

### 2. 同步状态反馈

每个设置 Section 底部新增同步状态组件：

```vue
<div class="sync-status">
  <el-icon v-if="syncing"><Loading /></el-icon>
  <el-icon v-else-if="lastSyncAt"><Check /></el-icon>
  <span>{{ syncStatusText }}</span>
</div>
```

**状态**：
- `syncing`：正在同步
- `synced`：已同步，显示时间
- `error`：同步失败

### 3. 单测补齐

#### settings-domain.test.ts
覆盖函数：
- `getUiSettings`
- `updateUiSettings`
- `updateRoutingUiSettings`
- `updateModelManagementUiSettings`
- `updateGeneralUiSettings`
- `updateGeneralUiSettingsThrottled`

每个函数测试场景：
- 成功 envelope 响应
- 失败 envelope 响应
- plain payload 响应（如适用）

#### alert-domain.test.ts
覆盖函数：
- `getAlerts`
- `acknowledgeAlert`
- `resolveAlert`
- `acknowledgeAllAlerts`
- `clearResolvedAlerts`

每个函数测试场景同上。

## 改动文件
1. `web/src/api/settings-domain.ts` - 新增节流函数
2. `web/src/views/settings/index.vue` - 添加同步状态显示
3. `web/src/api/settings-domain.test.ts` - 新增单测
4. `web/src/api/alert-domain.test.ts` - 新增单测

## 风险与回滚
- 节流函数内部状态可能导致内存泄漏，需在组件卸载时清理
- 回滚点：删除新增函数和测试文件

## 版本建议
PATCH

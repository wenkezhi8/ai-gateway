# 设置页优化实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现设置页节流提交、同步状态反馈、API 单测补齐

**Architecture:** 在 settings-domain.ts 新增节流函数，设置页各 Section 底部添加同步状态，为两个 domain 文件补齐完整单测

**Tech Stack:** Vue 3, TypeScript, Vitest

---

### Task 1: 补齐 settings-domain.test.ts 单测

**Files:**
- Create: `web/src/api/settings-domain.test.ts`

**Step 1: 创建测试文件框架**

```ts
import { beforeEach, describe, expect, it, vi, afterEach } from 'vitest'
import { ApiError } from './envelope'
import {
  getUiSettings,
  updateUiSettings,
  updateRoutingUiSettings,
  updateModelManagementUiSettings,
  updateGeneralUiSettings
} from './settings-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('settings-domain', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.put.mockReset()
  })

  describe('getUiSettings', () => {
    it('unwraps success envelope', async () => {
      requestMock.get.mockResolvedValue({
        success: true,
        data: { routing: { auto_save_enabled: true } }
      })
      const data = await getUiSettings()
      expect(requestMock.get).toHaveBeenCalledWith('/admin/settings/ui')
      expect(data.routing?.auto_save_enabled).toBe(true)
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.get.mockResolvedValue({
        success: false,
        error: { code: 'not_found', message: 'Settings not found' }
      })
      await expect(getUiSettings()).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.get.mockResolvedValue({ routing: { auto_save_enabled: false } })
      const data = await getUiSettings()
      expect(data.routing?.auto_save_enabled).toBe(false)
    })
  })

  describe('updateUiSettings', () => {
    it('unwraps success envelope', async () => {
      requestMock.put.mockResolvedValue({
        success: true,
        data: { routing: { auto_save_enabled: true } }
      })
      const data = await updateUiSettings({ routing: { auto_save_enabled: true } })
      expect(requestMock.put).toHaveBeenCalledWith('/admin/settings/ui', { routing: { auto_save_enabled: true } })
      expect(data.routing?.auto_save_enabled).toBe(true)
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.put.mockResolvedValue({
        success: false,
        error: { code: 'invalid_request', message: 'Invalid payload' }
      })
      await expect(updateUiSettings({})).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.put.mockResolvedValue({ routing: { auto_save_enabled: true } })
      const data = await updateUiSettings({})
      expect(data.routing?.auto_save_enabled).toBe(true)
    })
  })

  describe('updateRoutingUiSettings', () => {
    it('calls updateUiSettings with routing payload', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { routing: { auto_save_enabled: true } } })
      await updateRoutingUiSettings({ auto_save_enabled: true })
      expect(requestMock.put).toHaveBeenCalledWith('/admin/settings/ui', { routing: { auto_save_enabled: true } })
    })
  })

  describe('updateModelManagementUiSettings', () => {
    it('calls updateUiSettings with model_management payload', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { model_management: { last_saved_at: '2026-02-28' } } })
      await updateModelManagementUiSettings({ last_saved_at: '2026-02-28' })
      expect(requestMock.put).toHaveBeenCalledWith('/admin/settings/ui', { model_management: { last_saved_at: '2026-02-28' } })
    })
  })

  describe('updateGeneralUiSettings', () => {
    it('calls updateUiSettings with settings payload', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { settings: { theme: 'dark' } } })
      await updateGeneralUiSettings({ theme: 'dark' })
      expect(requestMock.put).toHaveBeenCalledWith('/admin/settings/ui', { settings: { theme: 'dark' } })
    })
  })
})
```

**Step 2: 运行测试验证**

```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit -- src/api/settings-domain.test.ts
```

Expected: PASS

---

### Task 2: 补齐 alert-domain.test.ts 单测

**Files:**
- Create: `web/src/api/alert-domain.test.ts`

**Step 1: 创建测试文件**

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from './envelope'
import {
  getAlerts,
  acknowledgeAlert,
  resolveAlert,
  acknowledgeAllAlerts,
  clearResolvedAlerts
} from './alert-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('alert-domain', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.put.mockReset()
    requestMock.post.mockReset()
    requestMock.delete.mockReset()
  })

  describe('getAlerts', () => {
    it('unwraps success envelope', async () => {
      requestMock.get.mockResolvedValue({
        success: true,
        data: { alerts: [{ id: '1', status: 'firing' }] }
      })
      const data = await getAlerts()
      expect(requestMock.get).toHaveBeenCalledWith('/admin/alerts', { silent: true })
      expect(data.alerts[0].id).toBe('1')
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.get.mockResolvedValue({
        success: false,
        error: { code: 'internal_error', message: 'Failed to fetch' }
      })
      await expect(getAlerts()).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.get.mockResolvedValue({ alerts: [{ id: '2', status: 'resolved' }] })
      const data = await getAlerts()
      expect(data.alerts[0].id).toBe('2')
    })
  })

  describe('acknowledgeAlert', () => {
    it('unwraps success envelope', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { id: '1', status: 'acknowledged' } })
      const data = await acknowledgeAlert('1')
      expect(requestMock.put).toHaveBeenCalledWith('/admin/alerts/1/acknowledge', {})
      expect(data.id).toBe('1')
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.put.mockResolvedValue({
        success: false,
        error: { code: 'not_found', message: 'Alert not found' }
      })
      await expect(acknowledgeAlert('999')).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.put.mockResolvedValue({ id: '1', status: 'acknowledged' })
      const data = await acknowledgeAlert('1')
      expect(data.id).toBe('1')
    })
  })

  describe('resolveAlert', () => {
    it('unwraps success envelope', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { id: '1', status: 'resolved' } })
      const data = await resolveAlert('1')
      expect(requestMock.put).toHaveBeenCalledWith('/admin/alerts/1/resolve', {})
      expect(data.id).toBe('1')
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.put.mockResolvedValue({
        success: false,
        error: { code: 'not_found', message: 'Alert not found' }
      })
      await expect(resolveAlert('999')).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.put.mockResolvedValue({ id: '1', status: 'resolved' })
      const data = await resolveAlert('1')
      expect(data.id).toBe('1')
    })
  })

  describe('acknowledgeAllAlerts', () => {
    it('unwraps success envelope', async () => {
      requestMock.post.mockResolvedValue({ success: true, data: { count: 5 } })
      const data = await acknowledgeAllAlerts()
      expect(requestMock.post).toHaveBeenCalledWith('/admin/alerts/acknowledge-all', {})
      expect(data.count).toBe(5)
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.post.mockResolvedValue({
        success: false,
        error: { code: 'internal_error', message: 'Failed' }
      })
      await expect(acknowledgeAllAlerts()).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.post.mockResolvedValue({ count: 3 })
      const data = await acknowledgeAllAlerts()
      expect(data.count).toBe(3)
    })
  })

  describe('clearResolvedAlerts', () => {
    it('unwraps success envelope', async () => {
      requestMock.delete.mockResolvedValue({ success: true, data: { count: 2 } })
      const data = await clearResolvedAlerts()
      expect(requestMock.delete).toHaveBeenCalledWith('/admin/alerts/clear-resolved')
      expect(data.count).toBe(2)
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.delete.mockResolvedValue({
        success: false,
        error: { code: 'internal_error', message: 'Failed' }
      })
      await expect(clearResolvedAlerts()).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.delete.mockResolvedValue({ count: 1 })
      const data = await clearResolvedAlerts()
      expect(data.count).toBe(1)
    })
  })
})
```

**Step 2: 运行测试验证**

```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit -- src/api/alert-domain.test.ts
```

Expected: PASS

---

### Task 3: 实现节流批量提交函数

**Files:**
- Modify: `web/src/api/settings-domain.ts`

**Step 1: 添加节流函数**

在 `settings-domain.ts` 末尾添加：

```ts
let pendingPayload: UiSettingsPayload | null = null
let throttleTimer: ReturnType<typeof setTimeout> | null = null
let pendingResolvers: { resolve: (value: UiSettingsPayload) => void; reject: (error: unknown) => void }[] = []

export function updateGeneralUiSettingsThrottled(
  payload: Record<string, unknown>,
  delay = 500
): Promise<UiSettingsPayload> {
  pendingPayload = {
    ...pendingPayload,
    settings: { ...pendingPayload?.settings, ...payload }
  }

  if (throttleTimer) {
    clearTimeout(throttleTimer)
  }

  return new Promise((resolve, reject) => {
    pendingResolvers.push({ resolve, reject })
    throttleTimer = setTimeout(async () => {
      throttleTimer = null
      const currentPayload = pendingPayload
      const currentResolvers = pendingResolvers
      pendingPayload = null
      pendingResolvers = []

      try {
        const result = await updateUiSettings(currentPayload!)
        currentResolvers.forEach(r => r.resolve(result))
      } catch (e) {
        currentResolvers.forEach(r => r.reject(e))
      }
    }, delay)
  })
}

export function flushThrottledSettings(): void {
  if (throttleTimer) {
    clearTimeout(throttleTimer)
    throttleTimer = null
  }
  pendingPayload = null
  pendingResolvers = []
}
```

**Step 2: 运行测试验证**

```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit -- src/api/settings-domain.test.ts
```

Expected: PASS

---

### Task 4: 为节流函数添加测试

**Files:**
- Modify: `web/src/api/settings-domain.test.ts`

**Step 1: 添加节流函数测试**

在测试文件末尾添加：

```ts
import { flushThrottledSettings, updateGeneralUiSettingsThrottled } from './settings-domain'

describe('updateGeneralUiSettingsThrottled', () => {
  beforeEach(() => {
    flushThrottledSettings()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('merges multiple calls within delay period', async () => {
    requestMock.put.mockResolvedValue({ success: true, data: { settings: { a: 1, b: 2 } } })

    const p1 = updateGeneralUiSettingsThrottled({ a: 1 }, 500)
    const p2 = updateGeneralUiSettingsThrottled({ b: 2 }, 500)

    vi.advanceTimersByTime(500)
    await Promise.all([p1, p2])

    expect(requestMock.put).toHaveBeenCalledTimes(1)
    expect(requestMock.put).toHaveBeenCalledWith('/admin/settings/ui', { settings: { a: 1, b: 2 } })
  })

  it('resolves all promises with same result', async () => {
    requestMock.put.mockResolvedValue({ success: true, data: { settings: { x: 1 } } })

    const p1 = updateGeneralUiSettingsThrottled({ x: 1 }, 500)
    const p2 = updateGeneralUiSettingsThrottled({ x: 2 }, 500)

    vi.advanceTimersByTime(500)
    const [r1, r2] = await Promise.all([p1, p2])

    expect(r1).toEqual(r2)
  })

  it('rejects all promises on error', async () => {
    requestMock.put.mockResolvedValue({
      success: false,
      error: { code: 'error', message: 'fail' }
    })

    const p1 = updateGeneralUiSettingsThrottled({ a: 1 }, 500)
    vi.advanceTimersByTime(500)

    await expect(p1).rejects.toThrow(ApiError)
  })
})
```

**Step 2: 运行测试验证**

```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit -- src/api/settings-domain.test.ts
```

Expected: PASS

---

### Task 5: 在设置页添加同步状态反馈

**Files:**
- Modify: `web/src/views/settings/index.vue`

**Step 1: 添加同步状态类型和响应式数据**

在 `<script setup>` 中添加：

```ts
interface SyncStatus {
  syncing: boolean
  lastSyncAt: string | null
  error: string | null
}

const syncStatusMap = reactive<Record<string, SyncStatus>>({
  appearance: { syncing: false, lastSyncAt: null, error: null },
  gateway: { syncing: false, lastSyncAt: null, error: null },
  cache: { syncing: false, lastSyncAt: null, error: null },
  logging: { syncing: false, lastSyncAt: null, error: null },
  security: { syncing: false, lastSyncAt: null, error: null }
})

function formatSyncTime(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function getSyncStatusText(section: string): string {
  const status = syncStatusMap[section]
  if (!status) return ''
  if (status.syncing) return '同步中...'
  if (status.error) return `同步失败: ${status.error}`
  if (status.lastSyncAt) return `最后同步: ${formatSyncTime(status.lastSyncAt)}`
  return ''
}

async function syncSection(section: string, payload: Record<string, unknown>) {
  const status = syncStatusMap[section]
  if (!status) return

  status.syncing = true
  status.error = null

  try {
    await updateGeneralUiSettingsThrottled(payload)
    status.lastSyncAt = new Date().toISOString()
  } catch (e: any) {
    status.error = e.message || '未知错误'
  } finally {
    status.syncing = false
  }
}
```

**Step 2: 在各 Section 底部添加同步状态显示**

在每个 `<el-card v-show="activeSection === 'xxx'">` 的 `</el-form>` 后添加：

```vue
<div class="sync-status">
  <el-icon v-if="syncStatusMap[activeSection]?.syncing" class="is-loading"><Loading /></el-icon>
  <el-icon v-else-if="syncStatusMap[activeSection]?.lastSyncAt"><Check /></el-icon>
  <span>{{ getSyncStatusText(activeSection) }}</span>
</div>
```

**Step 3: 添加样式**

```scss
.sync-status {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  margin-top: var(--spacing-lg);
  padding-top: var(--spacing-md);
  border-top: 1px solid var(--border-color);
  font-size: var(--font-size-sm);
  color: var(--text-tertiary);

  .el-icon {
    font-size: 14px;
  }

  .is-loading {
    animation: spin 1s linear infinite;
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
```

**Step 4: 导入图标**

```ts
import { Loading, Check } from '@element-plus/icons-vue'
```

**Step 5: 运行 typecheck 和 build**

```bash
cd /Users/openclaw/ai-gateway/web && npm run typecheck && npm run build
```

Expected: PASS

---

### Task 6: 运行完整验证

**Step 1: 运行所有单测**

```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit
```

Expected: PASS

**Step 2: 运行 typecheck 和 build**

```bash
cd /Users/openclaw/ai-gateway/web && npm run typecheck && npm run build
```

Expected: PASS

---

### Task 7: 提交代码

```bash
git add web/src/api/settings-domain.ts web/src/api/settings-domain.test.ts web/src/api/alert-domain.test.ts web/src/views/settings/index.vue docs/plans/2026-02-28-settings-optimization-design.md docs/plans/2026-02-28-settings-optimization-plan.md
git commit -m "feat(settings): add throttled save, sync status feedback and API unit tests"
```

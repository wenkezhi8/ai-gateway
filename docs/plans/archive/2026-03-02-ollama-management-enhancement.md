# Ollama 管理页面独立化 + 向量管理菜单调整

> **For OpenCode:** 使用 executing-plans skill 逐任务执行此计划。

**目标**：从 routing 页面剥离 Ollama 管理，创建独立页面 `/ollama`，并调整版本可见性。

**核心变更**：
- **Ollama 管理**：标准版 + 企业版可见，侧边栏菜单
- **向量管理**：仅企业版可见，保持 Header 显示

**技术栈**：
- 后端：复用现有 `internal/handler/admin/ollama.go` 接口
- 前端：Vue 3, TypeScript, Pinia, Element Plus
- 测试：Vitest

---

## 需求确认

### 功能可见性矩阵

| 功能 | 基础版 | 标准版 | 企业版 |
|------|-------|-------|-------|
| Ollama 管理 | ❌ | ✅ | ✅ |
| 向量管理 | ❌ | ❌ | ✅ |
| 知识库 | ❌ | ❌ | ✅ |

### 变更详情

**1. Ollama 管理**
- 从 `/routing` 页面完全移除
- 新建独立页面 `/ollama`
- 侧边栏新增"Ollama 管理"菜单项
- 标准 + 企业版可见

**2. 向量管理**
- **保持当前实现**（仅企业版 Header 显示）
- 不在侧边栏添加菜单
- 版本守卫保持 `['enterprise']`

---

## Task 1: 探索现有 Ollama 代码

**目标**：了解现有 Ollama 相关代码结构

**步骤**：
1. 搜索 Ollama 相关 API 接口
2. 查找 Ollama 相关前端组件
3. 确认可复用的部分

---

## Task 2: 后端 API 封装

**Files:**
- Create: `web/src/api/ollama-domain.ts`

**说明**：复用现有后端接口
- `GET /api/admin/ollama/models` - 获取模型列表
- `POST /api/admin/ollama/models/{name}/load` - 加载模型
- `POST /api/admin/ollama/models/{name}/unload` - 卸载模型
- `GET /api/admin/ollama/running` - 获取运行中的模型

**Step 1: Write minimal implementation**

```typescript
// web/src/api/ollama-domain.ts
import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface OllamaModel {
  name: string
  size: string
  modified_at: string
  details?: {
    format: string
    family: string
    parameter_size: string
  }
}

export interface RunningModel {
  name: string
  model: string
  size: number
  digest: string
  details: {
    format: string
    family: string
    parameter_size: string
  }
}

export async function getOllamaModels() {
  const raw = await request.get('/admin/ollama/models')
  return unwrapEnvelope<OllamaModel[]>(raw, { allowPlain: true })
}

export async function getRunningModels() {
  const raw = await request.get('/admin/ollama/running')
  return unwrapEnvelope<RunningModel[]>(raw, { allowPlain: true })
}

export async function loadModel(name: string) {
  const raw = await request.post(`/admin/ollama/models/${encodeURIComponent(name)}/load`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function unloadModel(name: string) {
  const raw = await request.post(`/admin/ollama/models/${encodeURIComponent(name)}/unload`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function pullModel(name: string) {
  const raw = await request.post('/admin/ollama/pull', { name })
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function deleteModel(name: string) {
  const raw = await request.delete(`/admin/ollama/models/${encodeURIComponent(name)}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}
```

**Step 2: Verify**

Run: `cd web && npm run typecheck`
Expected: 类型检查通过

**Step 3: Commit**

```bash
git add web/src/api/ollama-domain.ts
git commit -m "feat(api): add Ollama domain API client"
```

---

## Task 3: 前端 Store

**Files:**
- Create: `web/src/store/domain/ollama.ts`
- Create: `web/src/store/domain/ollama.test.ts`

**Step 1: Write failing test**

```typescript
describe('ollama store', () => {
  it('should fetch ollama models', async () => {
    const store = useOllamaStore()
    
    // Mock API
    ollamaApiMock.getOllamaModels.mockResolvedValue([
      { name: 'llama2:7b', size: '3.8 GB', modified_at: '2024-01-01' }
    ])
    
    await store.fetchModels()
    
    expect(store.models).toHaveLength(1)
    expect(store.models[0].name).toBe('llama2:7b')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit ollama.test.ts`
Expected: FAIL（文件不存在）

**Step 3: Write minimal implementation**

```typescript
// web/src/store/domain/ollama.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  getOllamaModels,
  getRunningModels,
  loadModel,
  unloadModel,
  pullModel,
  deleteModel,
  type OllamaModel,
  type RunningModel
} from '@/api/edition-domain'

export const useOllamaStore = defineStore('ollama-domain', () => {
  const models = ref<OllamaModel[]>([])
  const runningModels = ref<RunningModel[]>([])
  const loading = ref(false)
  const error = ref('')

  async function fetchModels() {
    loading.value = true
    error.value = ''
    try {
      models.value = await getOllamaModels()
    } catch (err) {
      error.value = err instanceof Error ? err.message : '获取模型列表失败'
    } finally {
      loading.value = false
    }
  }

  async function fetchRunningModels() {
    try {
      runningModels.value = await getRunningModels()
    } catch (err) {
      console.error('Failed to fetch running models:', err)
    }
  }

  async function load(name: string) {
    try {
      await loadModel(name)
      await fetchRunningModels()
    } catch (err) {
      throw err
    }
  }

  async function unload(name: string) {
    try {
      await unloadModel(name)
      await fetchRunningModels()
    } catch (err) {
      throw err
    }
  }

  async function pull(name: string) {
    try {
      await pullModel(name)
      await fetchModels()
    } catch (err) {
      throw err
    }
  }

  async function remove(name: string) {
    try {
      await deleteModel(name)
      await fetchModels()
    } catch (err) {
      throw err
    }
  }

  return {
    models,
    runningModels,
    loading,
    error,
    fetchModels,
    fetchRunningModels,
    load,
    unload,
    pull,
    remove
  }
})
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit ollama.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/store/domain/ollama.ts web/src/store/domain/ollama.test.ts
git commit -m "feat(store): add Ollama store with model management"
```

---

## Task 4: Ollama 管理页面

**Files:**
- Create: `web/src/views/ollama/index.vue`
- Create: `web/src/views/ollama/index.test.ts`

**Step 1: Write failing test**

```typescript
describe('Ollama management page', () => {
  it('should render model list', () => {
    const wrapper = mount(OllamaPage, {
      global: { plugins: [createTestingPinia()] }
    })
    
    expect(wrapper.find('.ollama-page').exists()).toBe(true)
    expect(wrapper.text()).toContain('Ollama 模型管理')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit ollama/index.test.ts`
Expected: FAIL

**Step 3: Write minimal implementation**

```vue
<template>
  <div class="ollama-page">
    <div class="page-header">
      <h2>Ollama 模型管理</h2>
      <el-button type="primary" @click="showPullDialog = true">
        <el-icon><Plus /></el-icon>
        下载模型
      </el-button>
    </div>

    <!-- 运行中的模型 -->
    <el-card class="section-card">
      <template #header>
        <div class="card-header">
          <span>运行中的模型</span>
          <el-button text @click="refreshRunning">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </template>
      <el-table :data="ollamaStore.runningModels" v-loading="ollamaStore.loading">
        <el-table-column prop="name" label="模型名称" />
        <el-table-column prop="size" label="大小">
          <template #default="{ row }">
            {{ formatSize(row.size) }}
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="{ row }">
            <el-button type="danger" size="small" @click="handleUnload(row.name)">
              停止
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 本地模型列表 -->
    <el-card class="section-card">
      <template #header>
        <div class="card-header">
          <span>本地模型</span>
          <el-button text @click="refreshModels">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </template>
      <el-table :data="ollamaStore.models" v-loading="ollamaStore.loading">
        <el-table-column prop="name" label="模型名称" />
        <el-table-column prop="size" label="大小" />
        <el-table-column prop="modified_at" label="修改时间" />
        <el-table-column label="操作">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleLoad(row.name)">
              加载
            </el-button>
            <el-button type="danger" size="small" @click="handleDelete(row.name)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 下载模型对话框 -->
    <el-dialog v-model="showPullDialog" title="下载模型">
      <el-form>
        <el-form-item label="模型名称">
          <el-input v-model="pullModelName" placeholder="例如: llama2:7b" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPullDialog = false">取消</el-button>
        <el-button type="primary" @click="handlePull">下载</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useOllamaStore } from '@/store/domain/ollama'

const ollamaStore = useOllamaStore()

const showPullDialog = ref(false)
const pullModelName = ref('')

onMounted(async () => {
  await Promise.all([
    ollamaStore.fetchModels(),
    ollamaStore.fetchRunningModels()
  ])
})

function formatSize(bytes: number): string {
  return (bytes / 1024 / 1024 / 1024).toFixed(2) + ' GB'
}

async function refreshModels() {
  await ollamaStore.fetchModels()
}

async function refreshRunning() {
  await ollamaStore.fetchRunningModels()
}

async function handleLoad(name: string) {
  try {
    await ollamaStore.load(name)
    ElMessage.success('模型加载成功')
  } catch (err) {
    ElMessage.error('加载失败')
  }
}

async function handleUnload(name: string) {
  try {
    await ollamaStore.unload(name)
    ElMessage.success('模型已停止')
  } catch (err) {
    ElMessage.error('停止失败')
  }
}

async function handleDelete(name: string) {
  try {
    await ElMessageBox.confirm('确定删除该模型吗？', '确认', { type: 'warning' })
    await ollamaStore.remove(name)
    ElMessage.success('删除成功')
  } catch (err) {
    // 用户取消
  }
}

async function handlePull() {
  if (!pullModelName.value) return
  
  try {
    await ollamaStore.pull(pullModelName.value)
    ElMessage.success('开始下载模型')
    showPullDialog.value = false
    pullModelName.value = ''
  } catch (err) {
    ElMessage.error('下载失败')
  }
}
</script>

<style scoped lang="scss">
.ollama-page {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit ollama/index.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/ollama/
git commit -m "feat(ollama): add Ollama management page"
```

---

## Task 5: 从 Routing 页面移除 Ollama 相关代码

**Files:**
- Modify: `web/src/views/routing/index.vue`

**Step 1: 移除 Ollama 相关区块**

在 routing 页面中：
- 删除 Ollama 模型管理相关的 UI 组件
- 删除 Ollama 相关的 import
- 删除 Ollama 相关的状态和方法

**Step 2: Verify**

Run: `cd web && npm run typecheck && npm run build`
Expected: 通过

**Step 3: Commit**

```bash
git add web/src/views/routing/index.vue
git commit -m "refactor(routing): remove Ollama management section"
```

---

## Task 6: 路由注册

**Files:**
- Modify: `web/src/router/index.ts`

**Step 1: Add Ollama route**

```typescript
{
  path: '/ollama',
  name: 'Ollama',
  component: () => import('@/views/ollama/index.vue'),
  meta: { title: 'Ollama 管理', requiresAuth: true }
}
```

**Step 2: Verify**

Run: `cd web && npm run typecheck`
Expected: 通过

**Step 3: Commit**

```bash
git add web/src/router/index.ts
git commit -m "feat(router): add Ollama management route"
```

---

## Task 7: 菜单配置

**Files:**
- Modify: `web/src/components/Layout/menu-config.ts`
- Modify: `web/src/components/Layout/menu-config.test.ts`

**Step 1: Add Ollama menu item**

```typescript
import type { EditionConfig } from '@/api/edition-domain'
import { DASHBOARD_ROUTE } from '../../constants/navigation'

export interface MenuItem {
  path: string
  title: string
  icon: string
  minEdition?: 'standard' | 'enterprise'
}

const ALL_MENUS: MenuItem[] = [
  { path: DASHBOARD_ROUTE, title: '监控仪表盘', icon: 'Monitor' },
  { path: '/ops', title: '运维监控', icon: 'Operation' },
  { path: '/chat', title: 'AI 对话', icon: 'ChatDotRound' },
  { path: '/api-management', title: 'API 管理', icon: 'Connection' },
  { path: '/model-management', title: '模型管理', icon: 'Collection' },
  { path: '/providers-accounts', title: '账号与限额', icon: 'Key' },
  { path: '/usage', title: 'API 使用统计', icon: 'DataLine' },
  { path: '/trace', title: '请求链路追踪', icon: 'Share' },
  { path: '/routing', title: '路由策略', icon: 'Guide' },
  { path: '/cache', title: '缓存管理', icon: 'Box' },
  { path: '/ollama', title: 'Ollama 管理', icon: 'Cpu', minEdition: 'standard' },
  { path: '/alerts', title: '告警管理', icon: 'Bell' },
  { path: '/settings', title: '系统设置', icon: 'Setting' }
]

const EDITION_LEVEL = { basic: 1, standard: 2, enterprise: 3 }

export function getMenuItems(edition: EditionConfig | null): MenuItem[] {
  if (!edition) {
    return ALL_MENUS.filter(m => !m.minEdition)
  }
  
  const currentLevel = EDITION_LEVEL[edition.type] || 1
  
  return ALL_MENUS.filter(menu => {
    if (!menu.minEdition) return true
    
    const requiredLevel = EDITION_LEVEL[menu.minEdition]
    return currentLevel >= requiredLevel
  })
}
```

**Step 2: Update test**

```typescript
describe('getMenuItems', () => {
  it('should show Ollama menu for standard edition', () => {
    const menus = getMenuItems({
      type: 'standard',
      // ...
    })
    
    expect(menus.find(m => m.path === '/ollama')).toBeDefined()
  })
  
  it('should hide Ollama menu for basic edition', () => {
    const menus = getMenuItems({
      type: 'basic',
      // ...
    })
    
    expect(menus.find(m => m.path === '/ollama')).toBeUndefined()
  })
})
```

**Step 3: Run tests**

Run: `cd web && npm run test:unit menu-config.test.ts`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/components/Layout/menu-config.ts web/src/components/Layout/menu-config.test.ts
git commit -m "feat(menu): add Ollama management menu for standard+ editions"
```

---

## Task 8: 路由守卫更新

**Files:**
- Modify: `web/src/router/guards/edition-guard.ts`
- Modify: `web/src/router/guards/edition-guard.test.ts`

**Step 1: Add Ollama route guard**

```typescript
const VERSION_REQUIRED_ROUTES: Record<string, string[]> = {
  '/vector-db': ['enterprise'],
  '/knowledge': ['enterprise'],
  '/ollama': ['standard', 'enterprise']
}
```

**Step 2: Update test**

```typescript
it('should allow /ollama for standard edition', async () => {
  // ... setup standard edition
  
  const allowed = await canAccessEditionRoute('/ollama')
  expect(allowed).toBe(true)
})

it('should block /ollama for basic edition', async () => {
  // ... setup basic edition
  
  const allowed = await canAccessEditionRoute('/ollama')
  expect(allowed).toBe(false)
})
```

**Step 3: Run tests**

Run: `cd web && npm run test:unit edition-guard.test.ts`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/router/guards/
git commit -m "feat(guard): add Ollama route guard for standard+ editions"
```

---

## Task 9: 验证与文档

**Step 1: Run full test suite**

```bash
# 后端
make test
make lint
make build

# 前端
cd web && npm run typecheck
cd web && npm run build
cd web && npm run test:unit
```

**Step 2: Manual verification**

- [ ] 基础版：不显示 Ollama 菜单
- [ ] 标准版：显示 Ollama 菜单
- [ ] 企业版：显示 Ollama 菜单
- [ ] Ollama 页面功能正常
- [ ] Routing 页面不再显示 Ollama 区块
- [ ] 向量管理仅企业版可见

**Step 3: Update documentation**

- `docs/EDITION-GUIDE.md` - 更新版本功能对比表
- `README.md` - 更新功能列表

**Step 4: Commit**

```bash
git add docs/EDITION-GUIDE.md README.md
git commit -m "docs: update edition features documentation"
```

---

## 验证清单

### 功能验证

**基础版**：
- [ ] 不显示 Ollama 菜单
- [ ] 不显示向量管理入口
- [ ] 不显示知识库入口

**标准版**：
- [ ] 显示 Ollama 菜单
- [ ] 不显示向量管理入口
- [ ] 不显示知识库入口

**企业版**：
- [ ] 显示 Ollama 菜单
- [ ] 显示向量管理入口（Header）
- [ ] 显示知识库入口（Header）

### 技术验证

- [ ] 所有单元测试通过
- [ ] `make lint` 通过
- [ ] `make test` 通过
- [ ] `make build` 成功
- [ ] `npm run typecheck` 通过
- [ ] `npm run build` 成功
- [ ] `npm run test:unit` 通过

---

## 完成标准

仅当以下条件**全部满足**时，才能声明"已完成交付"：

1. ✅ 所有 Task 的测试通过
2. ✅ 后端 `make lint`、`make test`、`make build` 全部通过
3. ✅ 前端 `typecheck`、`build`、`test:unit` 全部通过
4. ✅ 基础版、标准版、企业版功能验证全部通过
5. ✅ 文档已更新
6. ✅ 工作区干净（`git status --short` 结果为空）

---

## 预估时间

- **Task 1-2**：1 小时（API + Store）
- **Task 3-5**：1.5 小时（页面 + 路由）
- **Task 6-8**：0.5 小时（菜单 + 守卫）
- **Task 9**：1 小时（验证 + 文档）
- **总计**：4 小时（约 0.5 个工作日）

---

## 执行回填（2026-03-03）

### 计划对照矩阵

- [x] Task 1：探索现有 Ollama 代码
- [x] Task 2：后端 API 封装（`web/src/api/ollama-domain.ts`）
- [x] Task 3：前端 Store（`web/src/store/domain/ollama.ts`）
- [x] Task 4：Ollama 管理页面（`web/src/views/ollama/index.vue`）
- [x] Task 5：从 Routing 页面移除 Ollama Tab（`web/src/views/routing/index.vue`）
- [x] Task 6：路由注册（`web/src/router/index.ts`）
- [x] Task 7：菜单配置（`web/src/components/Layout/menu-config.ts`）
- [x] Task 8：路由守卫更新（`web/src/router/guards/edition-guard.ts`）
- [x] Task 9：验证与文档更新（`docs/EDITION-GUIDE.md`、`README.md`）

### 回填验证清单

- [x] 基础版：不显示 Ollama 菜单
- [x] 标准版：显示 Ollama 菜单
- [x] 企业版：显示 Ollama 菜单
- [x] Routing 页面不再显示 Ollama Tab
- [x] 向量管理仅企业版可见（Header）
- [x] 所有前端单元测试通过
- [x] `make lint` 通过
- [x] `make test` 通过
- [x] `make build` 通过
- [x] `cd web && npm run typecheck` 通过
- [x] `cd web && npm run build` 通过
- [x] `cd web && npm run test:unit` 通过

### 回填命令记录

- `PATH="$(go env GOPATH)/bin:$PATH" make lint`：通过
- `make test`：通过
- `make build`：通过
- `go build ./cmd/gateway`：通过
- `cd web && npm run typecheck`：通过
- `cd web && npm run build`：通过
- `cd web && npm run test:unit`：通过（59 files / 170 tests）

**创建日期**：2026-03-02  
**负责人**：OpenCode  
**状态**：已执行完成

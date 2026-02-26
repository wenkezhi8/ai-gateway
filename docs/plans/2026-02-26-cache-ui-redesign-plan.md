# Cache UI Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 重设 `/cache` 页面为冷静仪表盘风，强化缓存类型说明 + 指标 + 操作的阅读路径，并保持现有功能不变。

**Architecture:** 使用 `cache-type-meta` 作为类型真相源，初始化缓存类型卡片数据；在 Vue 组件内重排模板为“概览 → 类型 → 语义签名 → 细节面板”三段式结构，同时统一样式与色板。

**Tech Stack:** Vue 3 + Element Plus + SCSS + Vitest

---

### Task 1: 通过 TDD 添加缓存类型卡片初始化工具

**Files:**
- Modify: `/Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web/src/utils/cache-type-meta.ts`
- Modify: `/Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web/src/utils/cache-type-meta.test.ts`

**Step 1: Write the failing test**

```ts
import { buildCacheTypeCards } from './cache-type-meta'

describe('buildCacheTypeCards', () => {
  it('merges stats and preserves order', () => {
    const cards = buildCacheTypeCards([
      { id: 'response', hitRate: 75, entries: 12, size: '2 MB', enabled: false }
    ])

    expect(cards[0].id).toBe('response')
    expect(cards[0].hitRate).toBe(75)
    expect(cards[0].entries).toBe(12)
    expect(cards[0].enabled).toBe(false)
    expect(cards[1].id).toBe('request')
    expect(cards[1].hitRate).toBe(0)
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web && npm run test:unit -- cache-type-meta`

Expected: FAIL with "buildCacheTypeCards is not defined"

**Step 3: Write minimal implementation**

```ts
export interface CacheTypeState {
  id: CacheTypeId
  enabled?: boolean
  hitRate?: number
  entries?: number
  size?: string
}

export interface CacheTypeCard extends CacheTypeMeta {
  enabled: boolean
  hitRate: number
  entries: number
  size: string
}

export const buildCacheTypeCards = (states: CacheTypeState[] = []): CacheTypeCard[] => {
  const byId = new Map(states.map(state => [state.id, state]))
  return CACHE_TYPE_ORDER.map(id => {
    const meta = CACHE_TYPE_META[id]
    const state = byId.get(id)
    return {
      ...meta,
      enabled: state?.enabled ?? true,
      hitRate: state?.hitRate ?? 0,
      entries: state?.entries ?? 0,
      size: state?.size ?? '0 MB'
    }
  })
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web && npm run test:unit -- cache-type-meta`

Expected: PASS

**Step 5: Commit**

```bash
git add /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web/src/utils/cache-type-meta.ts \
        /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web/src/utils/cache-type-meta.test.ts
git commit -m "test(cache-ui): add cache type card builder"
```

---

### Task 2: 重排缓存页模板结构为三段式布局

**Files:**
- Modify: `/Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web/src/views/cache/index.vue`

**Step 1: Write the failing test**

```ts
// UI 结构调整不新增单测；保持 Task 1 中的单元测试覆盖
```

**Step 2: Run test to verify it fails**

Skip (no new tests for template structure).

**Step 3: Write minimal implementation**

- 引入 `buildCacheTypeCards`，用其初始化 `cacheTypes`。
- 结构顺序调整为：Hero + Stats → Cache Type Grid → Semantic Signatures → Detail Panel。
- 缓存类型卡片展示：类型名 + 英文别名 + 说明 + Key 前缀 + 命中率/条目数。
- 语义签名区域独立成段，保留刷新按钮与表格。

**Step 4: Run test to verify it passes**

Run: `cd /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web && npm run test:unit -- cache-type-meta`

Expected: PASS

**Step 5: Commit**

```bash
git add /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web/src/views/cache/index.vue
git commit -m "feat(cache-ui): restructure cache page layout"
```

---

### Task 3: 统一样式与色板，完成冷静仪表盘风

**Files:**
- Modify: `/Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web/src/views/cache/index.vue`

**Step 1: Write the failing test**

```ts
// 视觉样式调整不新增单测
```

**Step 2: Run test to verify it fails**

Skip (no new tests for styling).

**Step 3: Write minimal implementation**

- 统一 CSS 变量（背景、卡片、强调色、状态色）。
- 类型卡片加入 tone 颜色标识与渐变背景。
- 添加轻微入场动效（opacity + translateY）。
- 移动端网格改为单列，保证操作可触达。

**Step 4: Run test to verify it passes**

Run: `cd /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web && npm run test:unit -- cache-type-meta`

Expected: PASS

**Step 5: Commit**

```bash
git add /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web/src/views/cache/index.vue
git commit -m "style(cache-ui): redesign cache visuals"
```

---

### Task 4: 全量验证

**Files:**
- None

**Step 1: Run unit tests**

Run: `cd /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web && npm run test:unit`

Expected: PASS

**Step 2: Run typecheck**

Run: `cd /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web && npm run typecheck`

Expected: PASS

**Step 3: Run build**

Run: `cd /Users/openclaw/ai-gateway/.worktrees/cache-ui-redesign/web && npm run build`

Expected: PASS

**Step 4: Commit verification note**

```bash
git commit --allow-empty -m "chore(cache-ui): verify cache page"
```

# Calm Dashboard Theme Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a switchable “calm dashboard” theme (variant + mode), keep Apple theme, and redraw Dashboard/Cache/Routing pages with unified console styling.

**Architecture:** Theme state managed by `useTheme` (variant + mode), persisted to localStorage, applied via `data-theme`/`data-mode` attributes. UI uses CSS variables as tokens to style Element Plus and page layouts. Pages consume tokens to ensure consistent visual language.

**Tech Stack:** Vue 3 + Element Plus + SCSS + Vite + Vitest

---

### Task 1: Theme logic tests (TDD)

**Files:**
- Create: `/Users/openclaw/ai-gateway/web/src/utils/theme.test.ts`
- Modify: `/Users/openclaw/ai-gateway/web/src/composables/useTheme.ts`

**Step 1: Write the failing test**

```ts
import { describe, it, expect, beforeEach } from 'vitest'
import { initTheme, setTheme, setVariant } from '@/composables/useTheme'

// ... localStorage/document mocks ...

describe('theme', () => {
  it('loads theme from storage and applies data-theme/data-mode', () => {
    // set localStorage value
    // call initTheme()
    // expect documentElement dataset
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/openclaw/ai-gateway/web && npm run test:unit -- theme`
Expected: FAIL (theme logic not implemented)

**Step 3: Write minimal implementation**

```ts
export type ThemeVariant = 'apple' | 'dashboard'
export type ThemeMode = 'light' | 'dark' | 'auto'

export const initTheme = () => {
  // read storage, apply dataset
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/openclaw/ai-gateway/web && npm run test:unit -- theme`
Expected: PASS

**Step 5: Commit**

```bash
git add /Users/openclaw/ai-gateway/web/src/utils/theme.test.ts /Users/openclaw/ai-gateway/web/src/composables/useTheme.ts
git commit -m "test(theme): add theme state and toggle tests"
```

---

### Task 2: Header theme switcher

**Files:**
- Modify: `/Users/openclaw/ai-gateway/web/src/components/Layout/index.vue`

**Step 1: Write failing test (optional)**
Not required for UI toggle.

**Step 2: Implement dropdown menu**

```vue
<el-dropdown trigger="click" @command="handleThemeCommand">
  <el-tooltip :content="themeTooltip" placement="bottom">
    <button class="theme-btn">...</button>
  </el-tooltip>
  <template #dropdown>
    <el-dropdown-menu>
      <el-dropdown-item disabled>主题风格</el-dropdown-item>
      <el-dropdown-item command="variant:apple">Apple</el-dropdown-item>
      <el-dropdown-item command="variant:dashboard">仪表盘</el-dropdown-item>
      <el-dropdown-item divided disabled>模式</el-dropdown-item>
      <el-dropdown-item command="mode:light">亮色</el-dropdown-item>
      <el-dropdown-item command="mode:dark">暗色</el-dropdown-item>
      <el-dropdown-item command="mode:auto">跟随系统</el-dropdown-item>
      <el-dropdown-item divided command="mode:toggle">快速切换模式</el-dropdown-item>
    </el-dropdown-menu>
  </template>
</el-dropdown>
```

**Step 3: Commit**

```bash
git add /Users/openclaw/ai-gateway/web/src/components/Layout/index.vue
git commit -m "feat(theme): add header theme switcher"
```

---

### Task 3: Settings theme variant

**Files:**
- Modify: `/Users/openclaw/ai-gateway/web/src/views/settings/index.vue`

**Step 1: Add variant radio group**

```vue
<el-form-item label="主题风格">
  <el-radio-group v-model="settings.themeVariant" @change="handleThemeVariantChange">
    <el-radio-button value="apple">Apple</el-radio-button>
    <el-radio-button value="dashboard">仪表盘</el-radio-button>
  </el-radio-group>
</el-form-item>
```

**Step 2: Bind to useTheme**

```ts
const { setTheme, setVariant, currentTheme } = useTheme()
settings.theme = currentTheme.value.mode
settings.themeVariant = currentTheme.value.variant
```

**Step 3: Commit**

```bash
git add /Users/openclaw/ai-gateway/web/src/views/settings/index.vue
git commit -m "feat(theme): add theme settings"
```

---

### Task 4: Dashboard theme tokens + Element Plus overrides

**Files:**
- Modify: `/Users/openclaw/ai-gateway/web/src/styles/variables.scss`
- Modify: `/Users/openclaw/ai-gateway/web/src/styles/index.scss`
- Modify (if needed): `/Users/openclaw/ai-gateway/web/src/styles/apple.scss`

**Step 1: Add dashboard light/dark tokens**

```scss
[data-theme="dashboard"] {
  --bg-app: #0f1418;
  --bg-card: #151b20;
  --border-color: #26313a;
  --accent: #2ec4b6;
  // ...
}

[data-theme="dashboard"][data-mode="dark"] {
  // adjusted dark tokens
}
```

**Step 2: Element Plus overrides**

```scss
.el-card {
  background: var(--bg-card);
  border-color: var(--border-color);
}
```

**Step 3: Commit**

```bash
git add /Users/openclaw/ai-gateway/web/src/styles/variables.scss /Users/openclaw/ai-gateway/web/src/styles/index.scss
git commit -m "style(theme): add dashboard tokens and component overrides"
```

---

### Task 5: Redraw Dashboard/Cache/Routing

**Files:**
- Modify: `/Users/openclaw/ai-gateway/web/src/views/dashboard/index.vue`
- Modify: `/Users/openclaw/ai-gateway/web/src/views/cache/index.vue`
- Modify: `/Users/openclaw/ai-gateway/web/src/views/routing/index.vue`

**Step 1: Dashboard layout polish**
- Metrics cards: layered typography, consistent padding
- Charts: unified border/background

**Step 2: Cache page tokens**
- Remove hardcoded colors, use tokens

**Step 3: Routing page tokens**
- Strategy cards & tables unified

**Step 4: Commit**

```bash
git add /Users/openclaw/ai-gateway/web/src/views/dashboard/index.vue /Users/openclaw/ai-gateway/web/src/views/cache/index.vue /Users/openclaw/ai-gateway/web/src/views/routing/index.vue
git commit -m "style(theme): redraw dashboard cache routing"
```

---

### Task 6: Full verification

**Step 1: Run tests**

```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit
cd /Users/openclaw/ai-gateway/web && npm run typecheck
cd /Users/openclaw/ai-gateway/web && npm run build
```

**Step 2: Commit**

```bash
git add -A
git commit -m "chore(theme): verify dashboard theme"
```

---

## Execution Options
1. **Subagent-Driven (this session)** — dispatch fresh subagent per task, review between tasks.
2. **Parallel Session (separate)** — new session uses executing-plans with checkpoints.


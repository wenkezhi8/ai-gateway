# Ollama Console Mixed Layout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 Ollama 控制台改为“上方服务管理固定可见 + 下方 Tab（意图路由/向量管理）”的单页混合布局，并按职责拆分模型管理与模型切换/启动能力。

**Architecture:** 保留现有 `useOllamaConsoleCore` 作为单一数据源，重构 `index.vue` 只负责页面编排，`OllamaServiceTab.vue` 专注服务生命周期与模型资产管理，`IntentRoutingTab.vue` 和 `VectorManagementTab.vue` 各自补充“切换模型 + 启动模型”入口。通过现有 ctx 方法复用后端 API，避免新增接口。

**Tech Stack:** Vue 3 + TypeScript + Element Plus + Vitest

---

### Task 1: 页面骨架改造（上方面板固定 + 下方 Tab）

**Files:**
- Modify: `web/src/views/ollama/index.vue`
- Test: `web/src/views/ollama/index.test.ts`

**Step 1: Write the failing test**

在 `web/src/views/ollama/index.test.ts` 增加断言：

```ts
it('should render service panel and lower config tabs together', async () => {
  const wrapper = mount(OllamaConsolePage, { global: buildGlobal() })
  expect(wrapper.text()).toContain('Ollama 服务管理')
  expect(wrapper.text()).toContain('意图路由')
  expect(wrapper.text()).toContain('向量管理')
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/views/ollama/index.test.ts`
Expected: FAIL（当前结构仍依赖旧的整页 Tab 语义）

**Step 3: Write minimal implementation**

在 `web/src/views/ollama/index.vue` 将主内容区改为：

```vue
<OllamaServiceTab :ctx="ctx" />
<div class="panel lower-panel">
  <el-tabs v-model="activeTab" class="console-tabs">
    <el-tab-pane label="意图路由" name="intent">
      <IntentRoutingTab :ctx="ctx" />
    </el-tab-pane>
    <el-tab-pane label="向量管理" name="vector">
      <VectorManagementTab :ctx="ctx" />
    </el-tab-pane>
  </el-tabs>
</div>
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/views/ollama/index.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/ollama/index.vue web/src/views/ollama/index.test.ts
git commit -m "refactor(ollama): switch to top service panel with lower config tabs"
```

### Task 2: 服务管理面板职责收敛（只保留服务与模型资产管理）

**Files:**
- Modify: `web/src/views/ollama/components/OllamaServiceTab.vue`
- Test: `web/src/views/ollama/ollama-stop-button.test.ts`

**Step 1: Write the failing test**

为 `OllamaServiceTab` 增加断言：不再渲染“启用意图分类器”“启用向量 Pipeline”开关。

```ts
expect(wrapper.text()).not.toContain('启用意图分类器')
expect(wrapper.text()).not.toContain('启用向量 Pipeline')
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/views/ollama/ollama-stop-button.test.ts`
Expected: FAIL（当前组件仍包含两个开关）

**Step 3: Write minimal implementation**

删除 `OllamaServiceTab.vue` 中 `toggle-row` 对应 DOM 与 `onClassifierEnableChange` / `onVectorPipelineEnableChange` 两个函数。

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/views/ollama/ollama-stop-button.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/ollama/components/OllamaServiceTab.vue web/src/views/ollama/ollama-stop-button.test.ts
git commit -m "refactor(ollama): move intent and vector toggles out of service panel"
```

### Task 3: 意图路由增加“切换模型 + 启动模型”

**Files:**
- Modify: `web/src/views/ollama/components/IntentRoutingTab.vue`
- Test: `web/src/views/ollama/intent-routing-config.test.ts`

**Step 1: Write the failing test**

在 `intent-routing-config.test.ts` 新增断言：存在“切换模型”“启动模型”按钮，并且不存在“下载模型”“删除模型”入口。

```ts
expect(wrapper.text()).toContain('切换模型')
expect(wrapper.text()).toContain('启动模型')
expect(wrapper.text()).not.toContain('下载模型')
expect(wrapper.text()).not.toContain('删除模型')
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/views/ollama/intent-routing-config.test.ts`
Expected: FAIL

**Step 3: Write minimal implementation**

在 `IntentRoutingTab.vue` 顶部配置区新增“模型运行控制”卡片：

```vue
<el-form-item label="当前模型">
  <el-select v-model="ctx.classifierSwitchModel" ... />
</el-form-item>
<el-button :loading="ctx.classifierSwitching" @click="ctx.switchClassifierModel">切换模型</el-button>
<el-button type="success" :loading="ctx.ollamaStarting" @click="ctx.startOllama">启动模型</el-button>
```

说明：启动模型走现有 `startOllama`，不新增下载/删除能力。

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/views/ollama/intent-routing-config.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/ollama/components/IntentRoutingTab.vue web/src/views/ollama/intent-routing-config.test.ts
git commit -m "feat(ollama): add classifier model switch and start controls"
```

### Task 4: 向量管理增加“切换模型 + 启动模型”

**Files:**
- Modify: `web/src/views/ollama/components/VectorManagementTab.vue`
- Test: `web/src/views/ollama/vector-scope.test.ts`

**Step 1: Write the failing test**

在 `vector-scope.test.ts` 增加断言：存在“切换模型”“启动模型”，不存在“下载模型”“删除模型”。

```ts
expect(wrapper.text()).toContain('切换模型')
expect(wrapper.text()).toContain('启动模型')
expect(wrapper.text()).not.toContain('下载模型')
expect(wrapper.text()).not.toContain('删除模型')
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/views/ollama/vector-scope.test.ts`
Expected: FAIL

**Step 3: Write minimal implementation**

在 `VectorManagementTab.vue` 向量配置卡片顶部加入模型切换与启动区：

```vue
<el-form-item label="Embedding 模型切换">
  <el-select v-model="ctx.dualModelConfig.vector_ollama_embedding_model" ... />
</el-form-item>
<el-button :loading="ctx.dualModelSaving" @click="ctx.saveDualModelConfigData">切换模型</el-button>
<el-button type="success" :loading="ctx.ollamaStarting" @click="ctx.startOllama">启动模型</el-button>
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/views/ollama/vector-scope.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/views/ollama/components/VectorManagementTab.vue web/src/views/ollama/vector-scope.test.ts
git commit -m "feat(ollama): add vector model switch and start controls"
```

### Task 5: 回归验证与收尾

**Files:**
- Modify (if needed): `web/src/views/ollama/*.vue`
- Modify (if needed): `web/src/views/ollama/*.test.ts`

**Step 1: Run focused unit suite**

Run: `cd web && npm run test:unit -- src/views/ollama`
Expected: PASS

**Step 2: Run typecheck**

Run: `cd web && npm run typecheck`
Expected: PASS

**Step 3: Run build**

Run: `cd web && npm run build`
Expected: PASS

**Step 4: Final cleanup check**

Run: `git status --short`
Expected: only intended files changed

**Step 5: Commit**

```bash
git add web/src/views/ollama
git commit -m "refactor(ollama): align panel responsibilities and model operations"
```

### 执行约束

- 全程遵守 @superpowers/test-driven-development：每个行为先写失败测试再实现。
- 完成宣告前遵守 @superpowers/verification-before-completion：必须提供命令与结果证据。
- 保持 DRY/YAGNI：不新增后端接口，不新增重复入口。

# Ollama 控制台功能聚合（执行计划）

## 目标

将 Ollama 相关能力统一聚合到 `/ollama`，并把 `/routing` 收敛为纯路由策略页面。

## 新页面结构

### `/ollama`

1. Tab: `Ollama`
   - 服务安装/启动/停止
   - 模型下载/删除
   - 常用模型一键选择
   - 本地模型与运行模型状态
   - 自动轮询与轮询间隔

2. Tab: `意图路由`
   - 意图模型配置（启用/Shadow/地址/主模型/候选模型/超时/阈值）
   - 控制面开关（归一化读/缓存门禁/风险控制/工具门控/Model Fit/参数建议）
   - 分类器健康与统计
   - 任务类型模型映射（8 类：代码生成、日常对话、逻辑推理、数学计算、事实查询、创意写作、翻译、其他）
   - 级联路由策略展示

3. Tab: `向量管理`
   - 向量模型配置（Pipeline、回写、服务地址、Embedding 模型、维度、超时、端点模式）
   - 向量索引状态与重建
   - 冷热分层配置与统计
   - Qdrant 条件配置
   - Pipeline 健康检查与在线测试

### `/routing`

- 仅保留路由策略视图（默认模式、默认策略、默认模型、任务分布、反馈评估与触发优化）。

## 实施步骤

1. 新建 `useOllamaConsole`，承接 Ollama/意图路由/向量管理状态与动作。
2. 在 `web/src/views/ollama/components/` 下创建：
   - `TabStateView.vue`
   - `OllamaServiceTab.vue`
   - `IntentRoutingTab.vue`
   - `VectorManagementTab.vue`
3. 重写 `web/src/views/ollama/index.vue` 为三 Tab 控制台。
4. 精简 `web/src/views/routing/index.vue`，去除模型管理与向量管理 Tab。
5. 删除 `web/src/views/routing/components/` 中旧的 Ollama/双模型/向量组件。
6. 更新并迁移前端单测，覆盖新页面结构与关键文案。

## 验证命令

```bash
cd web && npm run typecheck
cd web && npm run build
cd web && npm run test:unit
```

## 完成标准

- `/ollama` 三个 Tab 功能可用，文案与结构与需求一致。
- `/routing` 不再承载 Ollama、双模型、向量管理 UI。
- 前端 typecheck/build/unit tests 全通过。

## 执行回填（已完成）

### 交付状态

- 状态：已完成
- 完成时间：2026-03-03
- 关联提交：`9a4f24c9`

### 已完成项对照

1. 新建 `useOllamaConsole`，承接 Ollama/意图路由/向量管理状态与动作。已完成。
2. 新建 `web/src/views/ollama/components/` 下四个组件：`TabStateView.vue`、`OllamaServiceTab.vue`、`IntentRoutingTab.vue`、`VectorManagementTab.vue`。已完成。
3. 重写 `web/src/views/ollama/index.vue` 为三 Tab 控制台（`Ollama`、`意图路由`、`向量管理`）。已完成。
4. 精简 `web/src/views/routing/index.vue`，去除模型管理与向量管理 Tab。已完成。
5. 删除 `web/src/views/routing/components/` 中旧的 Ollama/双模型/向量组件。已完成。
6. 更新并迁移前端单测，覆盖新页面结构与关键文案。已完成。

### 关键文件

- 新增：
  - `web/src/views/ollama/composables/useOllamaConsole.ts`
  - `web/src/views/ollama/components/TabStateView.vue`
  - `web/src/views/ollama/components/OllamaServiceTab.vue`
  - `web/src/views/ollama/components/IntentRoutingTab.vue`
  - `web/src/views/ollama/components/VectorManagementTab.vue`
  - `web/src/views/ollama/intent-routing-config.test.ts`
  - `web/src/views/ollama/ollama-stop-button.test.ts`
  - `web/src/views/ollama/ollama-running-model.test.ts`
  - `web/src/views/ollama/vector-scope.test.ts`
- 修改：
  - `web/src/views/ollama/index.vue`
  - `web/src/views/ollama/index.test.ts`
  - `web/src/views/routing/index.vue`
  - `web/src/views/routing/composables/useRoutingConsole.ts`
  - `web/src/api/routing-domain.ts`
  - `web/src/views/routing/routing-tabs-layout.test.ts`
- 删除：
  - `web/src/views/routing/components/OllamaTab.vue`
  - `web/src/views/routing/components/ModelManagementTab.vue`
  - `web/src/views/routing/components/VectorManagementTab.vue`
  - `web/src/views/routing/routing-dual-model-management.test.ts`
  - `web/src/views/routing/routing-vector-scope.test.ts`

### 验证记录

执行命令：

```bash
cd web && npm run typecheck
cd web && npm run build
cd web && npm run test:unit
```

结果：全部通过（`60` files / `172` tests）。

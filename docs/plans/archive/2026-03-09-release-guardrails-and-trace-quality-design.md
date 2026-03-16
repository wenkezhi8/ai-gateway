# 发布护栏与 Trace 质量提升（Design）

## 背景

- 当前发布收尾依赖人工执行多条命令，容易遗漏。
- PR 变更范围缺少自动校验，main 历史漂移时可能把无关文件带入 PR。
- 发布账号在修改 workflow 文件时可能在 push 阶段被远端拒绝，反馈过晚。
- `/trace` 页面已有较多源码字符串断言，但缺少真实 DOM 渲染回归。
- `answer_source` 协议值与中文展示文案分散在多个文件，存在漂移风险。

## 目标

1. 增加 PR 文件范围自动检查，确保只交付本次需求相关目录。
2. 增加 workflow 变更白名单检查（你选择的 B 方案），提前阻断不允许改动。
3. 为 `/trace` 增加真实 DOM 渲染测试，覆盖 `answer_source` 中文展示。
4. 集中定义 `answer_source` 协议值与前端文案，减少前后端漂移。
5. 将发布验收整合为统一脚本，并接入 CI。
6. 增加遗留分支定期报告（只报告、人工清理）。
7. 增加交付状态自动校验（tag / PR merged / main 对齐）。

## 非目标

- 不改动现有业务能力与路由行为。
- 不自动删除远端分支。
- 不改变后端 `answer_source` 对外协议值（仍保持英文 canonical 值）。

## 方案概览

### 1) PR 范围检查与 workflow 白名单

- 新增配置文件描述“label -> 允许路径”规则。
- 在 PR CI 中执行检查脚本：
  - 读取 PR labels。
  - 获取 diff 文件列表。
  - 校验是否命中允许路径。
- 单独增加 workflow 白名单规则：仅允许特定 workflow 文件改动；其余一律失败。

### 2) answer_source 集中定义

- 后端：在 `internal/constants` 增加 canonical 值常量，并让 `admin/trace_answer_source.go` 复用。
- 前端：在 `web/src/constants` 新增协议值与中文标签常量，`trace-domain` 与 `trace` 页面统一引用。

### 3) /trace 真实 DOM 渲染测试

- 引入 `@vue/test-utils + jsdom`。
- 新增组件级测试，mock `getTraces/getTraceDetail/clearTraces`，断言页面真实渲染结果。
- 同时保留必要静态结构断言（作为低成本结构守卫）。

### 4) 发布脚本与 CI 接入

- 新增统一发布验收脚本：聚合 git 门禁、后端测试、前端 typecheck/build、交付状态检查。
- 新增交付状态脚本：检查 `HEAD` 是否命中 tag、PR 是否 merged、`main` 是否与 `origin/main` 对齐。
- 在 CI 中新增手动触发发布验收 workflow，并在标签推送时自动运行。

### 5) 遗留分支定期报告

- 新增 schedule workflow 每周生成“远端未合并分支报告”，自动写入 issue（可复用同一 issue）。
- 不执行删除动作，维持人工清理闭环。

## 风险与回滚

- 风险：PR 规则过严导致正常 PR 被阻断。
  - 缓解：配置规则支持 `always_allowed` 与标签维度放行。
- 风险：新增前端测试依赖导致测试环境变化。
  - 缓解：仅对特定组件测试使用 jsdom，其他测试保持默认环境。
- 回滚：可单独回退 workflow/脚本/测试，不影响业务主路径。

## 验收标准

1. PR 中出现不在规则内的文件时 CI 明确失败并给出文件列表。
2. PR 修改非白名单 workflow 文件时 CI 明确失败。
3. `/trace` DOM 测试可断言中文 `AI回复来源` 标签真实渲染。
4. `answer_source` 在后端与前端均有集中定义且单点引用。
5. 一条命令可完成发布验收，并可在 CI 复用。
6. 每周自动生成遗留分支报告 issue（仅报告）。
7. 交付状态脚本可输出 tag / PR / main 对齐结果。

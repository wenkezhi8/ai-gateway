# 2026-03-13 Token Compression OpenClaw 设计

- PLAN_ID: `2026-03-13-token-compression-openclaw`
- 状态: Baseline
- 范围: AI Gateway 长会话压缩、RAG 可开关、单路失败回退

## 目标
在不明显偏离答案的前提下，降低长会话输入 token 消耗，并保证可观测、可回滚、可灰度。

## 核心原则
1. 历史压缩优先，RAG 默认关闭。
2. 先压缩后代理请求。
3. 压缩失败直接回退原始全量请求（不做双路二次请求）。
4. 缓存命中优先，命中时跳过压缩。

## 三层上下文模型
1. L0 不可压缩层：system/developer 约束、当前问题、关键格式禁令。
2. L1 近邻原文层：最近 N 轮会话原文。
3. L2 历史摘要层：结构化摘要（目标、约束、数字锚点、已尝试方案、未完成项）。

## RAG 开关策略
1. `rag_dependency_enable=false`：仅历史压缩。
2. `rag_dependency_enable=true`：history+rag 压缩。
3. RAG 检索失败：退回 history-only 压缩；仍失败则回退原始请求。

## P0 清单（首发必须）
1. TokenBudgetReducer 前置接入 ChatCompletions。
2. compression 配置域与持久化接入。
3. 本地摘要模型调用与摘要注入开关。
4. RAG 可开关与失败降级链路。
5. 压缩/原始缓存键隔离与 dedup 维度隔离。
6. usage/trace 新增压缩与回退指标。
7. 自动回滚阈值（回退率、错误率）与灰度门禁。

## P1 清单（次阶段增强）
1. 动态预算（按模型窗口+task_type 分位）
2. 净节省 token 指标（扣除压缩调用成本）
3. 按 task_type/request_type 细粒度灰度策略

## 验收口径
1. 高输入场景平均输入 token 下降 >=40%。
2. 错误率不高于基线。
3. 一致性偏离率不高于基线。

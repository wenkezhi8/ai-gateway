# Vector DB Projectbook Gap Closure Design

> 范围基于《`/Users/openclaw/Desktop/AI-Gateway-Vector-DB-项目书.md`》中当前仓库尚未闭环的工程能力，按“工程闭环优先”执行。

## 1. 目标边界

本轮目标不是覆盖项目书全部长期平台项（如完整 Prometheus/Grafana 基础设施、100 万向量压测报告），而是补齐当前代码仓可直接交付与验证的缺口，形成“可用、可管、可恢复”的完整工程闭环。

本轮必须完成：

1. Collection empty 能力（API + UI + 测试）
2. Text search 链路（text -> embedding -> vector search）
3. Import 闭环增强（JSON/CSV/PDF 路径、失败明细、重试与页面入口）
4. RBAC/审计可用闭环（权限粒度补齐、审计查询 API + UI）
5. 限流 Redis 化（替换进程内限流）
6. 备份策略闭环（自动计划、保留与清理、恢复链路）

## 2. 架构拆分

按串行五阶段推进：

1. 集合管理补齐：empty 语义和幂等行为先落地
2. 检索补齐：text 与 vector 路径统一收敛到同一检索执行器
3. 导入补齐：任务状态机贯通上传/导入/失败回溯/重试
4. 治理补齐：RBAC、审计查询、Redis 限流
5. 运维补齐：备份策略（自动、保留、清理、恢复）

每阶段要求：

- 先写失败测试，再写最小实现
- 阶段内完成最小门禁再进入下一阶段

## 3. 数据流与错误处理

### 3.1 Collection empty

数据流：handler -> service -> backend 清空 -> repo 统计归零 -> audit 记录。

错误映射：

- 集合不存在：404
- 参数错误：400
- 后端不可用：503

幂等：重复清空返回成功。

### 3.2 Text search

数据流：请求 text -> embedding -> 搜索执行器 -> 统一结果模型。

错误映射：

- text 无效：400
- embedding 失败：502
- 集合不存在：404
- 后端异常：503

兼容规则：同时提供 text 与 vector 时优先 vector。

### 3.3 Import

数据流：创建任务（pending）-> 执行（running）-> 完成（completed）/失败（failed）。

PDF 路径：pdf 解析 -> 分块 -> embedding -> upsert。

错误策略：业务失败通过任务状态体现并记录失败明细；接口自身保持可轮询。

### 3.4 治理与备份

- RBAC：中间件统一鉴权，未授权 403。
- 审计：关键写操作必须留痕，支持检索。
- 限流：Redis 令牌桶，超限 429 + Retry-After。
- 备份：自动任务失败不影响在线查询，但需可观测、可重试、可清理。

## 4. 测试与验收

后端：

- empty 成功/异常/幂等
- text search 成功/embedding 失败/优先级
- import 三路径 + 失败明细 + 重试
- rbac 权限边界
- audit 查询过滤
- redis 限流并发行为
- backup 策略与恢复状态机

前端：

- empty 交互
- text 搜索
- import 任务进度与重试
- 审计/权限/备份关键流程

全量门禁：

- `make lint`
- `make test`
- `make build`
- `go build ./cmd/gateway`
- `cd web && npm run typecheck && npm run build && npm run test:unit`

## 5. 交付更新

实现完成后，必须同步更新：

- `docs/API_REFERENCE.md`
- `docs/USER_GUIDE.md`
- `docs/plans/2026-03-02-vector-db-full-scope-delivery-matrix.md`

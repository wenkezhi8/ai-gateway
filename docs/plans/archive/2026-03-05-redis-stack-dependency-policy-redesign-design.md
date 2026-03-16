# Redis Stack 依赖策略重设计（Design）

## 背景

- 当前项目已经在后端实现了向量缓存场景下的 Redis Stack 能力校验（`FT._LIST` + `JSON.GET`），并在向量关闭时支持 Redis 失败回落内存。
- 但版本依赖门禁与脚本依赖门禁仍把基础版（`basic`）视为 Redis 强依赖，导致“基础版无 Redis 也能跑”的目标被阻断。
- 依赖决策分散在多个入口（后端定义、多个脚本），存在口径漂移风险。

## 目标

1. 基础版默认无 Redis 也可运行。
2. 当 `vector_cache.enabled=true` 时，无论版本都要求 Redis（且向量链路必须满足 Redis Stack 能力）。
3. 标准版/企业版保持现有 Redis 依赖能力，不删除 Redis 代码路径与配置。
4. 将依赖决策统一为“配置驱动”，多脚本共用同一口径。

## 非目标

- 不移除 Redis 相关代码、配置、部署模板。
- 不改变现有版本功能矩阵（菜单与能力边界维持现状）。

## 方案选型

采用“共享策略文件 + 双端解析（Go + Shell）”方案：

- 新增策略源文件（建议：`configs/edition-dependency-policy.json`）。
- Go 与 Shell 都从该策略源得出“目标依赖集合”，避免规则重复硬编码。
- 保持系统接口形态稳定，仅调整基础版依赖输出与门禁逻辑。

## 策略模型

- `base_by_edition`
  - `basic`: `[]`
  - `standard`: `["redis", "ollama"]`
  - `enterprise`: `["redis", "ollama", "qdrant"]`
- `conditional_rules`
  - 条件：`vector_cache.enabled=true`
  - 追加依赖：`["redis"]`

最终依赖 = `base_by_edition[edition] ∪ conditional_rules`。

## 关键行为定义

1. `basic + vector_cache.enabled=false`
   - 无 Redis 前置门禁。
2. `basic + vector_cache.enabled=true`
   - 仍要求 Redis；向量初始化阶段继续走 Redis Stack 能力校验。
3. `standard/enterprise`
   - 保持各自依赖门禁。
4. Redis 连接失败
   - 若向量关闭：允许内存回落。
   - 若向量开启：Fail-fast。

## 影响面

- 后端版本定义与切换门禁。
- 脚本依赖解析（`dev-restart` / `setup-edition-env` / `start-gateway` / `docker`）。
- 前端版本卡片依赖展示（由后端 definitions 自然驱动）。
- 文档口径更新。

## 风险与回滚

- 风险：策略文件解析失败时可能导致脚本或门禁行为异常。
- 兜底：Go 与 Shell 均提供内置 fallback（沿用当前标准版默认策略）。
- 回滚：回退相关文件至旧门禁逻辑；Redis Stack 能力校验代码保持不动。

## 验收标准

1. 切换到基础版时，不因 Redis 健康状态失败而被阻断。
2. 基础版在向量开启时，依然会要求 Redis（并在非 Stack 能力下失败）。
3. 标准版/企业版依赖门禁行为不退化。
4. 相关 Go 与脚本测试覆盖新增行为并通过。

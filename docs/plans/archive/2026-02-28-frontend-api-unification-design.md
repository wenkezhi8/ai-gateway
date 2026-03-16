# 前端 API 统一与状态层重构设计（方案 C）

**目标**
- 前端业务数据统一通过 API 获取，消除多源数据与兜底解包导致的错乱。
- 建立可扩展的状态层，统一处理加载、错误、空态、成功态与写后读一致性。
- 将业务配置持久化迁移到后端，`localStorage` 仅保留 UI 偏好项。

**范围**
- 前端：`web/src/views/**`、`web/src/store/**`、`web/src/api/**`、`web/src/composables/**`。
- 后端：新增/补齐设置类管理接口（供前端统一读取与保存业务配置）。

## 1. 架构分层设计

采用四层单向依赖：

1. `View 层`（`views/*.vue`）
- 只负责渲染与交互。
- 禁止直接 `request.get/post/put/delete`。
- 仅调用 `domain` 层暴露的状态与 action。

2. `Domain 状态层`（建议 `stores/domain/*` 或 `composables/domain/*`）
- 按业务域拆分：`routing`、`cache`、`ops`、`model-management`、`settings`、`auth`。
- 管理统一状态机：`idle/loading/success/empty/error`。
- 负责写操作后的刷新、回滚、提示文案与重试策略。

3. `API 门面层`（`api/*`）
- 每个业务域对应一个 API 门面文件。
- 只返回规范化后的业务对象，不向上泄漏原始 response。
- 统一管理 URL、参数、返回结构映射。

4. `传输与协议层`（`api/request.ts`）
- 统一认证、401 跳转、错误映射、响应 envelope 解包。
- 对上层暴露统一契约，禁止页面出现 `data?.data || data` 兼容分支。

约束：
- OpenAI/Anthropic 协议兼容接口在网关侧保持协议原样。
- 管理台前端只消费统一 envelope 的管理接口。

## 2. 统一数据流与迁移顺序

统一数据流：
1. 页面进入调用 domain `init()`。
2. domain 并发调用 API 门面获取数据。
3. API 门面通过 `request` 层完成统一解包与错误规范化。
4. domain 更新状态机并驱动 UI。
5. 写操作由 domain action 统一处理，成功后执行精准合并或重拉。

分批迁移（可回滚）：

1. 批次 1：协议底座统一
- 在 `request` 层增加统一解包与标准错误对象映射。
- 新增 API 辅助：`unwrapEnvelope`、`mapApiError`。

2. 批次 2：高风险页面优先
- 迁移 `routing`、`cache`、`ops` 页面到 domain + API 门面。
- 删除 `data?.data || data` 与 `res?.data || []`。

3. 批次 3：业务默认数据下沉
- 移除 `chat/models` 的业务默认模型与服务商硬编码兜底。
- 改为后端 API 引导加载，前端仅保留空态 UI。

4. 批次 4：业务配置去 `localStorage`
- `model-management`、`routing`、`settings` 的业务配置迁移到后端持久化。
- `localStorage` 仅保留主题、语言、面板展开等 UI 偏好。

5. 批次 5：全站收尾守卫
- 扫描并禁止 `views/store` 直接引用 `@/api/request`。
- 扫描并禁止 `data?.data || data` 等非统一解包写法。

## 3. 错误处理与一致性规则

统一错误处理：
1. `request` 层仅做传输级错误处理与标准化。
2. `api` 层仅输出两类结果：
- 成功：规范化业务对象。
- 失败：抛出 `ApiError { code, message, status, detail? }`。
3. `domain` 层统一处理重试、降级、提示文案与状态切换。

一致性铁律：
1. 页面禁止直接 `request.*`。
2. 页面禁止 `data?.data || data`/`data.data || data`。
3. 业务配置禁止 `localStorage` 持久化。
4. 同一业务域只允许一个状态源。
5. 写后必须刷新或精准合并，禁止长期手改本地数组维持一致性。

## 4. 测试策略与验收门槛

TDD 节奏：
1. Red：先写失败测试（守卫测试 + API 解包单测 + domain 状态机单测）。
2. Green：最小实现通过测试（先底座，再逐页迁移）。
3. Refactor：清理重复逻辑并保持全绿。

测试类型：
1. 单元测试
- API：解包与错误映射。
- Domain：四态流转、写后刷新。
- 页面：关键交互只触发 domain action。

2. 集成/E2E
- `routing`：加载、保存、重试、刷新一致性。
- `cache`：统计、规则、健康检查的一致展示。
- `ops`：导出与展示数据一致。
- 认证：401 后跳转与回退行为正确。

3. 守卫测试
- 禁止 `views/store` 直接引入 `@/api/request`。
- 禁止出现 `data?.data || data` / `data.data || data`。

验收门槛：
1. 前端业务数据全部经 API 门面 + domain 状态层。
2. 业务配置由后端持久化，跨端一致。
3. `cd web && npm run typecheck` 通过。
4. `cd web && npm run test:unit` 通过。
5. 回归测试覆盖本次关键路径。

## 5. 风险与回滚

主要风险：
1. 迁移期间新旧路径并存导致重复请求。
2. 后端设置接口补齐前，`settings` 域迁移会被阻塞。
3. 页面临时依赖旧结构时可能出现字段空值。

控制策略：
1. 采用“新增 domain -> 双轨验证 -> 删除旧逻辑”分批切换。
2. 每批独立提交，失败可按批次回滚。
3. 对关键页面启用守卫测试阻止回归。

回滚点：
1. 每个批次保持独立 commit，可单独 `git revert <hash>`。
2. 保留后端兼容字段过渡窗口，待前端全部迁移后再清理。

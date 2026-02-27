# 全站“冷静仪表盘风”主题 + 关键页面重绘 设计说明

日期：2026-02-27

## 背景与目标
在现有 Apple 主题基础上新增可切换的“冷静仪表盘风”主题，统一 Dashboard/Cache/Routing 三页视觉风格，强化控制台感、信息块化与可扫读性。保留 Apple 主题作为默认，支持系统深浅色跟随与本地持久化，不改后端接口。

## 范围
- 主题逻辑：`web/src/composables/useTheme.ts`
- 顶栏入口：`web/src/components/Layout/index.vue`
- 设置页：`web/src/views/settings/index.vue`
- 主题变量：`web/src/styles/variables.scss`
- 全局样式：`web/src/styles/index.scss`
- Apple 主题兼容：`web/src/styles/apple.scss`（必要时）
- 页面重绘：`web/src/views/dashboard/index.vue`、`web/src/views/cache/index.vue`、`web/src/views/routing/index.vue`
- 单元测试：`web/src/utils/theme.test.ts`

## 设计原则
- 冷静仪表盘风：冷灰底、蓝绿强调色、卡片化信息块、轻量动效、可扫读性强。
- 主题 token 化：用 CSS 变量统一颜色、边框、阴影、间距、动效。
- 控制面优先：强调可扫读性与信息层级，减少视觉噪音。

## 主题系统设计
- 主题风格：`apple | dashboard`
- 模式：`light | dark | auto`
- DOM 标记：
  - `document.documentElement.dataset.theme = "apple" | "dashboard"`
  - `document.documentElement.dataset.mode = "light" | "dark"`
- 本地持久化：`ai-gateway-theme` JSON `{ variant, mode }`
- `auto` 模式监听 `prefers-color-scheme`
- Apple dark 兼容：保留旧 `data-theme="dark"` 或新增兼容标记

## 视觉与组件规范
- 基础色：低饱和冷灰底，蓝绿强调，维持高可读性。
- 卡片：统一圆角、描边、阴影与内边距。
- 表格：行间距提升、表头弱对比强调、状态色克制。
- 表单：输入、选择器统一底色、描边、聚焦态。
- 动效：统一 `transition`，不使用强烈动效。

## 页面重绘方向
- Dashboard：指标卡分层（主指标大号字 + 次指标小号字），图表区统一边框/底色/留白。
- Cache：去除局部硬编码色，全部走主题 token；信息块分层清晰。
- Routing：策略卡 / 权重表 / 模型评分统一视觉，强调可扫读布局。

## 兼容性与风险
- 主题变量覆盖可能导致局部对比度下降：通过 token 调整与对比度校验降低风险。
- 旧主题兼容：保留 Apple 主题默认行为与旧 dark 标记兼容。

## 验收标准
- 顶栏切换入口可用，刷新后主题保持。
- Auto 模式随系统切换。
- Dashboard/Cache/Routing 视觉统一，无布局错乱与闪烁。


# 全站仪表盘主题设计（可切换）

日期：2026-02-27

## 背景
当前前端以 Apple 风格为主，页面间风格不完全统一。用户希望新增一套“冷静仪表盘风”主题，并可在全站切换；同时对关键页面进行视觉重绘，形成统一的控制台体验。

## 目标
- 新增“仪表盘主题（dashboard）”并支持与现有 Apple 风格切换
- 主题切换入口置于顶栏右上角
- 主题选择持久化（localStorage）
- 支持自动跟随系统深浅色偏好（prefers-color-scheme）
- 对关键页面（Dashboard / Cache / Routing）进行视觉重绘

## 约束
- 不改变后端接口与业务逻辑
- 以 CSS 变量为主题真相源（token 化）
- 页面改造以结构保持为前提，避免重构交互逻辑

## 主题架构
- 主题变量：在 `:root`（Apple）与 `[data-theme="dashboard"]`（仪表盘）下分别维护一套 CSS 变量
- 组件映射：通过全局 CSS 变量覆盖 Element Plus 组件的默认样式（按钮、表格、Tabs、Tag、Dialog、Card）
- 切换机制：切换 `document.documentElement.dataset.theme`，并写入 `localStorage`
- 自动跟随系统：监听 `prefers-color-scheme`，在未手动锁定时自动切换

## 主题 Token（仪表盘）
- 颜色：`--dash-primary`, `--dash-accent`, `--dash-bg`, `--dash-panel`, `--dash-border`, `--dash-text`
- 阴影：`--dash-shadow-sm/md/lg`
- 字体：标题/正文/数值（数值使用等宽体）
- 卡片：背景/边框/hover/内边距
- 表格：表头背景/行 hover/条纹色

## 页面级重绘范围
1. Dashboard（运营监控页）
   - 强化指标卡与趋势图信息层级
   - 重排告警/诊断区域，形成“监控台节奏”
2. Cache（缓存管理页）
   - 全页统一使用仪表盘 token
   - 强化类型卡、策略区、内容区视觉一致性
3. Routing（路由策略页）
   - 策略卡、权重表、模型评分区统一风格

## 交互与动效
- 轻量动效：淡入 + 轻微位移 + hover 上浮
- 不增加复杂动画，保持信息可读与性能稳定

## 持久化与系统偏好
- `localStorage`: `theme = apple | dashboard`
- `prefers-color-scheme`：未锁定主题时自动跟随系统

## 测试与验收
- 单测：主题切换工具函数（如 `useTheme`）
- 构建：`npm run build`
- 验收清单：
  - 主题切换入口可用
  - 刷新后主题保持
  - 系统偏好自动切换
  - Dashboard/Cache/Routing 三页风格统一

## 风险与回滚
- 风险：全局变量覆盖可能影响局部组件对比度
- 回滚：仅前端样式变更，可回退至 Apple 主题变量

## 影响范围
- `web/src/styles/variables.scss`
- `web/src/styles/index.scss`
- `web/src/components/`（主题切换入口组件）
- `web/src/views/dashboard/`, `web/src/views/cache/`, `web/src/views/routing/`
- `web/src/utils/`（主题切换逻辑）

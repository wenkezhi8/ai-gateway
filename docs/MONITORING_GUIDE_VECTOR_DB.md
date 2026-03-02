# 向量模块监控配置指南

## 指标入口
- 汇总指标：`GET /api/admin/vector-db/metrics/summary`
- 告警规则：`/api/admin/vector-db/alerts/rules`

## 告警配置
1. 在管理端监控页创建规则。
2. 配置 channels（`webhook` / `email` / `console`）。
3. 使用 `POST /api/admin/vector-db/alerts/rules/notify-test` 验证多渠道通知。

## 故障排查
- 指标为空：检查 collection 与 import job 元数据是否存在。
- 通知失败：检查请求体 channels 是否在允许集合内。

# 向量模块运维手册

## 日常巡检
- 检查指标汇总接口响应。
- 检查导入失败任务数。
- 检查告警规则启用数与通知通道可用性。

## 常用命令
- 后端验证：`make lint && go test ./... && make build`
- 前端验证：`cd web && npm run typecheck && npm run build && npm run test:unit`

## 故障处置
1. 限流异常：确认 Redis 可用性与回退路径。
2. 检索异常：确认 Qdrant 连接与 collection 状态。
3. 导入异常：查看导入错误日志并按任务重试。

# 向量模块备份恢复手册

## 备份
1. 创建备份：`POST /api/admin/vector-db/backups`
2. 策略执行：`POST /api/admin/vector-db/backups/policy/run`

## 恢复
1. 选择备份任务 ID。
2. 执行恢复：`POST /api/admin/vector-db/backups/:id/restore`
3. 校验 collection 数据量与检索可用性。

## 常见问题
- 任务失败：查看 `error_message` 并重试。
- 快照过多：使用策略保留数量自动清理。

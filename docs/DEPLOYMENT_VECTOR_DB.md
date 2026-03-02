# 向量模块部署文档

## 依赖
- Qdrant 服务
- SQLite 文件路径（默认同网关配置）
- Redis（可选，用于限流）

## 部署步骤
1. 配置 `QDRANT_URL` 与 `QDRANT_API_KEY`。
2. 配置 SQLite 数据文件路径并执行迁移。
3. 启动网关：`make build && ./bin/ai-gateway`。
4. 健康检查：`GET /health`。

## 回滚
1. 切回上一版本二进制。
2. 使用备份快照执行恢复（见 `docs/BACKUP_RESTORE_MANUAL_VECTOR_DB.md`）。

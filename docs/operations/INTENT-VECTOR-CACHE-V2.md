# Intent + Vector Cache V2 上线手册

## 1. 目标

- 在网关引入本地 `intent-engine`（单次推理输出意图+槽位+向量）。
- 使用 Redis Stack 统一承载：JSON 文档 + 向量检索索引。
- 实现双层缓存命中：`exact key` > `vector semantic` > `upstream provider`。

## 2. 依赖

- Redis Stack 7.2+（`redis/redis-stack-server`）
- intent-engine 服务（提供 `/v1/intent-embed` 与 `/health`）

## 3. 配置项

`configs/config.json` 新增：

- `intent_engine.enabled`
- `intent_engine.base_url`
- `intent_engine.timeout_ms`
- `intent_engine.language`
- `intent_engine.expected_dimension`
- `vector_cache.enabled`
- `vector_cache.index_name`
- `vector_cache.key_prefix`
- `vector_cache.dimension`
- `vector_cache.query_timeout_ms`
- `vector_cache.thresholds`
- `vector_cache.ttl_seconds`

可用环境变量覆盖：

- `INTENT_ENGINE_ENABLED`
- `INTENT_ENGINE_BASE_URL`
- `INTENT_ENGINE_TIMEOUT_MS`
- `INTENT_ENGINE_LANGUAGE`
- `INTENT_ENGINE_EXPECTED_DIMENSION`
- `VECTOR_CACHE_ENABLED`
- `VECTOR_CACHE_INDEX_NAME`
- `VECTOR_CACHE_KEY_PREFIX`
- `VECTOR_CACHE_DIMENSION`
- `VECTOR_CACHE_QUERY_TIMEOUT_MS`

## 4. 上线步骤（直切）

1. 部署 Redis Stack 与 intent-engine，并确认健康检查通过。
2. 发布网关（默认建议 `INTENT_ENGINE_ENABLED=false`、`VECTOR_CACHE_ENABLED=false`）。
3. 调用 `POST /api/admin/cache/vector/rebuild` 预建索引。
4. 打开 `INTENT_ENGINE_ENABLED=true`，观察 10 分钟错误率与延迟。
5. 打开 `VECTOR_CACHE_ENABLED=true`，观察 30 分钟命中率与误命中。
6. 打开控制台可视化（`/cache`、`/routing`）。

## 5. 运维接口

- `GET /api/admin/router/intent-engine/config`
- `PUT /api/admin/router/intent-engine/config`
- `GET /api/admin/router/intent-engine/health`
- `GET /api/admin/cache/vector/stats`
- `POST /api/admin/cache/vector/rebuild`

## 6. 快速回滚

1. 向量异常：`VECTOR_CACHE_ENABLED=false`
2. 意图引擎异常：`INTENT_ENGINE_ENABLED=false`
3. Redis Stack 异常：回退 `redis/redis-stack-server` 到原 Redis 并关闭 vector cache

> 关闭以上开关后，网关自动退回原有缓存与直连上游路径。


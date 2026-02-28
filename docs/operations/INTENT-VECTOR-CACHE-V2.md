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
- `vector_cache.cold_vector_enabled`
- `vector_cache.cold_vector_query_enabled`
- `vector_cache.cold_vector_backend`（`sqlite|qdrant`）
- `vector_cache.cold_vector_dual_write_enabled`
- `vector_cache.cold_vector_similarity_threshold`
- `vector_cache.cold_vector_top_k`
- `vector_cache.hot_memory_high_watermark_percent`
- `vector_cache.hot_memory_relief_percent`
- `vector_cache.hot_to_cold_batch_size`
- `vector_cache.hot_to_cold_interval_seconds`
- `vector_cache.cold_vector_sqlite_path`
- `vector_cache.cold_vector_qdrant_url`
- `vector_cache.cold_vector_qdrant_api_key`
- `vector_cache.cold_vector_qdrant_collection`
- `vector_cache.cold_vector_qdrant_timeout_ms`

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
- `VECTOR_COLD_ENABLED`
- `VECTOR_COLD_QUERY_ENABLED`
- `VECTOR_COLD_BACKEND`
- `VECTOR_COLD_DUAL_WRITE_ENABLED`
- `VECTOR_COLD_SIMILARITY_THRESHOLD`
- `VECTOR_COLD_TOP_K`
- `VECTOR_HOT_MEMORY_HIGH_WATERMARK_PERCENT`
- `VECTOR_HOT_MEMORY_RELIEF_PERCENT`
- `VECTOR_HOT_TO_COLD_BATCH_SIZE`
- `VECTOR_HOT_TO_COLD_INTERVAL_SECONDS`
- `VECTOR_COLD_SQLITE_PATH`
- `VECTOR_COLD_QDRANT_URL`
- `VECTOR_COLD_QDRANT_API_KEY`
- `VECTOR_COLD_QDRANT_COLLECTION`
- `VECTOR_COLD_QDRANT_TIMEOUT_MS`

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
- `GET /api/admin/cache/vector/tier/stats`
- `POST /api/admin/cache/vector/tier/migrate`
- `POST /api/admin/cache/vector/tier/promote`

## 6. 冷热分层运行模式

- `cold_vector_enabled=false`：完全关闭冷层，仅热层 Redis 参与查询。
- `cold_vector_enabled=true && cold_vector_query_enabled=false`：冷层仅归档（迁移/写入），在线查询不访问冷层。
- `cold_vector_enabled=true && cold_vector_query_enabled=true`：热层 miss 后访问冷层，冷层命中后异步回暖到热层。

默认策略：

- `cold_vector_backend=sqlite`
- `cold_vector_query_enabled=true`
- `hot_memory_high_watermark_percent=75`
- `hot_memory_relief_percent=65`
- `hot_to_cold_batch_size=500`
- `hot_to_cold_interval_seconds=30`

## 7. 灰度与回滚建议

1. 初始上线保持 `cold_vector_enabled=false`。  
2. 先开归档（`cold_vector_enabled=true` 且 `cold_vector_query_enabled=false`），观察迁移与写入稳定性。  
3. 再开冷层在线查询（`cold_vector_query_enabled=true`），观察 P95 延迟与命中准确率。  
4. 如需切换冷后端，先启用 `cold_vector_dual_write_enabled=true` 同步一段时间，再切 `cold_vector_backend`。  
5. 故障时一键关闭 `cold_vector_enabled`，系统自动回退为纯热层 + 上游模型路径。

## 8. 快速回滚

1. 向量异常：`VECTOR_CACHE_ENABLED=false`
2. 意图引擎异常：`INTENT_ENGINE_ENABLED=false`
3. Redis Stack 异常：回退 `redis/redis-stack-server` 到原 Redis 并关闭 vector cache
4. 冷层异常：`VECTOR_COLD_ENABLED=false`

> 关闭以上开关后，网关自动退回原有缓存与直连上游路径。

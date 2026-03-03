# Redis Stack 全量统一设计

## 背景

- 项目在不同入口对 Redis 的要求不一致：部分脚本/文档/部署模板仍使用 plain Redis 口径。
- 向量缓存依赖 RediSearch 与 RedisJSON；当运行在 plain Redis 时会出现“前端显示启用但重建索引失败”的冲突。

## 目标

1. 项目运行、脚本、部署模板、文档统一为 Redis Stack。
2. 当 `vector_cache.enabled=true` 时，启动阶段必须校验 Redis Stack 能力，不满足时直接失败（fail-fast）。
3. 避免静默降级导致的技术冲突。

## 设计

### 1. 后端启动门禁

- 在缓存初始化阶段增加 Redis Stack 能力探测：
  - 校验 `FT._LIST`（RediSearch）
  - 校验 `JSON.GET`（RedisJSON）
- 若能力缺失且向量缓存启用，启动直接返回错误，主程序退出。

### 2. 本地开发脚本统一

- `scripts/dev-restart.sh` 统一按 Redis Stack 启动/校验：
  - 端口不在时自动拉起 `redis/redis-stack-server`
  - 启动后强制执行 `redis-cli FT._LIST` 验证
  - 验证失败立即退出并给修复命令

### 3. 部署模板统一

- 所有 compose 模板统一使用 `redis/redis-stack-server:7.2.0-v18`
- 启动命令统一为 `redis-stack-server`（避免 `redis-server` 造成模块未加载）

### 4. 文档统一

- 项目文档统一强调：向量能力依赖 Redis Stack，plain Redis 不支持。

## 风险与回滚

- 风险：历史环境未升级 Redis Stack 时，升级后会在启动阶段被阻断。
- 回滚：临时可关闭 `VECTOR_CACHE_ENABLED=false` 应急；长期仍需升级 Redis Stack。

# Redis Stack Unification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将项目 Redis 运行口径统一为 Redis Stack，并在向量缓存启用时强制能力校验，消除技术冲突。

**Architecture:** 后端在 `InitCacheManager` 阶段引入 Redis Stack 能力探测，向量启用时不满足能力则 fail-fast。脚本与部署模板统一为 `redis-stack-server` 启动方式，文档同步更新为 Redis Stack-only。

**Tech Stack:** Go、Vue、Shell、Docker Compose、Redis Stack

---

### Task 1: 启动门禁测试与实现

**Files:**
- Modify: `internal/bootstrap/gateway_test.go`
- Create: `internal/cache/redis_stack_capability.go`
- Create: `internal/cache/redis_stack_capability_test.go`
- Modify: `internal/bootstrap/gateway.go`
- Modify: `cmd/gateway/main.go`

**Step 1: Write the failing test**
- 为 `InitCacheManager` 添加 fail-fast 预期测试（vector init 失败时返回 error）。

**Step 2: Run test to verify it fails**
Run: `go test ./internal/bootstrap -run TestInitCacheManager -v`
Expected: FAIL（签名与行为尚未支持）。

**Step 3: Write minimal implementation**
- 将 `InitCacheManager` 改为返回 `(*cache.Manager, error)`。
- 增加 Redis Stack 能力探测工具函数。
- 在 `main.go` 中处理初始化错误并终止启动。

**Step 4: Run test to verify it passes**
Run: `go test ./internal/bootstrap -run TestInitCacheManager -v && go test ./internal/cache -run TestEnsureRedisStackCapabilitiesWithExecutor -v`
Expected: PASS

### Task 2: 启动脚本与部署模板统一

**Files:**
- Modify: `scripts/dev-restart.sh`
- Modify: `scripts/start-gateway.sh`
- Modify: `docker-compose.yml`
- Modify: `deploy/docker-compose.prod.yml`
- Modify: `deploy/docker/docker-compose.yml`

**Step 1: Implement**
- `dev-restart.sh` 去除 plain Redis 启动路径，改为 Redis Stack 校验/自启动。
- compose 模板命令统一为 `redis-stack-server`。

**Step 2: Verify**
Run: `./scripts/dev-restart.sh`
Expected: Redis Stack 就绪校验通过，网关正常启动。

### Task 3: 文档统一口径

**Files:**
- Modify: `PROJECT.md`
- Modify: `docs/FAQ.md`
- Modify: `ENV-CONFIGURATION.md`

**Step 1: Implement**
- 替换 plain Redis 指南，统一改为 Redis Stack 安装/验证命令。

**Step 2: Verify**
- 人工检查文档关键章节无 `redis-server` 与 `redis:7-alpine` 遗留。

### Task 4: 全量验证

**Step 1: Backend**
Run: `go test ./...`

**Step 2: Build**
Run: `go build ./cmd/gateway`

**Step 3: Frontend**
Run: `cd web && npm run typecheck && npm run build`

**Step 4: Runtime**
Run: `redis-cli -p 6379 FT._LIST`
Expected: 命令可用且可返回索引列表。

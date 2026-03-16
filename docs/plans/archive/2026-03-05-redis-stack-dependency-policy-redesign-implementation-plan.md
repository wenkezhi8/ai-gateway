# Redis Stack 依赖策略重设计 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 去除基础版对 Redis 的强依赖，并将版本依赖判定统一为配置驱动，在向量缓存开启时仍强制 Redis。

**Architecture:** 新增统一依赖策略源，后端与脚本从同一策略计算 required dependencies。基础版默认无 Redis，向量开关作为条件规则追加 Redis。Redis Stack 能力校验仍保留在向量初始化路径。

**Tech Stack:** Go, Shell, Vue3/Pinia, Vitest, Docker Compose, Redis Stack

---

### Task 1: 策略源与后端版本定义对齐

**Files:**
- Create: `configs/edition-dependency-policy.json`
- Modify: `internal/config/edition.go`
- Test: `internal/config/edition_test.go`

**Step 1: Write the failing test**
- 在 `internal/config/edition_test.go` 新增基础版依赖应为空的测试。

**Step 2: Run test to verify it fails**
Run: `go test ./internal/config -run TestEditionDefinitions_BasicDependencies -v`
Expected: FAIL（当前基础版依赖为 redis）。

**Step 3: Write minimal implementation**
- 修改 `internal/config/edition.go`，将基础版 `Dependencies` 设为 `[]string{}`。
- 新增策略文件初版（用于后续脚本与门禁统一读取）。

**Step 4: Run test to verify it passes**
Run: `go test ./internal/config -run TestEditionDefinitions_BasicDependencies -v`
Expected: PASS

### Task 2: 后端切版门禁行为回归（basic 不被 redis 卡死）

**Files:**
- Modify: `internal/handler/admin/edition_test.go`
- Modify: `internal/handler/admin/edition.go`（仅在必要时）

**Step 1: Write the failing test**
- 新增用例：当依赖状态里 `redis=false` 时，`PUT /api/admin/edition` 切到 `basic` 返回 200。

**Step 2: Run test to verify it fails**
Run: `go test ./internal/handler/admin -run TestEditionAPI_UpdateEdition_ToBasicWithoutRedis_ShouldSucceed -v`
Expected: FAIL（当前 basic 依赖 redis）。

**Step 3: Write minimal implementation**
- 若 Task 1 已改基础版 dependencies，通常无需额外实现；仅补齐必要返回字段断言。

**Step 4: Run test to verify it passes**
Run: `go test ./internal/handler/admin -run TestEditionAPI_UpdateEdition_ToBasicWithoutRedis_ShouldSucceed -v`
Expected: PASS

### Task 3: 脚本依赖决策统一为“版本 + 向量开关”

**Files:**
- Modify: `scripts/lib/edition-deps-policy.sh`
- Modify: `scripts/lib/edition-runtime.sh`
- Modify: `scripts/lib/dependency-manager.sh`
- Modify: `scripts/dev-restart.sh`

**Step 1: Write the failing tests**
- 在 `scripts/setup_edition_env_test.go` 增加两个场景（复用现有执行器）：
  - `basic + vector_cache.enabled=false` 无 redis 可通过。
  - `basic + vector_cache.enabled=true` 无 redis 应失败。

**Step 2: Run tests to verify they fail**
Run: `go test ./scripts -run "TestSetupEditionEnv_Basic.*" -v`
Expected: FAIL（当前 basic 强制 redis）。

**Step 3: Write minimal implementation**
- `edition-deps-policy.sh` 增加按配置读取 `vector_cache.enabled` 的条件依赖函数。
- `dependency-manager.sh` 的 required 依赖从新函数读取。
- `dev-restart.sh` 继续走统一 manager，无需二次硬编码判断。

**Step 4: Run tests to verify they pass**
Run: `go test ./scripts -run "TestSetupEditionEnv_Basic.*" -v`
Expected: PASS

### Task 4: setup-edition-env 与统一策略对齐

**Files:**
- Modify: `scripts/setup-edition-env.sh`
- Test: `scripts/setup_edition_env_test.go`

**Step 1: Write the failing test**
- 新增用例：`--edition basic --runtime native --apply-config false` 且 config 中 vector 关闭时，不再校验 redis。

**Step 2: Run test to verify it fails**
Run: `go test ./scripts -run TestSetupEditionEnv_BasicWithoutVector_ShouldSkipRedisRequirement -v`
Expected: FAIL

**Step 3: Write minimal implementation**
- setup 脚本根据“目标版本 + vector 开关”生成 required 列表，不再固定包含 redis。

**Step 4: Run test to verify it passes**
Run: `go test ./scripts -run TestSetupEditionEnv_BasicWithoutVector_ShouldSkipRedisRequirement -v`
Expected: PASS

### Task 5: 前端与文档口径同步

**Files:**
- Modify: `web/src/components/Layout/menu-config.test.ts`
- Modify: `docs/EDITION-GUIDE.md`
- Modify: `ENV-CONFIGURATION.md`
- Modify: `PROJECT.md`

**Step 1: Write failing/updated assertions**
- 将前端测试中的 basic 示例依赖从 `['redis']` 调整为 `[]`（仅示例数据，不影响菜单逻辑）。

**Step 2: Run tests to verify expected behavior**
Run: `cd web && npm run test:unit -- menu-config.test.ts`
Expected: PASS（或先 FAIL 后修复）。

**Step 3: Update docs**
- 统一文档口径：基础版默认可无 Redis；向量缓存启用时必须 Redis Stack。

### Task 6: 全量验证

**Step 1: Backend tests**
Run: `go test ./internal/config ./internal/handler/admin ./scripts`
Expected: PASS

**Step 2: Backend build**
Run: `go build ./cmd/gateway`
Expected: PASS

**Step 3: Frontend verification**
Run: `cd web && npm run typecheck && npm run build`
Expected: PASS

**Step 4: Runtime smoke (optional in local env)**
Run: `./scripts/dev-restart.sh`
Expected: basic + vector off 时无 redis 仍可启动；vector on 时给出明确 redis 依赖提示。

# 发布护栏与 Trace 质量提升 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 增加发布与 PR 护栏自动化，补齐 `/trace` 真实渲染测试，并集中 `answer_source` 协议定义，降低交付漂移与回归风险。

**Architecture:** 通过“配置驱动的 PR 变更规则 + CI 检查脚本 + 发布验收脚本”构建发布护栏；通过“前后端常量集中 + DOM 渲染测试”保证 trace 展示稳定。业务逻辑保持不变，主要增强校验与可维护性。

**Tech Stack:** Bash, Python3, GitHub Actions, Go, Vue3, Vitest, @vue/test-utils, jsdom

---

### Task 1: PR 文件范围检查（仅交付本次需求文件）

**Files:**
- Create: `.github/pr-scope-rules.json`
- Create: `scripts/check_pr_scope.py`
- Modify: `.github/workflows/ci.yml`
- Test: `scripts`（通过脚本 dry-run）

**Step 1: Write the failing test**
- 用临时文件列表模拟一个“frontend label 但包含 `internal/` 文件”的输入场景，预期脚本失败。

**Step 2: Run test to verify it fails**
Run: `python3 scripts/check_pr_scope.py --labels frontend --changed-files-file /tmp/changed.txt --rules .github/pr-scope-rules.json`
Expected: FAIL，并输出违规文件。

**Step 3: Write minimal implementation**
- 增加规则文件（label->allowed_paths）。
- 实现脚本：读取规则、labels、变更文件并输出违规项。
- 在 CI pull_request 流程接入该脚本。

**Step 4: Run test to verify it passes**
Run: 同上命令（将变更文件改为 `web/**`）
Expected: PASS

### Task 2: workflow 白名单检查（B 方案）

**Files:**
- Create: `.github/workflow-change-allowlist.txt`
- Create: `scripts/check_workflow_changes.sh`
- Modify: `.github/workflows/ci.yml`

**Step 1: Write the failing test**
- 构造变更文件包含 `.github/workflows/unknown.yml`，预期脚本失败。

**Step 2: Run test to verify it fails**
Run: `bash scripts/check_workflow_changes.sh --base-ref main --allowlist .github/workflow-change-allowlist.txt`
Expected: FAIL（当 diff 命中非白名单 workflow）。

**Step 3: Write minimal implementation**
- 实现 workflow 变更检查脚本。
- 在 CI pull_request 中接入，提前失败而非 push 时远端拒绝。

**Step 4: Run test to verify it passes**
Run: 同上命令（仅白名单文件变更）
Expected: PASS

### Task 3: 集中定义 answer_source（后端）

**Files:**
- Create: `internal/constants/trace_answer_source.go`
- Modify: `internal/handler/admin/trace_answer_source.go`
- Test: `internal/handler/admin/trace_test.go`, `internal/handler/admin/cache_request_traces_test.go`

**Step 1: Write the failing test**
- 新增/调整断言，确保 canonical 值来自常量定义且结果不变。

**Step 2: Run test to verify it fails**
Run: `go test ./internal/handler/admin -run Trace -v`
Expected: FAIL（在替换前）。

**Step 3: Write minimal implementation**
- 将 `exact_raw/exact_prompt/semantic/v2/provider_chat` 统一到 `internal/constants`。
- `trace_answer_source.go` 仅保留归一化逻辑，返回值引用常量。

**Step 4: Run test to verify it passes**
Run: `go test ./internal/handler/admin -run "Trace|CacheRequest" -v`
Expected: PASS

### Task 4: 集中定义 answer_source（前端）

**Files:**
- Create: `web/src/constants/trace-answer-source.ts`
- Modify: `web/src/api/trace-domain.ts`
- Modify: `web/src/api/cache-domain.ts`
- Modify: `web/src/views/trace/index.vue`

**Step 1: Write the failing test**
- 调整 trace 测试改为从常量导入后断言渲染标签。

**Step 2: Run test to verify it fails**
Run: `cd web && npm run test:unit -- src/views/trace/index.test.ts`
Expected: FAIL（在页面尚未改为常量前）。

**Step 3: Write minimal implementation**
- 集中导出协议值列表、类型与中文标签映射。
- 页面与 API 类型统一引用。

**Step 4: Run test to verify it passes**
Run: `cd web && npm run test:unit -- src/views/trace/index.test.ts src/api/trace-domain.test.ts`
Expected: PASS

### Task 5: `/trace` 真实 DOM 渲染测试

**Files:**
- Modify: `web/vitest.config.ts`
- Modify: `web/package.json`
- Modify: `web/package-lock.json`
- Modify: `web/src/views/trace/index.test.ts`
- Test: `web/src/views/trace/index.test.ts`

**Step 1: Write the failing test**
- 在 `index.test.ts` 增加组件 mount 测试：mock `getTraces` 返回 `answer_source=v2`，断言页面出现“向量缓存”。

**Step 2: Run test to verify it fails**
Run: `cd web && npm run test:unit -- src/views/trace/index.test.ts`
Expected: FAIL（依赖/环境未就绪或组件未可测试）。

**Step 3: Write minimal implementation**
- 配置 `@vitejs/plugin-vue` + jsdom 环境。
- 引入 `@vue/test-utils` 并使用轻量组件 stub 挂载页面。
- 保留必要快照，覆盖关键 DOM 区块。

**Step 4: Run test to verify it passes**
Run: `cd web && npm run test:unit -- src/views/trace/index.test.ts`
Expected: PASS

### Task 6: 发布验收脚本与交付状态检查脚本

**Files:**
- Create: `scripts/release-acceptance.sh`
- Create: `scripts/delivery-status.sh`
- Modify: `.github/workflows/ci.yml`（或新增 release workflow）
- Test: `scripts`（脚本 dry-run + 部分真实命令）

**Step 1: Write the failing test**
- 使用 dry-run 模式校验脚本必须输出 gate 项（git 三连、tag、PR、main 同步）。

**Step 2: Run test to verify it fails**
Run: `bash scripts/release-acceptance.sh --dry-run`
Expected: FAIL（实现前脚本不存在）。

**Step 3: Write minimal implementation**
- `release-acceptance.sh` 聚合：git 门禁、后端测试、前端 typecheck/build、`delivery-status.sh`。
- `delivery-status.sh` 校验：HEAD-tag、PR merged（可传 `--pr`）、main 对齐。
- 在 CI 增加手动触发/标签触发入口。

**Step 4: Run test to verify it passes**
Run: `bash scripts/release-acceptance.sh --dry-run --skip-frontend --skip-backend`
Expected: PASS

### Task 7: 遗留分支定期报告（只报告）

**Files:**
- Create: `.github/workflows/branch-hygiene-report.yml`
- Create: `scripts/branch-hygiene-report.sh`

**Step 1: Write the failing test**
- 先运行脚本（本地 repo 若无 remote 数据可允许输出空报告），确保输出 markdown。

**Step 2: Run test to verify it fails**
Run: `bash scripts/branch-hygiene-report.sh --output /tmp/branch-report.md`
Expected: FAIL（实现前脚本不存在）。

**Step 3: Write minimal implementation**
- 生成包含“分支名、最后提交时间、是否已合并 main”的 markdown 报告。
- workflow 按周调度并创建/更新 issue（标签 `branch-hygiene`）。

**Step 4: Run test to verify it passes**
Run: `bash scripts/branch-hygiene-report.sh --output /tmp/branch-report.md`
Expected: PASS

### Task 8: 全量验证

**Step 1: Backend tests**
Run: `go test ./...`
Expected: PASS

**Step 2: Frontend tests**
Run: `cd web && npm run test:unit -- src/views/trace/index.test.ts src/api/trace-domain.test.ts`
Expected: PASS

**Step 3: Frontend static checks**
Run: `cd web && npm run typecheck && npm run build`
Expected: PASS

**Step 4: Script smoke checks**
Run: `bash scripts/release-acceptance.sh --dry-run --skip-frontend --skip-backend`
Expected: PASS

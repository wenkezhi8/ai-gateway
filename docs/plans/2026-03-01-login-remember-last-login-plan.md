# 登录页记住上次登录实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 登录页不再预置默认账号密码，仅在勾选“记住我”后记住上次成功登录的账号密码。

**Architecture:** 抽离一个轻量凭据持久化工具模块，登录页加载时读取本地凭据并回填；登录成功后按“记住我”状态决定保存或清理。通过 Vitest 覆盖工具模块的读写行为。

**Tech Stack:** Vue 3, TypeScript, Pinia, Vitest

---

### Task 1: 凭据持久化工具（TDD）

**Files:**
- Create: `web/src/views/login/remember-credentials.ts`
- Create: `web/src/views/login/remember-credentials.test.ts`

**Step 1: 写失败测试（Red）**
- 覆盖场景：无数据返回空、非法 JSON 返回空、正常保存与读取、取消记住后清理。

**Step 2: 运行测试确认失败**
- Run: `cd web && npm run test:unit -- src/views/login/remember-credentials.test.ts --run`
- Expected: 至少 1 条失败（函数未实现）。

**Step 3: 最小实现（Green）**
- 实现 `loadRememberedCredentials` / `persistRememberedCredentials`。

**Step 4: 运行测试确认通过**
- Run: `cd web && npm run test:unit -- src/views/login/remember-credentials.test.ts --run`
- Expected: 全部通过。

### Task 2: 登录页接入“记住上次登录”

**Files:**
- Modify: `web/src/views/login/index.vue`

**Step 1: 页面初始化时加载已记住凭据**
- 若存在本地保存值，回填 `username/password` 并将 `remember=true`。

**Step 2: 登录成功后按 remember 状态持久化**
- remember=true：保存本次账号密码。
- remember=false：清理已保存账号密码。

**Step 3: 明确不再提供默认账号密码填充**
- 保持首次进入登录页为空。

### Task 3: 回归验证

**Files:**
- Verify only

**Step 1: 运行单测**
- `cd web && npm run test:unit -- src/views/login/remember-credentials.test.ts --run`

**Step 2: 运行类型检查**
- `cd web && npm run typecheck`

**Step 3: 运行构建**
- `cd web && npm run build`

**Step 4: 手工验证**
- 首次打开登录页为空。
- 勾选“记住我”登录后刷新可自动回填。
- 不勾选登录后刷新不回填。

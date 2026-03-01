# 开发重启脚本防呆加固 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 避免 `/trace` 页面看起来回退，确保每次重启都运行与当前代码一致的前后端构建产物。

**Architecture:** 将 `scripts/dev-restart.sh` 改为严格失败即退出流程：前端构建成功后再构建后端二进制，清理旧进程后仅启动 `bin/gateway`，并在启动后校验 8566 端口、`/health` 以及 `/trace` 引用资产可访问。

**Tech Stack:** Bash, npm, Go, curl, lsof

---

### Task 1: 脚本流程改造

**Files:**
- Modify: `scripts/dev-restart.sh`

**Step 1: 启用严格模式**
- 使用 `set -euo pipefail`，任一步失败立即退出。

**Step 2: 固定构建顺序**
- 先执行 `cd web && npm run build`。
- 再执行 `go build -o bin/gateway ./cmd/gateway`。

**Step 3: 清理旧进程并强校验端口**
- 停止 `go run` 与各类 gateway 进程。
- 确认 8566 无旧监听，再启动新二进制。

**Step 4: 启动后健康与资产校验**
- 校验 `/health` 返回 healthy。
- 校验 `/trace` 页面可访问并解析出 `/assets/*.js`。
- 对首个 JS 资源执行 HTTP 检查（不能返回 HTML）。

### Task 2: 运行验证

**Files:**
- Verify only

**Step 1: 执行重启脚本**
- Run: `./scripts/dev-restart.sh`

**Step 2: 验证端口和健康**
- Run: `lsof -i :8566 -n -P`
- Run: `curl -s http://localhost:8566/health`

**Step 3: 验证 trace 资源**
- Run: `curl -s http://localhost:8566/trace`
- 确认返回页面引用了 `/assets/index-*.js`。

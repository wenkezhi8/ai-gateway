# AI Gateway Release Checklist

## 目标
将 docs/swagger/cors/metrics 四类高风险点集中到单一清单，避免并行发布漏检。

## 预检
1. 确认当前分支符合 `codex/feature/*` 规则。
2. 执行提交证据三连：`git rev-parse --short HEAD`、`git show --name-only --pretty='' HEAD`、`git status --short`。
3. 确认配置源：`CONFIG_PATH`、`SERVER_PORT`、`CORS_ALLOW_ORIGINS`、`METRICS_HOST`。

## Runtime Smoke（核心）
1. 执行 `./scripts/release-smoke.sh --base-url <url> --metrics-url <url>`。
2. docs 语义：
   - `/docs`、`/docs/` 必须返回 SPA，不得携带 `Location` 跳转到 swagger。
3. swagger 语义：
   - `/swagger`、`/swagger/` 必须 302 到 `/swagger/index.html`。
   - `/swagger/doc.json` 必须为 JSON，且体积不超过 smoke 采样阈值。
4. metrics 语义：
   - 网关端口 `/metrics` 必须关闭。
   - 仅本机 `127.0.0.1/localhost/::1` 允许访问 metrics。
5. CORS 语义：
   - 当 `CORS_ALLOW_ORIGINS` 为白名单时，allowed/blocked + preflight 都需校验。
   - 默认要求 `Vary: Origin`，可通过 runtime smoke 参数显式关闭。

## Release Acceptance（总门禁）
1. 推荐执行：
   - `./scripts/release-acceptance.sh --runtime-smoke-cors-from-env --runtime-smoke-require-vary-origin`
2. 受限网络环境：
   - `./scripts/release-acceptance.sh --allow-limited-network-skip`
3. 临时端口或多实例：
   - `./scripts/release-acceptance.sh --spawn-gateway --spawn-gateway-port 18566 --spawn-gateway-config ./configs/config.json`

## 失败快速定位
1. `docs/swagger` 失败：检查 `internal/docs/route_strategy.go` 与 `internal/router/router_release_fallback_test.go`。
2. `cors` 失败：检查 `internal/middleware/cors.go` 与 `scripts/release-smoke.sh` 的 CORS 断言。
3. `metrics` 失败：检查 `METRICS_HOST` 配置及 `internal/bootstrap/gateway.go` 启动日志告警。
4. `runtime smoke` 连接失败：先确认本机网络策略，再看 `release-acceptance.sh` 连通性预检输出。

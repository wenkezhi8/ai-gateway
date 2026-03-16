# 2026-03-13 Token Compression OpenClaw 实施计划

- PLAN_ID: `2026-03-13-token-compression-openclaw`
- 执行模式: brainstorming + TDD（Red -> Green -> Refactor）

## 开工流程卡
- 目标: 逐步完成 P0，保持可回滚与可观测。
- 改动范围: `internal/handler`、`internal/routing`、`internal/storage`、`internal/handler/admin`。
- Git 权限: 默认不执行 commit/push/tag。
- 必跑验证:
  1. `go test ./internal/handler/...`
  2. `go test ./internal/cache/...`
  3. `go test ./internal/storage/...`
  4. `make test`
  5. `make build`
  6. `go build ./cmd/gateway`
  7. `cd /Users/openclaw/ai-gateway && ./scripts/dev-restart.sh`
  8. `make lint`（触达 lint 规则时）

## P0 执行拆解
1. 配置与门禁
- 新增 `compression.*` 配置项和默认值。
- 固化 `PLAN_ID` 基线校验，防止跑旧计划。

2. 压缩链路接入
- 在 ChatCompletions 上游调用前接入预算判断与压缩。
- 缓存命中请求跳过压缩。

3. RAG 可开关
- 开启时引入知识片段参与压缩。
- 检索失败自动降级到 history-only。

4. 失败回退
- 压缩失败/超时回退原始请求。
- 仅允许一次回退。

5. 指标与可观测
- usage 新增压缩、RAG、回退字段。
- trace 新增 `context.compress`、`context.compress.rag.retrieve`、`context.compress.fallback`。

## P0 测试清单
1. 超预算触发压缩。
2. 预算内不压缩。
3. RAG 开启并命中。
4. RAG 检索失败降级 history-only。
5. 压缩失败回退原始请求。
6. 缓存命中跳过压缩。
7. 压缩/原始缓存 key 隔离。
8. usage/trace 指标输出。

## P1 执行拆解
1. 动态预算算法。
2. 净节省 token 指标。
3. 细粒度灰度开关。

## 完成回报模板约束
每次自动化运行必须输出：
1. 根因
2. 方案
3. 改动文件
4. 验证结果
5. 风险与回滚
6. 版本建议
7. 计划对照矩阵（仅引用本 PLAN_ID 的 P0/P1）

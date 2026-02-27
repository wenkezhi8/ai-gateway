# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.6] - 2026-02-27

### Performance
- **数据存储异步写入**: 性能提升 94.4% (2.4ms → 0.14ms/op)
- **Smart Router 锁优化**: 短临界区策略, 性能提升 1.9%
- **Redis 连接池配置**: 新增 PoolSize/MinIdleConns/MaxRetries/Timeout 参数
- **ProviderPool 优雅关闭**: 添加 stopWg + stopOnce 实现幂等关闭
- **LRU 缓存淘汰**: MemoryCache 支持最大条目数限制
- **统一 Logger**: 清理 13+ 文件的独立 logrus.New()

### Added
- 新增测试用例提升覆盖率:
  - `internal/datastore`: 覆盖率提升至 84.5%
  - `internal/provider`: 新增 Stop 相关测试
  - `internal/cache`: 新增 LRU 淘汰测试
  - `pkg/logger`: 新增统一 Logger 测试

### Changed
- 数据存储从同步写入改为异步批量写入
- 所有独立 Logger 实例改为使用 pkg/logger
- MemoryCache 新增 LRU 淘汰机制
- SmartRouter 优化锁粒度减少锁竞争

## [Unreleased]


### Added
- Enterprise optimization: golangci-lint configuration
- Enterprise optimization: ESLint 9 + Prettier configuration for frontend
- Enterprise optimization: .editorconfig for unified editor settings
- Enterprise optimization: pre-commit hooks configuration
- Enterprise optimization: Enhanced Makefile with CI commands

## [1.6.5] - 2026-02-27

### Added
- Control-layer strategy support for `context_load` hints to influence auto strategy selection when enabled.
- Control-layer RAG gate behavior that can disable deep-think path when `rag_needed=false`.
- Operations runbook for control rollout and rollback: `docs/control-layer-operations.md`.

### Fixed
- Control TTL band now applies after rule-store TTL matching, ensuring control signal precedence is effective.

## [1.6.4] - 2026-02-27

### Changed
- Control shadow mode now records tool-gate and model-fit decisions without mutating request/selection behavior.

### Added
- Tests to verify shadow-only observability behavior for tool gate and model-fit routing.

## [1.6.3] - 2026-02-27

### Added
- Unit tests for classifier control-signal parsing, clamp behavior, and parse-error wrapping.
- Unit tests for semantic query candidate chain ordering and dedup behavior in proxy control flow.

## [1.6.2] - 2026-02-27

### Added
- Unit tests for control-layer cache/tool gates and model-fit routing selection.

### Changed
- Workflow card now explicitly enforces continuous execution to final deliverable output unless blocked.

## [1.6.1] - 2026-02-27

### Fixed
- Preserve classifier control sub-flags when `control.enable=false` to avoid resetting staged control toggles during config save.

## [1.6.0] - 2026-02-27

### Added
- Control-layer fields for classifier output: `normalized_query`, `cacheable`, `ttl_band`, `risk_level`, `risk_tags`, `tool_needed`, `rag_needed`, `context_load`, and `model_fit`.
- Routing UI controls for classifier control toggles and control metrics display on `/routing`.
- Playwright coverage for control toggles save flow and control stats rendering (`web/tests/scenarios/routing.spec.ts`).

### Changed
- Semantic cache lookup now supports candidate query chain with optional normalized query path.
- Stream and non-stream cache write paths now support control-gated behavior (`cacheable`) and control TTL band mapping.
- Smart router auto selection supports model-fit routing signal when control feature flags are enabled.

### Fixed
- Fixed control-layer observability gap by adding parse error and control field coverage stats.
- Fixed inconsistent workflow reporting by requiring version tag in Git status report guidelines.

## [1.5.1] - 2026-02-26

### Changed
- Bump version to 1.5.1 for release tag alignment.

## [1.5.0] - 2026-02-26

### Added
- Admin router endpoint `GET /api/admin/router/classifier/models` to return classifier candidate models for manual switching.
- Playwright scenario `web/tests/scenarios/routing.spec.ts` to cover classifier model list refresh and model switching flow.
- Dashboard alert management section with filters and acknowledgment actions.
- Cache entries now record task type source for UI display.

### Changed
- Routing page now supports manual "refresh model list" for classifier candidates and updates options from real-time Ollama installed models.
- Router config response now enriches `classifier.candidate_models` with Ollama `/api/tags` results while preserving active model and configured fallbacks.
- Dashboard layout refreshed with hero summary and alert quick actions.

### Fixed
- Fixed `/routing` manual classifier model switch dropdown only showing static default candidates.
- Improved fallback behavior for classifier model list retrieval when Ollama is unavailable.
- Cache hit metrics now record task type source and avoid unknown-only classification.
- Classifier fallback now routes `unknown` to heuristic detection for short greetings.

## [1.0.0] - 2024-01-01

### Added
- Multi-provider support: OpenAI, Anthropic, Zhipu, DeepSeek, Qwen, etc.
- Intelligent rate limiting with per-user and global quotas
- Response caching to reduce API costs
- Flexible routing strategies (cost-based, round-robin, failover)
- OpenAI-compatible RESTful API
- Web dashboard for monitoring and configuration
- Docker and Docker Compose support
- Prometheus + Grafana monitoring stack
- JWT authentication
- Audit logging
- Swagger API documentation

### Security
- Request body size limit (10MB) to prevent DoS attacks
- API key masking in logs
- CORS middleware

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.6.5 | 2026-02-27 | Add context-load and RAG gating controls, fix TTL precedence, and add control ops runbook |
| 1.6.4 | 2026-02-27 | Add shadow-only observability for control tool gate and model-fit routing |
| 1.6.3 | 2026-02-27 | Add parser and semantic-candidate coverage for control-layer safety |
| 1.6.2 | 2026-02-27 | Add control-layer unit tests and update workflow card for direct final-result delivery |
| 1.6.1 | 2026-02-27 | Fix control config clamp to preserve sub-toggle values when control master switch is off |
| 1.6.0 | 2026-02-27 | Add 0.5B control signals, cache/tool gates, router fit selection, and routing control UI |
| 1.5.1 | 2026-02-26 | Patch release to align version and tag |
| 1.5.0 | 2026-02-26 | Fix classifier model list source, add routing classifier e2e coverage |
| 1.0.0 | 2024-01-01 | Initial release |

---

## How to Update

```bash
# Pull latest changes
git pull origin main

# Update dependencies
make deps

# Rebuild
make build

# Restart service
./bin/ai-gateway
```

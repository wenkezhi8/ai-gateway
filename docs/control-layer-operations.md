# Control Layer Operations Guide

This guide describes how to operate the 0.5B control layer safely in production.

## Scope

- Controller model only (`qwen2.5:0.5b-instruct`)
- No direct user-answer generation in controller path
- Fail-open behavior remains mandatory

## Feature Flags

All flags are under `classifier.control` in `/api/admin/router/config`.

- `enable`: master switch for control features
- `shadow_only`: evaluate and log decisions without mutating routing/request flow
- `normalized_query_read_enable`: semantic cache candidate includes normalized query first
- `cache_write_gate_enable`: control signals can gate cache write and ttl band mapping
- `risk_tag_enable`: log risk-level and risk-tags for observation
- `tool_gate_enable`: apply `tool_needed`/`rag_needed` decisions
- `model_fit_enable`: allow model-fit score to influence auto model selection

## Rollout Order

1. Set `enable=true` and `shadow_only=true`
2. Enable one sub-flag at a time
3. Observe metrics and logs for at least one traffic window
4. Set `shadow_only=false` only when confidence is acceptable

Recommended order:

1. `normalized_query_read_enable`
2. `cache_write_gate_enable`
3. `risk_tag_enable`
4. `tool_gate_enable`
5. `model_fit_enable`

## Fast Rollback

- Disable a single sub-flag first
- If impact continues, set `enable=false`
- Keep `fail_open=true`

No restart is required for config changes.

## What to Monitor

- Classifier stats (`/api/admin/router/classifier/stats`):
  - `llm_success`
  - `fallbacks`
  - `avg_llm_latency_ms`
  - `avg_control_latency_ms`
  - `parse_errors`
  - `control_fields_missing`
- Cache hit ratio and cache write volume
- Tool invocation ratio (before/after `tool_gate_enable`)
- Request latency (P95/P99)

## Decision Semantics

- `shadow_only=true`
  - tool gate/model-fit/rag gate decisions are logged only
  - no request mutation, no routing mutation
- `cacheable=false`
  - write cache is skipped when `cache_write_gate_enable=true`
- `ttl_band`
  - `short=1h`, `medium=24h`, `long=7d`
  - applied after rule-store TTL matching

## Incident Checklist

1. Check if control layer recently changed flags
2. Check parse errors and control field missing counters
3. Confirm whether service is in shadow mode
4. Disable the most recently enabled flag
5. If needed, set `enable=false` and keep `fail_open=true`

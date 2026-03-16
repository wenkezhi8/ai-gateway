# Google Native Adapter Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace Google OpenAI-compatible routing with a native Google Gemini provider adapter (including streaming) and align frontend default endpoint to native Google API.

**Architecture:** Add a dedicated `internal/provider/google` adapter implementing the existing `provider.Provider` interface. Wire it into bootstrap factory registration so runtime account routing creates native Google providers. Keep proxy routing contract unchanged, but ensure Google model lists and defaults stay consistent. Revert frontend Google endpoint constants from OpenAI-compatible path to native `v1beta` path.

**Tech Stack:** Go (Gin, net/http, provider abstraction), Vue 3 + TypeScript + Vitest.

---

### Task 1: Frontend endpoint rollback (TDD)

**Files:**
- Modify: `web/src/constants/pages.static-config.test.ts`
- Modify: `web/src/constants/pages/providers.ts`
- Modify: `web/src/constants/pages/providers-accounts.ts`

**Step 1: Write the failing test**
- Change endpoint assertion to native Google path: `https://generativelanguage.googleapis.com/v1beta`.

**Step 2: Run test to verify it fails**
- Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`
- Expected: FAIL because constants still point to OpenAI-compatible endpoint.

**Step 3: Write minimal implementation**
- Update Google default endpoint in both constants files to native path.

**Step 4: Run test to verify it passes**
- Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`
- Expected: PASS.

### Task 2: Google native provider package (TDD)

**Files:**
- Create: `internal/provider/google/client.go`
- Create: `internal/provider/google/converter.go`
- Create: `internal/provider/google/adapter.go`
- Create: `internal/provider/google/client_test.go`
- Create: `internal/provider/google/adapter_test.go`

**Step 1: Write the failing tests**
- Add tests for:
  - request URL path generation (`:generateContent`, `:streamGenerateContent`).
  - JSON conversion from unified request to Google request.
  - response conversion (content + usage + error).
  - adapter `Factory` / `DefaultModels` behavior.

**Step 2: Run tests to verify fail**
- Run: `go test ./internal/provider/google -v`
- Expected: FAIL (missing implementation / mismatched behavior).

**Step 3: Write minimal implementation**
- Implement native Google client + conversion + adapter with streaming support.

**Step 4: Run tests to verify pass**
- Run: `go test ./internal/provider/google -v`
- Expected: PASS.

### Task 3: Bootstrap and proxy integration (TDD)

**Files:**
- Modify: `internal/bootstrap/gateway.go`
- Modify: `internal/bootstrap/gateway_test.go`
- Modify: `internal/handler/proxy_test.go`

**Step 1: Write the failing tests**
- Update bootstrap test to require google factory resolves to provider without using openai adapter fallback.
- Keep/extend proxy model list test for Google model set including `gemini-3.1-pro-preview`.

**Step 2: Run tests to verify fail**
- Run: `go test ./internal/bootstrap ./internal/handler -run "Google|ProviderRegistry|GetModelsForProvider" -v`
- Expected: FAIL before wiring.

**Step 3: Write minimal implementation**
- Register `google` factory with native adapter package import.
- Ensure default model map includes Google defaults.

**Step 4: Run tests to verify pass**
- Run: `go test ./internal/bootstrap ./internal/handler -run "Google|ProviderRegistry|GetModelsForProvider" -v`
- Expected: PASS.

### Task 4: Full verification

**Files:**
- No additional files

**Step 1: Backend verification**
- Run: `go build ./cmd/gateway`

**Step 2: Frontend verification**
- Run: `cd web && npm run typecheck && npm run build`

**Step 3: Focused regression**
- Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`

**Step 4: Summarize evidence**
- Record which commands passed and any unrelated pre-existing failures.

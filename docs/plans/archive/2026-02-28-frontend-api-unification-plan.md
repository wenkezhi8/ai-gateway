# Frontend API Unification and Domain State Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate frontend data inconsistency by enforcing API-only business data flow, centralized response unwrapping, domain state ownership, and backend-persisted business settings.

**Architecture:** Introduce a strict View -> Domain -> API -> Request layering. Views can only read/write through domain modules; domain modules own request state transitions and write-back refresh. Business settings move from `localStorage` to a new backend UI settings API; `localStorage` remains only for UI preferences.

**Tech Stack:** Vue 3 + Pinia + TypeScript + Vitest + Axios, Go + Gin + JSON file persistence.

---

### Task 1: Add Global Guard Tests (forbidden request usage + fallback unwrap patterns)

**Files:**
- Modify: `web/src/constants/pages.static-config.test.ts`

**Step 1: Write the failing test**

```ts
it('disallows direct request module imports in views and stores', () => {
  const targets = ['src/views/cache/index.vue', 'src/views/routing/index.vue', 'src/views/ops/index.vue']
  for (const file of targets) {
    const content = readFileSync(join(process.cwd(), file), 'utf-8')
    expect(content).not.toContain("from '@/api/request'")
  }
})

it('disallows fallback response unwrapping patterns in migrated views', () => {
  const targets = ['src/views/cache/index.vue', 'src/views/routing/index.vue', 'src/views/ops/index.vue']
  for (const file of targets) {
    const content = readFileSync(join(process.cwd(), file), 'utf-8')
    expect(content).not.toMatch(/data\??\.data\s*\|\|\s*data/)
    expect(content).not.toMatch(/res\??\.data\s*\|\|\s*\{\}/)
    expect(content).not.toMatch(/res\??\.data\s*\|\|\s*\[\]/)
  }
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: FAIL (current views still import `@/api/request` and use fallback unwrap).

**Step 3: Minimal implementation**
- Keep failing state for now (no production change in this task).  
- Commit the red test as migration guard baseline.

**Step 4: Commit**

```bash
git add web/src/constants/pages.static-config.test.ts
git commit -m "test(web): add migration guards for request usage and response unwrap patterns"
```

---

### Task 2: Implement Unified Envelope Utilities (TDD)

**Files:**
- Create: `web/src/api/envelope.ts`
- Create: `web/src/api/envelope.test.ts`

**Step 1: Write the failing test**

```ts
import { describe, expect, it } from 'vitest'
import { ApiError, unwrapEnvelope } from './envelope'

describe('unwrapEnvelope', () => {
  it('returns data on success envelope', () => {
    expect(unwrapEnvelope({ success: true, data: { a: 1 } })).toEqual({ a: 1 })
  })

  it('throws ApiError on failed envelope', () => {
    expect(() => unwrapEnvelope({ success: false, error: { code: 'x', message: 'bad' } }))
      .toThrow(ApiError)
  })

  it('accepts protocol-style plain payload when allowPlain=true', () => {
    expect(unwrapEnvelope({ id: 'chatcmpl-1' }, { allowPlain: true })).toEqual({ id: 'chatcmpl-1' })
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/api/envelope.test.ts`  
Expected: FAIL (module not implemented).

**Step 3: Write minimal implementation**

```ts
export class ApiError extends Error {
  constructor(
    public code: string,
    public status?: number,
    public detail?: string
  ) {
    super(code)
  }
}

export function unwrapEnvelope<T>(raw: unknown, opts: { allowPlain?: boolean } = {}): T {
  if (raw && typeof raw === 'object' && 'success' in (raw as Record<string, unknown>)) {
    const env = raw as { success: boolean; data?: T; error?: { code?: string; message?: string; detail?: string } }
    if (env.success) return env.data as T
    throw new ApiError(env.error?.message || 'api_error', undefined, env.error?.detail)
  }
  if (opts.allowPlain) return raw as T
  throw new ApiError('invalid_envelope')
}
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/api/envelope.test.ts`  
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/api/envelope.ts web/src/api/envelope.test.ts
git commit -m "feat(web): add unified api envelope parser and error type"
```

---

### Task 3: Add Domain API Facades for Routing/Cache/Ops/Settings (TDD)

**Files:**
- Create: `web/src/api/routing-domain.ts`
- Create: `web/src/api/cache-domain.ts`
- Create: `web/src/api/ops-domain.ts`
- Create: `web/src/api/settings-domain.ts`
- Create: `web/src/api/domain-facades.test.ts`

**Step 1: Write the failing test**
- Mock `request` and assert each facade method calls endpoint and runs `unwrapEnvelope`.

```ts
it('routing facade unwraps /admin/router/config', async () => {
  ;(request.get as any).mockResolvedValue({ success: true, data: { default_strategy: 'auto' } })
  const data = await getRouterConfig()
  expect(data.default_strategy).toBe('auto')
})
```

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/api/domain-facades.test.ts`  
Expected: FAIL (facade files missing).

**Step 3: Write minimal implementation**
- Each facade method only does:
  1. call `request.*`
  2. `unwrapEnvelope(...)`
  3. return typed business data

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/api/domain-facades.test.ts`  
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/api/routing-domain.ts web/src/api/cache-domain.ts web/src/api/ops-domain.ts web/src/api/settings-domain.ts web/src/api/domain-facades.test.ts
git commit -m "feat(web): add domain api facades with unified envelope unwrapping"
```

---

### Task 4: Implement Routing Domain Store with 4-State Lifecycle (TDD)

**Files:**
- Create: `web/src/store/domain/routing.ts`
- Create: `web/src/store/domain/routing.test.ts`

**Step 1: Write the failing test**
- Verify `init()` transitions `idle -> loading -> success`.
- Verify empty payload transitions to `empty`.
- Verify API error transitions to `error`.

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/store/domain/routing.test.ts`  
Expected: FAIL (store missing).

**Step 3: Write minimal implementation**

```ts
export type LoadState = 'idle' | 'loading' | 'success' | 'empty' | 'error'

const state = ref<LoadState>('idle')
const error = ref<string>('')
async function init() {
  state.value = 'loading'
  try { /* fetch domain data */ state.value = 'success' } catch (e) { state.value = 'error' }
}
```

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/store/domain/routing.test.ts`  
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/store/domain/routing.ts web/src/store/domain/routing.test.ts
git commit -m "feat(web): add routing domain store with unified load states"
```

---

### Task 5: Implement Cache Domain Store with list/detail/write-back refresh (TDD)

**Files:**
- Create: `web/src/store/domain/cache.ts`
- Create: `web/src/store/domain/cache.test.ts`

**Step 1: Write the failing test**
- Verify `init()` loads stats/config/health.
- Verify `deleteEntry()` triggers refresh and preserves state consistency.
- Verify API failure leaves state in `error`.

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/store/domain/cache.test.ts`  
Expected: FAIL.

**Step 3: Write minimal implementation**
- Implement domain actions:
  - `init`
  - `reload`
  - `saveConfig`
  - `deleteEntry`
  - `cleanupInvalid`

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/store/domain/cache.test.ts`  
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/store/domain/cache.ts web/src/store/domain/cache.test.ts
git commit -m "feat(web): add cache domain store with write-after-read consistency"
```

---

### Task 6: Implement Ops Domain Store and Export Flow (TDD)

**Files:**
- Create: `web/src/store/domain/ops.ts`
- Create: `web/src/store/domain/ops.test.ts`

**Step 1: Write the failing test**
- Verify `loadDashboard/loadServices/loadProviders` unified state behavior.
- Verify `exportMetrics` consumes unified API facade and returns plain object for download.

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/store/domain/ops.test.ts`  
Expected: FAIL.

**Step 3: Write minimal implementation**
- Add unified ops domain state + actions.

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/store/domain/ops.test.ts`  
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/store/domain/ops.ts web/src/store/domain/ops.test.ts
git commit -m "feat(web): add ops domain store and export action"
```

---

### Task 7: Migrate Routing/Cache/Ops Views to Domain Stores (TDD)

**Files:**
- Modify: `web/src/views/routing/index.vue`
- Modify: `web/src/views/cache/index.vue`
- Modify: `web/src/views/ops/index.vue`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Step 1: Keep guard tests red**
- Re-run guard tests from Task 1 and observe current failures.

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: FAIL.

**Step 3: Write minimal implementation**
- Remove `@/api/request` direct imports from three views.
- Replace with domain store imports and action calls.
- Remove all fallback unwrapping patterns (`data?.data || data`, `res?.data || []`, `res?.data || {}`).

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: PASS for new guard cases.

**Step 5: Commit**

```bash
git add web/src/views/routing/index.vue web/src/views/cache/index.vue web/src/views/ops/index.vue web/src/constants/pages.static-config.test.ts
git commit -m "refactor(web): route cache ops views to domain stores only"
```

---

### Task 8: Remove Frontend Business Hardcoded Defaults (TDD)

**Files:**
- Modify: `web/src/store/chat.ts`
- Modify: `web/src/store/models.ts`
- Modify: `web/src/constants/store/chat.ts`
- Modify: `web/src/constants/store/models.ts`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Step 1: Write the failing test**
- Extend static-config test to assert:
  - `constants/store/chat.ts` does not contain business model list arrays.
  - `constants/store/models.ts` does not contain model catalog arrays.
  - stores do not fallback to business model/provider defaults when API returns empty.

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: FAIL (arrays still present).

**Step 3: Write minimal implementation**
- Keep only UI metadata constants (labels/colors/logo rules).
- Move provider/model source loading to API facade calls.
- Empty-state should be explicit UI state, not hidden defaults.

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/store/chat.ts web/src/store/models.ts web/src/constants/store/chat.ts web/src/constants/store/models.ts web/src/constants/pages.static-config.test.ts
git commit -m "refactor(web): remove business hardcoded defaults from chat and models stores"
```

---

### Task 9: Add Backend UI Settings API (TDD)

**Files:**
- Create: `internal/handler/admin/settings.go`
- Create: `internal/handler/admin/settings_test.go`
- Modify: `internal/handler/admin/admin.go`
- Modify: `internal/constants/routes.go`

**Step 1: Write the failing test**
- Add Gin handler tests:
  - `GET /api/admin/settings/ui` returns `{success:true,data:...}` default payload.
  - `PUT /api/admin/settings/ui` persists payload and subsequent GET returns updated data.
  - invalid payload returns structured error object.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/admin -run "Settings" -v`  
Expected: FAIL (handler/route missing).

**Step 3: Write minimal implementation**

```go
type UISettings struct {
  Routing struct {
    AutoSaveEnabled bool   `json:"auto_save_enabled"`
    LastSavedAt     string `json:"last_saved_at"`
  } `json:"routing"`
  ModelManagement struct {
    LastSavedAt string `json:"last_saved_at"`
  } `json:"model_management"`
  Settings map[string]any `json:"settings"`
}
```

- File-backed persistence (`./data/ui-settings.json`) with mutex.
- Register routes under admin group: `GET /settings/ui`, `PUT /settings/ui`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/admin -run "Settings" -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/handler/admin/settings.go internal/handler/admin/settings_test.go internal/handler/admin/admin.go internal/constants/routes.go
git commit -m "feat(admin): add ui settings api for frontend business config persistence"
```

---

### Task 10: Migrate settings/model-management/routing business localStorage to backend settings API (TDD)

**Files:**
- Modify: `web/src/views/settings/index.vue`
- Modify: `web/src/views/model-management/index.vue`
- Modify: `web/src/views/routing/index.vue`
- Modify: `web/src/api/settings-domain.ts`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Step 1: Write the failing test**
- Add static assertions:
  - disallow `ai-gateway-settings` business key persistence in views.
  - disallow `routing_task_mapping_auto_save`/`routing_task_mapping_last_saved` localStorage writes.
  - allow only explicit UI preference keys.

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: FAIL.

**Step 3: Write minimal implementation**
- Replace business key localStorage reads/writes with `settings-domain` API GET/PUT.
- Keep theme/locale related localStorage untouched.

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/views/settings/index.vue web/src/views/model-management/index.vue web/src/views/routing/index.vue web/src/api/settings-domain.ts web/src/constants/pages.static-config.test.ts
git commit -m "refactor(web): migrate business config persistence from localStorage to settings api"
```

---

### Task 11: Enforce API-only entrypoints in migrated modules (TDD)

**Files:**
- Modify: `web/src/store/alerts.ts`
- Modify: `web/src/composables/useGlobalData.ts`
- Modify: `web/src/constants/pages.static-config.test.ts`

**Step 1: Write the failing test**
- Add checks ensuring migrated stores/composables import from `@/api/*-domain` or domain stores, not `@/api/request`.

**Step 2: Run test to verify it fails**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: FAIL.

**Step 3: Write minimal implementation**
- Route `alerts` and global data loading to typed API/domain façade calls.
- Remove direct request transport dependency from high-level state modules.

**Step 4: Run test to verify it passes**

Run: `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`  
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/store/alerts.ts web/src/composables/useGlobalData.ts web/src/constants/pages.static-config.test.ts
git commit -m "refactor(web): enforce api facade boundaries in stores and composables"
```

---

### Task 12: End-to-End Verification and Evidence Collection

**Files:**
- No new files

**Step 1: Frontend unit tests**

Run: `cd web && npm run test:unit`  
Expected: PASS (or clearly listed pre-existing unrelated failures).

**Step 2: Frontend type and build**

Run: `cd web && npm run typecheck && npm run build`  
Expected: PASS.

**Step 3: Backend tests and build**

Run: `go test ./internal/handler/admin -v && go build ./cmd/gateway`  
Expected: PASS.

**Step 4: Focused behavior regression**

Run:
- `cd web && npm run test:unit -- src/constants/pages.static-config.test.ts`
- `cd web && npm run test:unit -- src/store/domain/routing.test.ts src/store/domain/cache.test.ts src/store/domain/ops.test.ts`
- `go test ./internal/handler/admin -run "Settings" -v`

Expected: PASS.

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat(web): complete api-only data flow and domain state unification"
```

---

## Execution Notes

1. Follow strict Red -> Green -> Refactor in each task; do not skip failing-test proof.
2. Keep each task isolated; do not batch multiple tasks into one commit.
3. Preserve protocol-compatible APIs (`/api/v1`, `/api/anthropic`) as plain payload consumers; do not force envelope there.
4. If unrelated workspace changes appear, stop and ask user before proceeding.

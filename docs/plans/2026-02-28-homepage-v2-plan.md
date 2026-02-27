# Homepage v2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign AI Gateway homepage around Superpowers workflow and TDD execution while preserving fast-start usability.

**Architecture:** Keep content as data in `content.ts`, validate structure via `content.test.ts`, and render sections in `index.vue` with responsive layout. Use TDD contract tests first, then implement minimal data/model updates, then UI refactor.

**Tech Stack:** Vue 3 (`<script setup>`), TypeScript, Element Plus, SCSS, Vitest

---

### Task 1: Expand content contract tests (RED)

**Files:**
- Modify: `/Users/openclaw/ai-gateway/web/src/views/home/content.test.ts`
- Test: `/Users/openclaw/ai-gateway/web/src/views/home/content.test.ts`

**Step 1: Write failing tests**

Add assertions for:
- `FLOW_NODES` length = 4
- `CAPABILITY_COLUMNS` length = 4 and each contains at least 3 points
- `QUICK_START_COMMANDS` includes `docker/source/api`

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit -- src/views/home/content.test.ts
```

Expected:
- FAIL for missing imports/assertions target fields (before implementation updates)

**Step 3: Commit**

```bash
git add /Users/openclaw/ai-gateway/web/src/views/home/content.test.ts
git commit -m "test(home): extend homepage v2 content contract"
```

---

### Task 2: Update content model (GREEN)

**Files:**
- Modify: `/Users/openclaw/ai-gateway/web/src/views/home/content.ts`
- Test: `/Users/openclaw/ai-gateway/web/src/views/home/content.test.ts`

**Step 1: Minimal implementation**

Ensure exports satisfy tests:
- `HERO_ACTIONS`
- `WORKFLOW_STEPS`
- `TDD_STAGES`
- `FLOW_NODES`
- `CAPABILITY_COLUMNS`
- `QUICK_START_COMMANDS`

**Step 2: Run focused test**

```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit -- src/views/home/content.test.ts
```

Expected:
- PASS

**Step 3: Commit**

```bash
git add /Users/openclaw/ai-gateway/web/src/views/home/content.ts /Users/openclaw/ai-gateway/web/src/views/home/content.test.ts
git commit -m "feat(home): align v2 content model with workflow-first structure"
```

---

### Task 3: Refactor homepage layout (REFACTOR)

**Files:**
- Modify: `/Users/openclaw/ai-gateway/web/src/views/home/index.vue`

**Step 1: Section structure**
- Hero with three CTA actions
- Flow visualization panel
- Workflow section with 6 steps
- TDD section with 4 stages
- Capability matrix
- Quick start tabs and command copy

**Step 2: Responsive style**
- Add/adjust breakpoints at `1024px` and `768px`
- Keep mobile single-column readability

**Step 3: Commit**

```bash
git add /Users/openclaw/ai-gateway/web/src/views/home/index.vue
git commit -m "feat(home): implement homepage v2 visual and section layout"
```

---

### Task 4: Full verification (VERIFY)

**Files:**
- Validate only (no required file change)

**Step 1: Run verification commands**

```bash
cd /Users/openclaw/ai-gateway/web && npm run test:unit
cd /Users/openclaw/ai-gateway/web && npm run typecheck
cd /Users/openclaw/ai-gateway/web && npm run build
cd /Users/openclaw/ai-gateway && ./scripts/dev-restart.sh
```

Expected:
- All checks pass

**Step 2: Commit verification milestone**

```bash
git commit --allow-empty -m "chore(home): verify homepage v2 rollout"
```

---

### Task 5: Final delivery report

**Files:**
- None

**Step 1: Report in AGENTS-required order**
- 根因
- 方案
- 改动清单
- 测试结果
- 风险与回滚
- 接口一致性
- 版本建议


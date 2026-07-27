# System Config Load Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent every save action inside `/system/config` from writing defaults or stale values when its source configuration failed to load.

**Architecture:** Keep the existing React Query hooks, forms, batch API, and dedicated Agent API. Each owner component derives writability from its own query (`isSuccess && !isFetching`), renders the existing load-failure/retry UI pattern, and repeats the guard inside the handler. Generic `sys_configs` forms serialize only touched fields; Agent keeps its existing full update contract after a successful load.

**Tech Stack:** React 19, TypeScript, Ant Design 6, TanStack Query 5, Vitest, Testing Library.

## Global Constraints

- Every change must match an existing OMC system-configuration business path.
- Do not add a generic configuration framework, new backend contract, database migration, or unrelated refactor.
- Agent configuration is sensitive: failed loads must disable save, test, and sync operations.
- Preserve write-only secret behavior and existing successful-load behavior.
- Do not commit or push unless the user explicitly requests it.

---

### Task 1: Guard the shared SystemConfig form

**Files:**
- Create: `omcmb/webcode/src/pages/system/SystemConfig/SystemConfig.loadGuard.test.tsx`
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/index.tsx`

**Interfaces:**
- Consumes: `useSysConfigsByCategory(category, enabled)` query state and existing `buildBatchItems`.
- Produces: a disabled save action during initial load, refetch, and error; successful writes contain only touched fields.

- [ ] **Step 1: Write failing UI tests**

Mock the SystemConfig child tabs and `useSystem` hooks. Assert that an error-state query renders `empty.loadFailed`, disables the basic-tab save button, and never calls `mutateAsync`. Assert that a successful query enables saving, and after editing one Basic field the mutation contains only that field.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd omcmb && npm run test --workspace webcode -- --run src/pages/system/SystemConfig/SystemConfig.loadGuard.test.tsx
```

Expected: FAIL because the current save button is enabled in the query error state and the current serializer sends all form values.

- [ ] **Step 3: Implement the minimal guard**

In `index.tsx`:

- Read `isError`, `isSuccess`, and `refetch` from the existing query.
- Hydrate/reset the form only after a successful query.
- Render an Ant Design error `Alert` with the existing `empty.loadFailed`, `empty.loadFailedDesc`, and `common.retry` messages.
- Disable save unless the current category query succeeded and is not fetching.
- Repeat the same condition at the start of `handleSave`.
- Filter validated form values through `form.isFieldTouched(key)` before calling `buildBatchItems`.
- Mark server-hydrated fields as `touched: false` so they are not treated as operator edits.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run the Task 1 command again. Expected: all tests in `SystemConfig.loadGuard.test.tsx` pass.

### Task 2: Guard self-managed sys_config cards

**Files:**
- Create: `omcmb/webcode/src/pages/system/SystemConfig/SelfManagedConfig.loadGuard.test.tsx`
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/PmRetentionSection.tsx`
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/RetentionBackpressureSection.tsx`
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/LogRetentionSection.tsx`

**Interfaces:**
- Consumes: each card's existing category query and batch mutation.
- Produces: independent load/error/write state per category card.

- [ ] **Step 1: Add failing tests for each card owner**

Add tests that return `isError: true`, `isSuccess: false`, and no data for PM retention, all retention/backpressure cards, and all log cards. Assert that every affected save button is disabled, retry remains available, and no mutation occurs. Add one successful-load test proving that only a touched field is sent.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd omcmb && npm run test --workspace webcode -- --run src/pages/system/SystemConfig/SelfManagedConfig.loadGuard.test.tsx
```

Expected: the new card tests fail because existing buttons ignore query errors.

- [ ] **Step 3: Implement per-card guards**

For each component:

- Read `isError`, `isSuccess`, `isFetching`, and `refetch` from the existing query.
- Populate form values only on successful loads and mark hydrated fields untouched.
- Render the same existing load-failure/retry messages.
- Disable save during loading, refetch, or error and repeat the check inside the save handler.
- Serialize only fields for which `form.isFieldTouched(key)` is true.
- Preserve PM Reset as an explicit operator change by setting reset fields with `touched: true`.
- Preserve the existing no-saveable warning when no fields changed.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run the Task 2 command again. Expected: all card guard tests pass.

### Task 3: Guard sensitive Agent operations

**Files:**
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/AgentSettings.test.tsx`
- Modify: `omcmb/webcode/src/pages/system/SystemConfig/AgentSettings.tsx`

**Interfaces:**
- Consumes: `useAdminAgentConfig()` state.
- Produces: save/test/sync operations available only after a successful configuration load; refetch remains available.

- [ ] **Step 1: Write a failing Agent error-state test**

Make the query mock configurable. Return `isError: true`, `isSuccess: false`, no data, and assert that Save, Test, and Sync are disabled while Retry/Reset remains enabled. Assert the load-failure alert is visible.

- [ ] **Step 2: Run the Agent test and verify RED**

Run:

```bash
cd omcmb && npm run test --workspace webcode -- --run src/pages/system/SystemConfig/AgentSettings.test.tsx
```

Expected: FAIL because sensitive Agent actions are currently enabled after a failed load.

- [ ] **Step 3: Implement the Agent guard**

Derive readiness from successful query data and no active fetch. Render the existing load-failure/retry alert. Disable Save, Test, and Sync when not ready; leave the refetch action enabled. Add a defensive readiness check before payload validation. Do not change the Agent payload, policy, or backend API.

- [ ] **Step 4: Run the Agent test and verify GREEN**

Run the Task 3 command again. Expected: all Agent tests pass.

### Task 4: Verify the complete frontend scope

**Files:**
- Verify only; no additional implementation files.

**Interfaces:**
- Consumes: Tasks 1-3.
- Produces: evidence that CFG-M01 is fixed without changing unrelated behavior.

- [ ] **Step 1: Run SystemConfig tests**

```bash
cd omcmb && npm run test --workspace webcode -- --run src/pages/system/SystemConfig
```

Expected: all SystemConfig test files pass.

- [ ] **Step 2: Run frontend type checking**

```bash
cd omcmb && npm run typecheck
```

Expected: TypeScript exits with code 0.

- [ ] **Step 3: Inspect the final diff**

```bash
git diff --check
git diff -- omcmb/webcode/src/pages/system/SystemConfig
```

Expected: no whitespace errors; changes are limited to load guards, touched-field serialization, tests, and the approved documents.

- [ ] **Step 4: Browser acceptance when the web stack is available**

Verify the actual `/system/config` DOM with a failed configuration request: affected save actions remain disabled, retry is available, and no batch request is emitted. If the stack is unavailable, report browser acceptance as not run rather than inferring success from source tests.

# #36 V1 Query Template Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make an edited active V1 KPI query template immediately update the query form and expose a complete read-only template detail view, while keeping existing query results until the user explicitly queries again.

**Architecture:** Treat the template returned by PATCH as the authoritative state. Update React Query caches from that response, use a small pure state helper to decide whether the active form changes, and isolate the read-only details UI in a focused V1 component.

**Tech Stack:** React 19, TypeScript 6, TanStack Query 5, Ant Design 6, Vitest, Testing Library.

## Global Constraints

- This MR modifies V1 only: `omcmb/webcode` and shared code consumed by V1.
- Do not modify `webcode-v2` or `webcode-v3` for #36.
- Saving does not automatically query and does not replace `submittedPayload` or displayed results.
- Editing a non-active template does not alter the current query form.
- All visible text uses shared i18n keys.

---

### Task 1: Synchronize updated templates into React Query caches

**Files:**
- Modify: `omcmb/frontend-core/src/hooks/api/usePmQuery.ts`
- Create: `omcmb/frontend-core/src/hooks/api/__tests__/usePmQueryTemplateUpdate.test.tsx`

**Interfaces:**
- Consumes: `pmQueryApi.update(id, input): Promise<QueryTemplate>`.
- Produces: `useUpdateQueryTemplate()` whose mutation result remains `QueryTemplate` and whose success handler replaces the matching item in every `['pm-query-templates','list',...]` cache plus `['pm-query-templates','detail',id]`.

- [ ] **Step 1: Write the failing cache synchronization test**

Create a QueryClient wrapper, seed the list with a `daily` template, mock `pmQueryApi.update` to return the same template with `hourly`, call the mutation, and assert the cached list and detail contain `hourly`.

- [ ] **Step 2: Run the test and verify RED**

Run: `cd omcmb && npm run test --workspace webcode -- ../frontend-core/src/hooks/api/__tests__/usePmQueryTemplateUpdate.test.tsx`

Expected: FAIL because the list cache still contains `daily` and detail cache is unset.

- [ ] **Step 3: Implement minimal authoritative-response cache updates**

In `onSuccess(updated, vars)`, call `setQueriesData` for list queries and replace only the matching item while preserving `total`; call `setQueryData` for the detail key; retain invalidation as server reconciliation.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run the command from Step 2. Expected: one passing test file.

---

### Task 2: Synchronize the active V1 query form without touching submitted results

**Files:**
- Create: `omcmb/webcode/src/pages/performance/KPIQuery/templateUpdateState.ts`
- Create: `omcmb/webcode/src/pages/performance/KPIQuery/templateUpdateState.test.ts`
- Modify: `omcmb/webcode/src/pages/performance/KPIQuery/index.tsx`

**Interfaces:**
- Produces: `applyUpdatedTemplateToActiveForm(activeTemplateId: string | undefined, currentPayload: QueryTemplatePayload, updated: QueryTemplate): QueryTemplatePayload`.
- The helper returns `updated.payload` only when IDs match; otherwise it returns `currentPayload` unchanged.

- [ ] **Step 1: Write failing pure-state tests**

Cover two cases: active template changes from `daily` to `hourly`; a different template leaves the current payload object unchanged.

- [ ] **Step 2: Run tests and verify RED**

Run: `cd omcmb && npm run test --workspace webcode -- src/pages/performance/KPIQuery/templateUpdateState.test.ts`

Expected: FAIL because the helper module does not exist.

- [ ] **Step 3: Implement the minimal helper**

Return `activeTemplateId === updated.id ? updated.payload : currentPayload`.

- [ ] **Step 4: Wire the PATCH response into V1 state**

Capture `const updated = await updateMut.mutateAsync(...)`. If it is active, call `setPayload` with the helper result, reset `timeRangeDirty`, and derive `customRange` from the returned payload. Do not call `setSubmittedPayload`, `setSubmittedRange`, `handleQuery`, or `refetchAgg`.

- [ ] **Step 5: Run focused tests and V1 typecheck**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- src/pages/performance/KPIQuery/templateUpdateState.test.ts
npm run typecheck --workspace webcode
```

Expected: tests and TypeScript pass.

---

### Task 3: Add a complete V1 template detail modal

**Files:**
- Create: `omcmb/webcode/src/pages/performance/KPIQuery/QueryTemplateDetailModal.tsx`
- Create: `omcmb/webcode/src/pages/performance/KPIQuery/QueryTemplateDetailModal.test.tsx`
- Modify: `omcmb/webcode/src/pages/performance/KPIQuery/index.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Produces: `QueryTemplateDetailModal({ open, template, metricLabels, onClose })`.
- Displays name, visibility, description, device type, device SN list, friendly metric names with ID fallback, granularity, time-range preset/custom range, created time, and updated time.

- [ ] **Step 1: Write the failing detail rendering test**

Render an hourly private template and assert visible labels/values for `hourly`, device SN, friendly metric name, visibility, and time range.

- [ ] **Step 2: Run the test and verify RED**

Run: `cd omcmb && npm run test --workspace webcode -- src/pages/performance/KPIQuery/QueryTemplateDetailModal.test.tsx`

Expected: FAIL because the component does not exist.

- [ ] **Step 3: Implement the minimal read-only modal**

Use Ant Design `Modal`, `Descriptions`, `Tag`, and `Typography`; all labels come from `perf.kpiQuery.detail.*` i18n keys in both locales.

- [ ] **Step 4: Add the V1 detail entry**

Add a detail icon action before edit/delete in each template list item, stop event propagation, store the selected detail template, and render `QueryTemplateDetailModal`. When the list cache updates, resolve the displayed template by ID so an open detail modal shows the new payload.

- [ ] **Step 5: Run component tests and V1 typecheck**

Run the focused test command from Step 2 and `cd omcmb && npm run typecheck --workspace webcode`.

Expected: all pass.

---

### Task 4: Regression verification and Docker acceptance

**Files:**
- Modify only if a verification-discovered defect requires a new failing test first.

**Interfaces:**
- Verifies the complete V1 flow; produces no new API.

- [ ] **Step 1: Run all relevant frontend verification**

Run:

```bash
cd omcmb
npm run test --workspace webcode -- ../frontend-core/src/hooks/api/__tests__/usePmQueryTemplateUpdate.test.tsx src/pages/performance/KPIQuery/templateUpdateState.test.ts src/pages/performance/KPIQuery/QueryTemplateDetailModal.test.tsx
npm run typecheck
```

Expected: focused tests, skin parity, and all currently configured typechecks pass.

- [ ] **Step 2: Run backend template API regression**

Run: `cd omcgo && GOCACHE=/tmp/omc-go-build go test ./internal/pm/querytemplate -count=1`

Expected: PASS.

- [ ] **Step 3: Rebuild and start Docker services**

Run: `docker compose -p omc -f deployments/docker/docker-compose.yml up -d --build app web`

Expected: app and web containers start; `docker compose ... ps` reports running/healthy services.

- [ ] **Step 4: Execute real acceptance**

Create a temporary V1 private template with `daily`, edit it to `hourly`, and verify: PATCH response and database payload are hourly; the V1 query form shows hourly without reselecting; no query fires during save; clicking Query sends `granularity=hourly`; details show hourly and full configuration.

- [ ] **Step 5: Clean temporary data and check diff**

Delete the temporary template, confirm its database count is zero, run `git diff --check`, and confirm `.codex/` remains untracked and unstaged.

---

### Task 5: Commit, GitLab MR, and merge

**Files:**
- Include only the spec, plan, V1/shared implementation, tests, and i18n files listed above.

**Interfaces:**
- Produces a merged GitLab MR closing #36.

- [ ] **Step 1: Run verification-before-completion checks**

Re-run the focused tests, `npm run typecheck`, backend querytemplate test, Docker status, and `git diff --check` from fresh commands.

- [ ] **Step 2: Commit implementation**

Commit message: `fix(pm): 修复查询模板更新后粒度不生效`

- [ ] **Step 3: Push and create GitLab MR**

Push `fix/issue-36-query-template-refresh`; create an MR targeting `main` with `Closes #36` and the exact verification evidence.

- [ ] **Step 4: Verify and merge**

Confirm GitLab reports mergeable/no conflicts and merge the exact tested SHA, removing the remote source branch.

- [ ] **Step 5: Synchronize local main**

Switch to `main`, pull `--ff-only`, delete the local feature branch, and confirm only local `.codex/` remains untracked.

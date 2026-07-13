# MML Task Pagination and Limits Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make MML task creation usable up to 200 devices and 2000 plan rows with real pagination and matching frontend/backend limits.

**Architecture:** Parsed plan rows and selected SNs stay client-side and use stable array pagination. Device search keeps the existing server-side query pagination. Shared TypeScript limits protect all three skins, while Go service validation enforces the same limits at the API boundary.

**Tech Stack:** React 19, TypeScript, Ant Design 6, Tailwind-based v2/v3 skins, Vitest, Go 1.25, Testify.

## Global Constraints

- Maximum unique devices per task: 200.
- Maximum plan rows per task: 2000.
- Default page size: 20; options: 20, 50, 100.
- Pagination must preserve original plan order and must never silently truncate data.
- Backend request values `common` and `device_bound` remain unchanged.
- The unrelated file `docs/analysis/mml-script-task-create-trigger-issues-20260709.md` is excluded.

---

### Task 1: Shared frontend scale contract

**Files:**
- Create: `omcmb/frontend-core/src/utils/mmlTaskScale.ts`
- Create: `omcmb/frontend-core/src/utils/__tests__/mmlTaskScale.test.ts`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Produces: `MML_MAX_TASK_DEVICES = 200`, `MML_MAX_PLAN_ITEMS = 2000`, `MML_PREVIEW_PAGE_SIZE = 20`, `MML_PREVIEW_PAGE_SIZE_OPTIONS = [20, 50, 100]`, `validateMmlTaskScale(deviceSns, planItemCount)`.

- [x] Write tests proving 200 devices/2000 rows pass, while 201 devices or 2001 rows return a typed limit issue.
- [x] Run `npm test --workspace webcode -- mmlTaskScale.test.ts` and confirm RED because the utility does not exist.
- [x] Implement constants, unique-SN counting, and validation; add localized error messages with current and maximum values.
- [x] Re-run the focused test and confirm GREEN.

### Task 2: Backend API enforcement

**Files:**
- Modify: `omcgo/internal/mml/service.go`
- Modify: `omcgo/internal/mml/service_test.go`

**Interfaces:**
- Consumes: `ExecuteRequest.DeviceSNs`, `ExecuteRequest.PlanItems`.
- Produces: `commonerrors.ErrInvalidInput` errors before task persistence when either limit is exceeded.

- [x] Add failing service tests for 201 common-mode devices, 2001 plan rows, and 201 unique device SNs in device-bound rows.
- [x] Run the three focused Go tests and confirm RED.
- [x] Implement early constants and validation in `ExecuteCommand`/`normalizePlanItems` without changing valid requests.
- [x] Re-run focused MML service tests and confirm GREEN.

### Task 3: v1 previews and selected devices

**Files:**
- Create: `omcmb/webcode/src/pages/mml/components/PaginatedDeviceSnList.tsx`
- Create: `omcmb/webcode/src/pages/mml/components/__tests__/PaginatedDeviceSnList.test.tsx`
- Modify: `omcmb/webcode/src/pages/mml/ScriptTask/index.tsx`
- Modify: `omcmb/webcode/src/pages/mml/components/ScriptTaskDrawer.tsx`
- Modify: `omcmb/webcode/src/pages/mml/Console/constants.ts`

**Interfaces:**
- Produces: a selected-device summary/table with removal and 20/50/100 paging; Ant Design plan tables with visible totals and size controls.

- [x] Write a failing component test proving 25 selected SNs render only the first 20, page 2 shows the remainder, and removal emits the next list.
- [x] Implement the reusable device list and replace the tag-only selected SN presentation.
- [x] Configure both v1 plan tables for 20/50/100 pagination and enforce shared scale validation before save/submit.
- [x] Re-run component and scale tests.

### Task 4: v2/v3 real pagination

**Files:**
- Modify: `omcmb/webcode-v2/src/pages/mml/ScriptTask.tsx`
- Modify: `omcmb/webcode-v3/src/pages/mml/script/index.tsx`

**Interfaces:**
- Consumes: shared scale constants and parsed plan arrays.
- Produces: page-controlled previews that render all rows across pages instead of `slice(0, 20/30)` truncation, plus paginated parsed/entered device summaries.

- [x] Replace fixed slices with page/page-size state and visible total/range controls in both skins.
- [x] Validate 200/2000 limits before create/execute submission and show explicit errors.
- [x] Confirm no fixed preview slice remains in the two files.

### Task 5: Verification and delivery

**Files:**
- Modify: `docs/superpowers/plans/2026-07-10-mml-task-pagination-limits.md`

- [x] Run focused Vitest and Go tests.
- [x] Run `cd omcmb && npm run skin-parity && npm run typecheck`.
- [x] Run `git diff --check` and scoped ESLint.
- [x] Rebuild/deploy the local web/app stack and verify v1/v2/v3 plus app health endpoints.
- [x] Fetch latest `origin/main`, run a non-destructive conflict check, commit, push, and create a GitLab MR.

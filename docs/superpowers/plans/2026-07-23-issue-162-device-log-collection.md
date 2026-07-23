# Issue 162 Device Log Collection UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep users on the device list after triggering runtime-log collection, show a clear task-management hint, and remove the obsolete per-device log download column.

**Architecture:** Keep the existing `RUNTIME_LOG_COLLECT` UFTE task creation flow unchanged. Change only the V1 device-list success feedback and column definitions; retain station-log download APIs for the dedicated log and file-management pages.

**Tech Stack:** React, TypeScript, Ant Design, React Router, Vitest/Playwright, project i18n dictionaries.

## Global Constraints

- Modify only the V1 `webcode` shell and shared i18n strings required by this behavior.
- Do not change UFTE backend APIs, task payloads, execution mode, or task-management polling.
- User-visible text must be present in both `zh-CN` and `en-US`.
- Preserve existing unrelated working-tree changes.

---

### Task 1: Add device-list interaction regression coverage

**Files:**
- Modify: `omcmb/webcode/e2e/device-list.spec.ts`

**Interfaces:**
- Consumes: device-list batch selection and `RUNTIME_LOG_COLLECT` task creation in mock mode.
- Produces: regression coverage for remaining on `/device/list`, the success hint, and permanent removal of the log-download column.

- [x] **Step 1: Write the failing tests**

Add one test that exposes every column through the legacy visibility key and asserts that `latestLog` still cannot render, plus one test that triggers log collection and expects the new hint while the URL remains `/device/list`.

- [x] **Step 2: Run tests to verify they fail**

Run:

```bash
cd omcmb/webcode
npx playwright test e2e/device-list.spec.ts --project=chromium --grep "日志收集"
```

Expected: FAIL because the old “任务已创建，即将跳转” toast appears and navigation changes to `/transfer/center`, or because the runtime-log column is still rendered when legacy column visibility exposes it.

- [x] **Step 3: Keep the tests unchanged for the implementation cycle**

Do not weaken URL, message, or column assertions after observing the expected failures.

### Task 2: Change runtime-log collection success behavior

**Files:**
- Modify: `omcmb/webcode/src/pages/device/DeviceList/index.tsx:1076-1097`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Consumes: `createUfteTask.mutateAsync(CreateUnifiedFileTransferTaskInput)`.
- Produces: `device.action.logCollectTriggered` success feedback without automatic navigation.

- [x] **Step 1: Replace redirecting feedback with the approved message**

After `mutateAsync` succeeds, show:

```ts
void message.success({
  content: t('device.action.logCollectTriggered'),
  duration: 5,
});
```

Remove only the subsequent `navigate('/transfer/center?...')` call; keep the task payload and failure behavior unchanged.

- [x] **Step 2: Add bilingual text and remove the obsolete redirect key**

Add:

```ts
'device.action.logCollectTriggered': '日志收集已触发，请前往“文件传输 > 任务管理”查看任务进度。',
```

and:

```ts
'device.action.logCollectTriggered': 'Log collection triggered. Check progress in File Transfer > Task Management.',
```

Delete the now-unused `ufte.taskCreatedAndNavigate` entries.

### Task 3: Remove the device-list log download column

**Files:**
- Modify: `omcmb/webcode/src/pages/device/DeviceList/index.tsx:42-43,585,2132-2161`
- Modify: `omcmb/webcode/src/pages/device/DeviceList/deviceExport.ts:20-56`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`
- Modify: `omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx`

**Interfaces:**
- Consumes: `DataTableColumn<Device>[]` and persisted column-order/visibility state.
- Produces: a device-list schema that contains no `latestLog` column; stale persisted keys are ignored by the existing `Map` lookup.

- [x] **Step 1: Delete the column and device-list-only dependencies**

Delete the `latestLog` column block, `useDownloadStationLog`, `stationLogApi`, the mutation instance, and the hook from the `columns` dependency list.

- [x] **Step 2: Clean related export and i18n metadata**

Change the non-exportable set to:

```ts
const NON_EXPORTABLE_KEYS = new Set<string>(['actions']);
```

Remove `device.latestLog` and `device.noLogFile`, which become unused. Update comments that still describe automatic navigation from the device list.

- [x] **Step 3: Run the focused test to verify green**

Run:

```bash
cd omcmb/webcode
npx playwright test e2e/device-list.spec.ts --project=chromium --grep "日志收集"
```

Expected: PASS.

### Task 4: Verify the frontend

**Files:**
- Verify all files modified above.

**Interfaces:**
- Consumes: final working tree.
- Produces: fresh test and type-check evidence.

- [x] **Step 1: Run the device-list test file**

```bash
cd omcmb/webcode
npx playwright test e2e/device-list.spec.ts --project=chromium
```

Expected: all tests pass.

- [x] **Step 2: Run V1 unit tests**

```bash
cd omcmb
npm test --workspace webcode
```

Expected: all tests pass.

Observed on the current branch: 1380 tests passed and 24 unrelated baseline tests failed in MML mocks, alarm-rule copy assertions, source-file URL handling, one UFTE category assertion, one KPI timeout, and one pre-existing English i18n message.

- [x] **Step 3: Run required type checking**

```bash
cd omcmb
npm run typecheck
```

Expected: exit code 0.

- [x] **Step 4: Review the final diff**

Confirm that only the approved UX, column removal, related tests, i18n cleanup, comments, and this plan changed.

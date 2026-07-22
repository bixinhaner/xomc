# Issue #157 RF Physical Cell Projection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Make the device-list RF status represent the device's own physical carriers, with BSC shown as not applicable and SC/DC/CA derived from per-device configuration.

**Architecture:** Keep RF projection in the Go backend, where raw parameters and device identity are available. Resolve a strict carrier count, map RF path families to physical indices, persist CSV only for a complete consistent set, and add a small frontend BSC capability guard for stale rows.

**Tech Stack:** Go 1.x, testify, React/TypeScript, Vitest, Ant Design.

## Global Constraints

- Do not infer current carrier count from the product assembly `radio_modes` capability list.
- Valid `NumOfCells` is authoritative; `/SC` and `/DC` are consistency checks and missing-value fallbacks; `/CA` requires `NumOfCells`.
- Do not truncate or duplicate RF values to satisfy a count.
- BSC `FAP/PGSM` never exposes child RF as its own RF.
- Preserve the existing `device_info.rf_status` CSV and external API contract.

---

### Task 1: Backend RF projection pure function

**Files:**
- Modify: `omcgo/internal/device/device_info_calc.go`
- Test: `omcgo/internal/device/info_calc_test.go`

**Interfaces:**
- Consumes: `map[string]string`, `model.Technology`, `productClass string`.
- Produces: `RFStatusProjection{Status, State, Reason, ExpectedCount}` through `CalcRFStatus(params, tech, productClass)`.

- [x] **Step 1: Write failing table tests for physical-carrier behavior**

Add cases that call:

```go
got := CalcRFStatus(tt.params, tt.tech, tt.productClass)
assert.Equal(t, tt.wantStatus, got.Status)
assert.Equal(t, tt.wantState, got.State)
```

The table must include BAIBLQ/SC with two FAPService values, DC with four discovered values, missing DC index 2, SC/NumOfCells conflict, CA without NumOfCells, BTS duplicate instances, BSC child RF, two NR CellConfig values, and duplicate same-index conflict.

- [x] **Step 2: Run the focused test and verify RED**

Run:

```bash
cd omcgo && go test ./internal/device -run 'TestCalcRFStatus' -count=1 -v
```

Expected: compile/test failure because the current function returns a string and has no device context.

- [x] **Step 3: Implement strict count and indexed-family projection**

Add internal types:

```go
type RFStatusProjectionState string

const (
    RFStatusValid        RFStatusProjectionState = "valid"
    RFStatusUnknown      RFStatusProjectionState = "unknown"
    RFStatusUnsupported  RFStatusProjectionState = "unsupported"
    RFStatusInconsistent RFStatusProjectionState = "inconsistent"
)

type RFStatusProjection struct {
    Status        string
    State         RFStatusProjectionState
    Reason        string
    ExpectedCount int
}
```

Implement strict `NumOfCells` parsing, `/SC` and `/DC` suffix parsing, BSC/BTS handling, path-family-specific index extraction, `1..N` completeness checks, equal duplicate collapse, and conflicting duplicate rejection. Keep existing normalization values and path-family priority.

- [x] **Step 4: Run focused tests and verify GREEN**

Run the focused command from Step 2. Expected: all `TestCalcRFStatus` cases pass.

- [x] **Step 5: Run adjacent calculation tests**

```bash
cd omcgo && go test ./internal/device -run '^(TestCalcRFStatus|TestCalcNumOfCells|TestCalcCellStatus)$' -count=1
```

Expected: PASS.

### Task 2: Pass ProductClass through device-info synchronization

**Files:**
- Modify: `omcgo/internal/device/device_info_sync.go`
- Modify: `omcgo/internal/device/device_service.go`
- Modify: `omcgo/internal/device/batch_processor.go`
- Modify: `omcgo/internal/device/rpc_response_subscriber.go`
- Modify: `omcgo/internal/provision/sync.go`
- Modify: `omcgo/internal/provision/sync_pathb.go`
- Modify: `omcgo/cmd/app/provider/paramsync.go`
- Test: `omcgo/internal/device/device_info_sync_test.go`
- Test: `omcgo/internal/provision/sync_pathb_test.go`

**Interfaces:**
- Changes `SyncFromParameters(ctx, deviceID, carrier, tech)` to `SyncFromParameters(ctx, deviceID, carrier, tech, productClass)`.
- Consumes `RFStatusProjection` from Task 1.
- Persists `projection.Status`; warns only for inconsistent or observed-but-incomplete data.

- [x] **Step 1: Add failing sync projection tests**

Create tests whose parameter repositories contain RF paths and whose `updateSyncFields` callbacks assert:

```go
assert.Equal(t, "on,off", fields["rf_status"])
```

for DC, and:

```go
assert.Equal(t, "", fields["rf_status"])
```

for BSC and inconsistent DC.

- [x] **Step 2: Run sync tests and verify RED**

```bash
cd omcgo && go test ./internal/device -run 'TestInfoSyncer_SyncFromParameters_RF' -count=1 -v
```

Expected: compile failure until ProductClass is accepted and the projection result is consumed.

- [x] **Step 3: Update the synchronization contract and production callers**

Change every interface and call to pass `device.ProductClass` (or `dev.ProductClass`). Tests unrelated to RF pass `""`. In `InfoSyncer`:

```go
rf := CalcRFStatus(paramValues, tech, productClass)
fields["rf_status"] = rf.Status
```

Log warnings with device ID, ProductClass, expected count, state and reason; do not log raw parameter values.

- [x] **Step 4: Run sync tests and package tests**

```bash
cd omcgo && go test ./internal/device ./internal/provision ./cmd/app/provider -count=1
```

Expected: PASS.

### Task 3: Frontend BSC guard and sync-path capability filtering

**Files:**
- Modify: `omcmb/webcode/src/pages/device/DeviceList/deviceRadioFieldSupport.ts`
- Test: `omcmb/webcode/src/pages/device/DeviceList/deviceRadioFieldSupport.test.ts`
- Modify: `omcmb/webcode/src/pages/device/DeviceList/deviceListParamSync.ts`
- Test: `omcmb/webcode/src/pages/device/DeviceList/deviceBatchTask.test.ts`
- Modify: `omcmb/webcode/src/pages/device/DeviceList/index.tsx`

**Interfaces:**
- Produces `supportsOwnRF(device): boolean` and `formatDeviceRFStatus(device, value): string`.
- `getDeviceListParamSyncPaths` excludes every `key === 'rfStatus'` entry for BSC.

- [x] **Step 1: Add failing frontend tests**

Assert:

```ts
expect(formatDeviceRFStatus(bsc, 'on,on')).toBe('-');
expect(formatDeviceRFStatus(bts, 'on')).toBe('on');
```

and assert BSC sync paths contain neither `FAPControl.LTE.RFTxStatus` nor `GsmBTSCellDT.{i}.RfState`, while BTS retains both supported GSM RF paths.

- [x] **Step 2: Run focused Vitest and verify RED**

```bash
cd omcmb && npx vitest run webcode/src/pages/device/DeviceList/deviceRadioFieldSupport.test.ts webcode/src/pages/device/DeviceList/deviceBatchTask.test.ts
```

Expected: failure because the RF helpers and BSC filtering do not exist.

- [x] **Step 3: Implement the capability guard**

Use exact case-insensitive `FAP/PGSM` matching. Apply `formatDeviceRFStatus` before the existing multi-cell renderer. Reuse `supportsOwnRF` in `paramAppliesToDevice` so BSC RF paths are not requested.

- [x] **Step 4: Run focused tests and typecheck**

```bash
cd omcmb && npx vitest run webcode/src/pages/device/DeviceList/deviceRadioFieldSupport.test.ts webcode/src/pages/device/DeviceList/deviceBatchTask.test.ts
cd omcmb && npm run typecheck
```

Expected: PASS.

### Task 4: Full verification and review

**Files:**
- Verify only; no new production files.

**Interfaces:**
- Confirms backend, frontend and persisted API behavior remain compatible.

- [x] **Step 1: Format changed source files**

```bash
gofmt -w omcgo/internal/device/device_info_calc.go omcgo/internal/device/device_info_sync.go omcgo/internal/device/info_calc_test.go omcgo/internal/device/device_info_sync_test.go
```

Also format every additional changed Go file if `gofmt -l` lists it.

- [x] **Step 2: Run full backend verification**

```bash
cd omcgo && go build ./...
cd omcgo && go test ./...
```

Expected: PASS; if sandbox blocks local listeners, rerun under the repository permission guidance.

- [x] **Step 3: Run full frontend verification**

```bash
cd omcmb && npm run typecheck
```

Expected: PASS.

- [x] **Step 4: Inspect the final diff and worktree boundaries**

```bash
git diff --check
git status --short
```

Expected: only Issue #157 plan/source/test changes plus the user's pre-existing `AGENTS.md` and issue-149 document.

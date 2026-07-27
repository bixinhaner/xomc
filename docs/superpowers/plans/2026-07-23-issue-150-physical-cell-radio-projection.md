# Issue 150 Physical Cell Radio Projection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make LTE/NR DL and UL EARFCN list values preserve one positional value per physical cell and exclude logical instances outside the configured carrier count.

**Architecture:** Add a focused backend physical-cell frequency projector that reuses the carrier-count rules already implemented for RF status. Keep the existing CSV database/API contract and run the projector after generic instance aggregation so it authoritatively corrects LTE/NR frequency fields without changing GSM or non-cell multi-instance behavior.

**Tech Stack:** Go, regexp, testify, PostgreSQL-backed `device_info` projection, React/TypeScript frontend verification.

## Global Constraints

- Valid `NumOfCells` is authoritative; `/SC` and `/DC` are consistency checks and missing-value fallbacks; `/CA` requires `NumOfCells`.
- Preserve duplicate values from different physical cells.
- Select one ordered candidate path per physical cell; never union fallback path families.
- Do not derive a missing UL value from DL without a confirmed device-model contract.
- Preserve existing `device_info.freq_point` / `ul_earfcn` CSV and API contracts.
- Do not modify or commit local `AGENTS.md` or `docs/review-report/20260723/`.

---

### Task 1: Regression tests for Issue 150

**Files:**
- Create: `omcgo/internal/device/device_radio_projection_test.go`
- Modify: `omcgo/internal/device/device_info_sync_test.go`

**Interfaces:**
- Consumes: existing `InfoSyncer.SyncFromParameters`.
- Produces: desired `projectRadioFrequencyFields(params, tech, productClass)` behavior.

- [ ] **Step 1: Write failing pure projection tests**

Cover:

```go
// DC: NumOfCells=2, DL has FAPService 1/2/3, UL cell 1/2 have equal values.
// Expected DL "39751,39952", UL "39751,39751".
```

Also cover per-cell RF-over-Common priority, incomplete known count, NR indexed paths,
and unknown-count compatibility.

- [ ] **Step 2: Write a failing InfoSyncer integration test**

Use `stubDeviceParamRepo` and assert the map passed to `UpdateSyncFields` contains:

```go
assert.Equal(t, "39751,39952", fields["freq_point"])
assert.Equal(t, "39751,39751", fields["ul_earfcn"])
```

- [ ] **Step 3: Run tests and verify RED**

Run:

```bash
cd omcgo
go test ./internal/device -run 'Test(ProjectRadioFrequencyFields|InfoSyncer_SyncFromParameters_RadioFrequencyProjection)' -count=1 -v
```

Expected: FAIL because the physical-cell frequency projector does not exist and the
current aggregator outputs three DL values and one deduplicated UL value.

### Task 2: Physical-cell frequency projector

**Files:**
- Create: `omcgo/internal/device/device_radio_projection.go`
- Modify: `omcgo/internal/device/device_info_calc.go`
- Modify: `omcgo/internal/device/device_info_sync.go`

**Interfaces:**
- Consumes: `map[string]string`, `model.Technology`, `productClass string`.
- Produces:

```go
type radioFrequencyProjection struct {
    dlValue    string
    ulValue    string
    dlObserved bool
    ulObserved bool
    complete   bool
    reason     string
}

func projectRadioFrequencyFields(
    params map[string]string,
    tech model.Technology,
    productClass string,
) radioFrequencyProjection
```

- [ ] **Step 1: Generalize the carrier-count resolver name**

Rename `resolveExpectedRFCarrierCount` to `resolveExpectedPhysicalCarrierCount` and
update `CalcRFStatus` without changing its behavior.

- [ ] **Step 2: Implement LTE and NR physical index discovery**

Use exact regular expressions for LTE `FAPService.{i}` and NR
`FAPService.1.CellConfig.{i}` frequency paths. With a known count, return `1..N`;
otherwise return sorted observed indices.

- [ ] **Step 3: Implement per-index candidate selection**

For each physical index, choose the first non-empty candidate in the design order.
Join values without value deduplication. If a known count is incomplete, return an
observed empty projection with a reason so stale persisted values are cleared.

- [ ] **Step 4: Integrate after generic aggregation**

In `InfoSyncer.SyncFromParameters`, apply the projection after
`aggregateInstanceFields`. Override only observed LTE/NR frequency fields and log a
warning for incomplete/inconsistent projection without logging raw values.

- [ ] **Step 5: Run focused tests and verify GREEN**

```bash
cd omcgo
go test ./internal/device -run 'Test(ProjectRadioFrequencyFields|InfoSyncer_SyncFromParameters_RadioFrequencyProjection|CalcRFStatus|AggregateInstanceFields)' -count=1 -v
```

Expected: PASS.

### Task 3: Full local verification and review

**Files:**
- Verify production/test changes only.

**Interfaces:**
- Confirms backend build, full backend tests, and unchanged frontend contract.

- [ ] **Step 1: Format and inspect**

```bash
gofmt -w omcgo/internal/device/device_radio_projection.go \
  omcgo/internal/device/device_radio_projection_test.go \
  omcgo/internal/device/device_info_calc.go \
  omcgo/internal/device/device_info_sync.go \
  omcgo/internal/device/device_info_sync_test.go
git diff --check
```

- [ ] **Step 2: Run backend verification**

```bash
cd omcgo
go build ./...
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend verification**

```bash
cd omcmb
npm run typecheck
```

Expected: PASS.

- [ ] **Step 4: Review the diff**

Verify that no frontend/API/schema changes are present and request a focused code
review against Issue 150 requirements.

### Task 4: Commit, push, MR, and test-server verification

**Files:**
- No additional production files unless verification finds a defect.

**Interfaces:**
- Produces a reviewable Issue 150 MR and test-environment evidence.

- [ ] **Step 1: Commit only Issue 150 files**

Use Conventional Commits:

```bash
git commit -m "fix(device): 按物理小区投影上下行频点"
```

- [ ] **Step 2: Push and create MR**

Push `codex/issue-150-physical-cell-projection` and create an MR targeting `main`
with `Closes #150`, root cause, test evidence, and deployment notes.

- [ ] **Step 3: Inspect test-server deployment**

Confirm the safe deployment mechanism and current version on `172.17.9.239`. Rebuild
only the backend `app` service using the server's documented compose stack.

- [ ] **Step 4: Reproject and verify the real device**

Trigger parameter synchronization for `120200055922C8B0068`, then verify:

- `NumOfCells=2`;
- DL EARFCN contains exactly two positional values (`39751,39952`);
- UL EARFCN contains exactly two positional values when both are reported;
- RF remains `[0/2]`;
- list and detail page agree.

- [ ] **Step 5: Record exact server commands, version, and observed values**

Include any deployment limitation or missing UL raw parameter explicitly; do not
claim test-server success without observed post-deployment evidence.

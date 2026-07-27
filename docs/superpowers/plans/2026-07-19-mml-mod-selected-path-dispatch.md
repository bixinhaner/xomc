# MML MOD Selected Path Dispatch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure a MOD execution sends and reads back only the Paths selected by the user for that execution.

**Architecture:** Keep the existing structured request and frontend selection flow unchanged. In the backend MOD compiler, project command sub-fields onto `Statement.Values` before building `param_refs`, so the SPV and existing MOD readback LST describe the selected Path set. Preserve the legacy text-MML fallback when no value key matches a command sub-field; structured page requests reject unknown Paths before compilation.

**Tech Stack:** Go, `testify`, existing OMC MML console executor.

## Global Constraints

- Do not add or change API fields.
- Do not add missing BLQ `param_mappings`.
- Do not change LST, ADD, or RMV behavior.
- Preserve MOD automatic LST readback.
- Do not commit or deploy until separately requested.

---

### Task 1: Limit MOD command entries to selected values

**Files:**
- Modify: `omcgo/internal/mml/console_executor.go`
- Test: `omcgo/internal/mml/console_executor_test.go`

**Interfaces:**
- Consumes: `Statement.Values map[string]string`, keyed by the selected sub-field `MMLCode`.
- Produces: a MOD command entry whose `param_refs` contains exactly the sub-fields present in `Statement.Values`.

- [x] **Step 1: Write the failing regression test**

Add `TestBuildEntry_MOD_OnlyIncludesSubFieldsWithValues` with three command sub-fields and three parameter refs, but only two entries in `Statement.Values`. Assert that the generated `param_refs` contains exactly those two selected items and that their `ParamCode` and `Tr069Path` values match.

- [x] **Step 2: Verify the red state**

Run:

```bash
cd omcgo
go test ./internal/mml -run TestBuildEntry_MOD_OnlyIncludesSubFieldsWithValues -count=1
```

Expected: FAIL because the current MOD branch includes all command sub-fields in `param_refs`.

- [x] **Step 3: Implement the minimal projection**

In the `MOD` branch of `buildStatementCommandEntry`, filter `subFields` by keys present in `stmt.Values`, then pass only that subset to `buildMODParamRefs`. Preserve the existing legacy behavior when no value key matches a command sub-field.

- [x] **Step 4: Verify the green state**

Run the focused regression test again and expect PASS.

- [x] **Step 5: Run related and full backend verification**

Run:

```bash
cd omcgo
go test ./internal/mml -count=1
go build ./...
go test ./...
```

Expected: all commands exit with status 0.

- [x] **Step 6: Inspect the final diff**

Run:

```bash
git diff --check
git status --short
git diff -- omcgo/internal/mml/console_executor.go omcgo/internal/mml/console_executor_test.go
```

Expected: only the selected-Path MOD regression and its minimal implementation are present, plus this plan document.

# Code Review: MML Phase 1 (API Alignment) + Phase 2 (Execution Chain)

**Date**: 2026-04-14
**Author**: AI Code Reviewer
**Scope**: mml
**Verdict**: PASS

---

## Summary

MML module Phase 1 + Phase 2 implementation: align backend/frontend interfaces, add task scheduling/retry/statistics fields, implement task lifecycle control (start/pause/cancel/delete), and replace hardcoded mock data with real API calls.

---

## Files Reviewed

### Backend (Go)

| File | Status |
|------|--------|
| `omcgo/internal/mml/model.go` | Modified - Added ExecuteType, TaskResult enums; 16 new fields on MMLTask |
| `omcgo/internal/mml/repository.go` | Modified - Added UpdateStatus, Delete to TaskRepository interface |
| `omcgo/internal/mml/pg_repository.go` | Modified - Extended Create/Update/List/scan; added UpdateStatus/Delete; **fixed SQL injection** |
| `omcgo/internal/mml/service.go` | Modified - Task control methods with state transition validation |
| `omcgo/internal/mml/handler.go` | Modified - New task control routes, GetScript handler |
| `omcgo/internal/mml/service_test.go` | Modified - 8 new tests for task control |
| `omcgo/internal/mml/handler_test.go` | Modified - 4 new handler tests |
| `omcgo/migrations/seed/900004_mml_enhance.sql` | New - Migration + seed data |

### Frontend (TypeScript/React)

| File | Status |
|------|--------|
| `omcmb/webcode/src/types/mml.ts` | Modified - Extended MMLTaskStatus, new types |
| `omcmb/webcode/src/services/api/mmlApi.ts` | Modified - Fixed params→parameters bug; new API methods; `||`→`??` for numeric defaults |
| `omcmb/webcode/src/hooks/api/useMML.ts` | Modified - New mutation hooks, polling hook |
| `omcmb/webcode/src/pages/mml/Console/hooks/useCommandSelection.ts` | Modified - Real API via useAllMMLCommands |
| `omcmb/webcode/src/pages/mml/Console/hooks/useDeviceSelection.ts` | Modified - Real API via deviceApi.getList |
| `omcmb/webcode/src/pages/mml/Console/hooks/useCommandExecution.ts` | Modified - Fixed sync→async result handling |
| `omcmb/webcode/src/pages/mml/ScriptTask/index.tsx` | Modified - Full rewrite with real API |
| `omcmb/webcode/src/mock/services/mmlService.ts` | Modified - Updated return types |
| `omcmb/webcode/src/mock/data/mml.ts` | Modified - Updated mock data with new fields |

---

## Findings

### CRITICAL (Fixed)

| # | File | Issue | Fix |
|---|------|-------|-----|
| C1 | `pg_repository.go` | SQL injection via ORDER BY - `sortBy` from user input used directly in `OrderBy()` without sanitization. Applied in 3 List methods (commands, scripts, tasks). | Added `commandAllowedSortColumns`, `scriptAllowedSortColumns`, `taskAllowedSortColumns` whitelist maps following existing pattern from `device_repository.go`. Also validated `sortDir` to only accept "asc"/"desc". |

### WARNING (Fixed)

| # | File | Issue | Fix |
|---|------|-------|-----|
| W1 | `mmlApi.ts:257` | `params` field name should be `parameters` to match backend `ExecuteHTTPRequest` struct tag. | Fixed: `payload.parameters = params` |
| W2 | `useCommandExecution.ts:75` | Was assuming `executeMutation` returns per-device results array, but backend returns `MMLTask`. | Fixed: Now shows "task created" message with task ID |
| W3 | `useCommandSelection.ts` | Used `MOCK_COMMANDS` hardcoded array. | Fixed: Uses `useAllMMLCommands()` hook |
| W4 | `useDeviceSelection.ts` | Used `DEVICE_LIST` hardcoded array. | Fixed: Uses `deviceApi.getList()` API call |
| W5 | `mmlApi.ts` mapBackendTask | `||` operator for numeric defaults (`|| 0`, `|| 60`) would incorrectly fallback when backend returns `0`. | Fixed: Changed to `??` (nullish coalescing) for all numeric and boolean defaults |

### INFO

| # | File | Note |
|---|------|------|
| I1 | `model.go` | New `ExecuteType` and `TaskResult` types follow existing `TaskStatus` pattern consistently |
| I2 | `service.go` | State transition validation uses sentinel errors (`ErrInvalidTransition`, `ErrCannotDeleteRunning`) - good pattern |
| I3 | `pg_repository.go` | `UpdateStatus` correctly sets `started_at`/`finished_at` based on target state |
| I4 | All test files | 30 tests pass, covering success and failure paths for task control |

---

## Test Results

```
=== Backend ===
$ cd omcgo && go build ./internal/mml/...
BUILD OK

$ cd omcgo && go test ./internal/mml/...
ok  github.com/omcgo/omcgo/internal/mml  0.729s

=== Frontend ===
$ cd omcmb/webcode && npx tsc --noEmit
(no errors)
```

---

## Verdict: PASS

All CRITICAL and WARNING issues have been fixed. The implementation follows existing project patterns (handler→service→repository, allowedSortColumns whitelist, BackendXxx→mapBackendXxx mapping).

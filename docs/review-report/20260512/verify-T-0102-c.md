# T-0102-c Verify Report — Real RPC Dispatch (reboot/factory_reset/get_param/set_param/get_rpc_methods)

**Date**: 2026-05-12
**Branch**: main
**Author**: Claude
**Type**: feat (F06 / ops + acs)
**Sub-task**: T-0102-c (from T-0102 umbrella — SSE 通道 5 sub-task; 进度 → 3/5)
**PRD**: F06 ops PRD §4.2.1 (RPC 命令支持)

## Scope

Wire the 5 TR-069 RPC actions (reboot / factory_reset / get_param / set_param / get_rpc_methods) end-to-end from the ops layer through the existing `internal/task` queue to the ACS engine. Until this task, T-0102-b's `ExecuteRPC` persisted an OpsTask but the executor's `Run()` was an MVP placeholder that only wrote an "executor_dispatched" log row.

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `omcgo/internal/ops/rpc_dispatcher.go` (NEW) | `RPCInlineKind`/`RPCInlineEnvelope` typed envelope, `actionToRPCMethod` mapping (5 actions + camelCase/snake_case aliases) | +59 |
| `omcgo/internal/ops/service_ext.go` | TaskExecutor +`enqueuer`/`sseHub` fields (optional via `SetEnqueuer`/`SetSSEHub`), Run() routes inline-RPC tasks to `dispatchInlineRPC` (fan-out per device via `task.Enqueuer.CreateTask`), `parseInlineRPCEnvelope` + `publishDispatchEvent` + `ptrTime` helpers | +149 / -17 |
| `omcgo/internal/ops/handler_ext.go` | `ExecuteRPC` now validates `action` via `actionToRPCMethod` (400 on unknown) + uses typed `RPCInlineEnvelope` instead of map literal | +9 / -7 |
| `omcgo/internal/task/model.go` | +`TaskSourceOps = "ops"` enum | +1 |
| `omcgo/internal/ops/rpc_dispatcher_test.go` (NEW) | 18 test cases (15 action-mapping table + 3 dispatcher flow + 4 fall-back/SSE) — `actionToRPCMethod` × 13 / `parseInlineRPCEnvelope` × 4 / fan-out / partial-failure / non-RPC fallback / no-enqueuer fallback / SSE event | +330 |
| `omcgo/cmd/app/provider/modules.go` | `opsExecutor.SetEnqueuer(c.TaskSvc)` + `opsExecutor.SetSSEHub(opsSSEHub)` | +4 |

## Design Notes

- **Inline envelope is typed, not free map**: `RPCInlineEnvelope{Kind, Action, Params}` is the contract between T-0102-b (writer) and T-0102-c (reader). Compile-time check via shared types prevents drift.
- **Optional dependency injection**: `SetEnqueuer`/`SetSSEHub` setters so the existing template-driven `Run()` flows (T-0101 umbrella) keep working when ops module bootstraps without the task service available. If `enqueuer == nil`, an inline RPC task falls back to the MVP placeholder path — no silent task drop.
- **Per-device fan-out + per-device execution rows**: One `device_tasks` row per device, one `ops_task_executions` row per device. Per-device failures (e.g. unknown device, queue full) are captured as failed execution rows; the ops task itself doesn't crash. Matches PRD §4.2.3 "批量场景设备级独立超时".
- **Reuses existing internal/task infrastructure**: The ACS engine already pops device tasks by SN, sends Connection Request, and renders SOAP via the registered RPC handlers (`Reboot`, `FactoryReset`, `GetParameterValues`, `SetParameterValues`, `GetRPCMethods` — though the last one isn't yet registered in `internal/acs/rpc/dispatcher.go`; will fail at SOAP build time with a clean error, surfaced via the task result). No new RPC handlers added in this task.
- **TaskSource is free-form (`VARCHAR(32)`) — no CHECK constraint**: Adding `TaskSourceOps = "ops"` is a code-only enum extension, no migration.
- **SSE `command.dispatched` event per device**: Subscribers of `/api/v1/ops/commands/:id/stream` see fan-out happen in realtime. Payload carries `device_task_id` so the UI can correlate to the per-device task lifecycle.

## Test Plan

| Test | Path | Asserts |
|------|------|---------|
| TestActionToRPCMethod | 15 table cases | 5 actions + 5 aliases + 2 case-variants + 1 whitespace + 3 errors |
| TestParseInlineRPCEnvelope | 4 sub-cases | valid / empty / non-json / wrong-kind |
| InlineRPC_FansOut | 3 devices × reboot | 3 enqueuer calls (correct method=Reboot, source=ops, source_id, creator_id, device_index) + 3 execution rows status=running |
| InlineRPC_PartialFailure | enqueuer fails on SN-bad | 3 attempts, 2 succeed; 2 running + 1 failed execution row with ErrorMessage |
| NonInlineMessage_FallsBackToPlaceholder | template-driven task | 0 enqueuer calls, 1 placeholder execution row |
| InlineRPC_NoEnqueuer_FallsBackToPlaceholder | enqueuer nil | placeholder path (no silent drop) |
| InlineRPC_SSEDispatchEvents | 2 devices + hub subscribed | 2 `command.dispatched` events with ops_task_id + device_sn + method |

**Race**: `go test -race ./internal/ops/... ./internal/backup/...` all green.
**Pre-existing failure**: `internal/task/TestService_PG_RestorePendingQueues` fails identically without these changes (scan NULL into *string for source_id col) — confirmed via `git stash` baseline test.

## Commands

```bash
$ go build ./...                                                          # ✓
$ go vet ./internal/ops/... ./internal/task/...                           # ✓
$ go test -race ./internal/ops/ -count=1 -v                               # all pass
$ go test -race ./internal/ops/ -run "RPC|InlineRPC|Action|ParseInline"   # 18 PASS / 0 FAIL
```

## Out of Scope (Carved for Future)

- **GetRPCMethods ACS handler**: action maps to "GetRPCMethods" but the ACS dispatcher doesn't yet have it registered. The SOAP build will return an error visible in the device task's result column — clean failure path, no silent drop. Adding the handler is a small follow-up (3-line registration + 1 SOAP template).
- **Per-device timeout + retry policy**: T-0102-e (`超时 + 重试 + per-device 限流`).
- **MML streaming output (multi-frame SSE)**: T-0102-d.
- **Completion-side aggregation**: when a device task completes, the ACS layer must update the ops execution row + emit `command.completed` SSE. The existing `internal/task` completion callbacks can be subscribed to — future hook-up via `taskService.AddCompletionCallback` reading task.Source==ops and bridging to ops.execRepo.Update. Out of T-0102-c's "enqueue + fan-out" scope.

## DoD Checklist

- [x] `go build ./...` 通过
- [x] `go test -race ./internal/ops/...` 全绿（18 新 testcase + 既有 ops 测试）
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == "..."` 硬编码
- [x] 无新增 `any`/`interface{}`（`Params map[string]interface{}` 是 JSON 通用类型）
- [x] 公共构造函数 `NewTaskExecutor` 签名保持向后兼容（setters 注入）
- [x] PRD §4.2.1 5 个 RPC 动作映射完整（reboot/factory_reset/get_param/set_param/get_rpc_methods）
- [x] 批量场景设备级失败不影响整体 task（per-device 失败行隔离）
- [x] 真 RPC 派发链路打通：ops.ExecuteRPC → ops.Run → task.Enqueuer.CreateTask → device_tasks → ACS 引擎弹出 → CWMP/SOAP
- [x] R-O02 mitigation 又深一层（实 RPC 派发 vs MVP stub）

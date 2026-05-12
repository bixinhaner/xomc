# T-0102-b Verify Report — POST /ops/commands/rpc Enqueue Endpoint

**Date**: 2026-05-12
**Branch**: main
**Author**: Claude
**Type**: feat (F06 / ops)
**Sub-task**: T-0102-b (from T-0102 umbrella — SSE 通道 5 sub-task; 进度 → 2/5)
**PRD**: F06 ops PRD §7.2 + §4.2.2

## Scope

Replace the MVP stub `ExecuteRPC` handler with a real enqueue path:

- Persist an `OpsTask` via the consumer-driven `TaskCreator` interface
- Risk-classify by action (existing `classifyRiskForAction`) + device count
  (existing `ApprovalService.EvaluateRiskLevel`) to compute overall risk
- L3 (dangerous) tasks → `approval_state=pending`, gated (no auto-dispatch)
- L1/L2 (safe/cautious) tasks → `approval_state=not_required`, fire-and-forget
  goroutine calls `executor.Run`
- Inline RPC envelope `{kind, action, params}` stashed in `OpsTask.Message`
  for T-0102-c's real dispatcher to consume back
- Audit log + SSE `command.enqueued` event on the per-task channel

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `omcgo/internal/ops/handler_ext.go` | +`TaskCreator` interface, +`taskCreator` field, rewrite `ExecuteRPC` body, +`runRPCAsync`/`buildRPCTaskName`/`mustJSON` helpers | +132 / -16 |
| `omcgo/internal/ops/handler_ext_test.go` (NEW) | 7 test cases for ExecuteRPC + stubs (mockTaskCreator/stubExecutor/noopAuditRepo) | +362 |
| `omcgo/cmd/app/provider/modules.go` | Wire `opsSvc` into `NewExtHandler` as the `TaskCreator` | +2 / -1 |

## Design Notes

- **Consumer-driven `TaskCreator` interface**: Only 1 method (`CreateTask`). Defined in the handler package where it's consumed, satisfied by `*Service` (which has `CreateTask`). Keeps the test from needing the full `Service` + repo chain.
- **Inline command spec lives in `OpsTask.Message`**: `ops_tasks` schema has no `inline_command` JSONB column today. Rather than adding a migration mid-iteration, the RPC envelope JSON-marshals into the existing `message TEXT` column. T-0102-c will read this back to feed the real dispatcher. If/when we add a typed column, the read side moves but no data migration is needed.
- **Fire-and-forget executor.Run**: HTTP request context is cancelled at 202; the executor needs a fresh ctx with a 60s timeout (matches PRD §4.2.3 "单命令默认 60s 超时"). Errors logged; task status field carries truth.
- **Approval gating happens at enqueue**: For L3 tasks, the task is created with `approval_state=pending` and is NOT auto-run. The existing `POST /tasks/:id/approve` endpoint (T-0101-d) advances it.
- **Risk evaluation is 2-dimensional**: `classifyRiskForAction` (action-driven) feeds `ApprovalService.EvaluateRiskLevel(deviceCount, actionRisk)` so a single `set_param` is cautious but `set_param` over 51 devices is dangerous (PRD §4.2.2 "批量 >50").

## Test Plan

| Test | Path | Asserts |
|------|------|---------|
| V1 SingleDevice_SafeAutoDispatch | get_param × 1 device | 202 / risk=safe / approval=false / DB-assigned task_id / Message contains envelope / executor.Run fires within 1s |
| V2 FactoryReset_RequiresApproval | factory_reset × 1 device | risk=dangerous / approval=true / executor NOT invoked (100ms wait + zero count) |
| V3 BatchReboot_RiskEscalation | reboot × 11 devices | risk=cautious (count>10) / approval=false / DeviceSNs JSON has 11 entries / TaskName "rpc:reboot on 11 devices" |
| V4 LargeBatch_DangerousEscalation | set_param × 51 devices | risk=dangerous (count>50) / approval=true / executor NOT invoked |
| V5 MissingDevice_400 | no device_sn AND no device_sns | 400 / TaskCreator NOT invoked |
| V6 CreateTaskError_500 | CreateTask returns db-down err | 500 / executor NOT invoked |
| V7 SSEEnqueuedEvent | subscribe before POST | SSE `command.enqueued` arrives within 1s on `command:<task_id>` channel with task_id + action |

**Race**: `go test -race ./internal/ops/ -count=1` passes (2.353s — all ops tests including these 7 new ones).

## Commands

```bash
$ go build ./...                                                   # ✓
$ go vet ./internal/ops/...                                        # ✓
$ go test -race ./internal/ops/ -run TestExecuteRPC -v -count=1    # 7 PASS / 0 FAIL
$ go test -race ./internal/ops/ -count=1                           # ok 2.353s (full ops package green)
```

## Out of Scope (Carved for T-0102-c)

- **Real RPC dispatch to ACS**: `executor.Run` is still the T-0101 MVP placeholder (records "executor_dispatched" placeholder execution + audits). T-0102-c will replace it with a real RPC handler that reads the inline envelope from `OpsTask.Message`, calls into the ACS engine, and emits per-step `command.dispatched`/`command.completed` SSE events.
- **MML streaming**: out of T-0102 RPC scope; tracked as T-0102-d (MML 流式输出).
- **Per-device timeout + retry**: T-0102-e.
- **Dedicated `inline_command JSONB` column**: future schema cleanup once the pattern is settled. Currently `OpsTask.Message` is the carrier; if a customer's freeform message conflicts, migration is straightforward.

## DoD Checklist

- [x] `go build ./...` 通过
- [x] `go test -race ./internal/ops/...` 全绿（7 新 testcase + 既有 ops 测试）
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == "..."` 硬编码
- [x] 无新增 `any`/`interface{}`（已有 `params map[string]interface{}` 是 JSON 通用类型，不在禁项范围）
- [x] HTTP 错误码契约：400 (validation) / 500 (persistence) / 202 (success)
- [x] Approval gating 反退化：L3 dangerous 路径绝不 auto-dispatch
- [x] 公共构造函数 `NewExtHandler` 已加 `taskCreator TaskCreator` 参数；DI 同步更新
- [x] PRD §4.2.2 风险等级 (L1/L2/L3) 行为完整对齐
- [x] PRD §7.2 endpoint 契约（POST → 入队 → 返 task_id）完整兑现

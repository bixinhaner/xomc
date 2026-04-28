# T-0050 Verification Report — provision Service Facade

- **Task**: Wave 2 / Block B.4 (T-0050) — `internal/provision/` thin service facade
- **Date**: 2026-04-28
- **Worktree**: `agent-a3cbefc3` (branch `worktree-agent-a3cbefc3`)
- **Charter pass criterion**: `ls omcgo/internal/provision/*service*.go ≥ 1`

## 1. Result Summary

| Item | Result |
|------|--------|
| `omcgo/internal/provision/service.go` | Created |
| `omcgo/internal/provision/service_test.go` | Created (≥ 3 tests) |
| `go build ./...` | PASS (no output) |
| `go test -race -count=1 ./internal/provision/...` | PASS (`ok` 1.767 s) |
| `ls .../*service*.go` | 2 files (≥ 1, criterion met) |

## 2. Files Touched

### New

- `omcgo/internal/provision/service.go` — Service facade (~250 lines).
- `omcgo/internal/provision/service_test.go` — 13 sub-tests across 9 top-level
  test functions (table-/sub-test style).

### Untouched

- All existing files in `internal/provision/` (engine.go, orchestrator.go,
  state_machine.go, matcher.go, sync.go, model_upload.go,
  discovery_repository.go, pg_repository.go, repository.go, handler.go,
  sync_plan_store.go, model.go).
- `cmd/app/provider/modules.go`, `cmd/app/provider/provision.go` — left as-is
  per Wave 2 path-mutex isolation rules; main session will integrate the
  facade.

## 3. Relationship to Existing Engine / Orchestrator / State Machine

The `internal/provision/` package already had a service-shaped surface split
across multiple collaborators. The new `Service` is a **thin facade** that
unifies the high-level entry points without reimplementing any logic.

| Facade method | Delegates to |
|---------------|-------------|
| `DiscoverDevice(ctx, sn)` | `device.DeviceService.GetBySerialNumber` (via internal `deviceLookup` adapter) |
| `TriggerProvisioning(ctx, deviceID)` | `device.DeviceService.GetDevice` → builds `bootstrapEvent` → `ProvisioningEngine.HandleBootstrap` |
| `GetTaskState(ctx, deviceID)` | `ProvisioningTaskRepository.GetByDeviceID` |
| `GetTask(ctx, taskID)` | `ProvisioningTaskRepository.GetByID` |
| `ListTasks(ctx, filter)` | `ProvisioningTaskRepository.List` |
| `RetryFailedTask(ctx, taskID)` | `ProvisioningTaskRepository.{GetByID, Create}` (mirrors `handler.Retry`) |
| `HandleRPCResult(...)` | `ProvisioningEngine.HandleRPCResult` |
| `IsTerminalState(state)` | package-level `IsTerminal` |
| `ValidateStateTransition(curr, target)` | package-level `ValidateTransition` |

### Why a facade rather than a rewrite

`engine.go` / `orchestrator.go` / `state_machine.go` are battle-tested and
exhaustively unit-tested (engine_test.go alone is 1079 lines, plus
orchestrator_test.go, state_machine_test.go, matcher_test.go). Replacing
them risks regressions in the four-path bootstrap flow (template / auto
sync / model upload / fail-back) and the GPN/GPV two-phase sync. The facade
adds a single import-stable entry point for downstream callers (router DI,
future RPC bridges, mock-friendly tests) while preserving every existing
collaborator.

### Internal abstractions added

- `engineHandle` (interface, unexported) — minimal slice of
  `*ProvisioningEngine` so the facade is testable without spinning up the
  full engine graph.
- `deviceLookup` (interface, unexported) — minimal slice of
  `*device.DeviceService` for the same reason.
- `deviceForProvision` (struct, unexported) — narrow projection of
  `model.Device` containing only the fields needed to build a
  `bootstrapEvent`. This keeps the lookup interface decoupled from the
  full `model` surface area.
- `realDeviceLookup` (struct, unexported) — adapter from
  `*device.DeviceService` to `deviceLookup`.
- Test-only `newServiceWithMocks` — internal constructor for the smaller
  interfaces; not exported.

### Errors

Three sentinel errors are exported for callers to discriminate without
parsing strings:

- `ErrDeviceNotFound`
- `ErrTaskNotFound`
- `ErrTaskNotRetryable`

## 4. Test Coverage

```
ok  	github.com/omcgo/omcgo/internal/provision	1.767s  (full pkg, race)
ok  	github.com/omcgo/omcgo/internal/provision	0.676s	coverage: 6.7%  (-run service tests)
```

The 6.7 % coverage figure is the share of statements covered when running
**only** the service tests; the package’s overall coverage including the
existing engine/orchestrator/state-machine suites is much higher. The new
test file adds 13 sub-tests covering:

- `DiscoverDevice`: found / not-found / empty-input / missing-lookup paths
- `TriggerProvisioning`: success / device-missing / engine-not-configured
- `GetTaskState`: returns state / not-found
- `GetTask`: found / not-found
- `ListTasks`: pass-through with totals
- `RetryFailedTask`: success / non-failed-state / missing / repo Create error
- `HandleRPCResult`: pass-through invocation count
- `IsTerminalState` + `ValidateStateTransition`: allow / reject paths
- `NewService` with all-nil deps: graceful errors, no panics

All tests use the existing `mockTaskRepo` (function-field pattern from
`engine_test.go`) plus two new lightweight stubs (`stubEngine`,
`stubDeviceLookup`).

## 5. Charter / Backlog Compliance

- W2.B.4 pass criterion `ls omcgo/internal/provision/*service*.go ≥ 1` —
  **MET** (2 files: `service.go`, `service_test.go`).
- No edits to `cmd/app/provider/{modules,provision}.go` — main session
  reserved for DI wiring.
- No edits to other modules (task / events / core / mr / syslog / interop).
- No git commit, push, pull, or backlog edits performed by this agent.
- No new `go.mod` dependencies introduced.

## 6. Self-Verification Commands

```bash
cd <worktree>/omcgo
go build ./...                                                   # PASS
go test -race -count=1 ./internal/provision/...                  # PASS
ls internal/provision/*service*.go                               # 2 files
```

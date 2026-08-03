# Redis Terminal Task Compaction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bound the high-rate ACS terminal-task Redis working set without weakening late-response fencing or durable recovery.

**Architecture:** Compact a task only at the existing atomic transition-finalization point after PostgreSQL, cross-slot cleanup, and event publication are acknowledged. Redis retains an ID/status tombstone with the existing TTL; complete terminal details come from PostgreSQL.

**Tech Stack:** Go, go-redis, Redis Lua scripts, miniredis, PostgreSQL task repository.

## Global Constraints

- Preserve the 10-minute minimum and 15-minute default terminal fence TTL.
- Never compact a transition while any compensation acknowledgement remains pending.
- No data migration and no non-OMC deletion.
- Use test-first changes and retain existing Redis fail-safe semantics.

---

### Task 1: Atomic terminal tombstone

**Files:**
- Modify: `omcgo/internal/task/redis_queue.go`
- Test: `omcgo/internal/task/redis_queue_terminal_ttl_test.go`
- Test: `omcgo/internal/task/transition_cas_test.go`

**Interfaces:**
- Produces: `RedisTaskQueue.GetByID` returns `&Task{ID: taskID, Status: terminalStatus}` when a finalized tombstone has no `data` field.
- Produces: `finalizeTaskTransitionScript` removes the `data` field only for a fully acknowledged terminal transition.

- [ ] **Step 1: Write failing tests**

Add assertions that a finalized terminal task has no Redis `data` field, has only the expected terminal status when read through `GetByID`, retains its TTL, and still rejects a late competing transition. Add a compensation test proving `data` remains while `event_pending=1` and disappears only after event acknowledgement.

- [ ] **Step 2: Verify RED**

Run: `go test ./omcgo/internal/task -run 'TestRedisTaskQueue_(TerminalTransitionsUseShortTTLAndCleanIndexes|TerminalTransitionIsIdempotentForLateDuplicate)|TestRedisTaskQueue_Transition' -count=1`

Expected: FAIL because finalized terminal hashes still contain full `data`.

- [ ] **Step 3: Implement minimal atomic compaction**

Record a `transition_terminal` flag in `taskTransitionCASScript`. In `finalizeTaskTransitionScript`, when all acknowledgement flags are zero, delete `data` only if that flag is `1`, remove all transition metadata, and apply the existing `transition_ttl_ms`. Update `GetByID` to reconstruct a terminal tombstone from the `status` field when `data` is absent; missing non-terminal `data` remains an absent task.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./omcgo/internal/task -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Run: `git add omcgo/internal/task/redis_queue.go omcgo/internal/task/redis_queue_terminal_ttl_test.go omcgo/internal/task/transition_cas_test.go && git commit -m 'fix(task): 压缩Redis终态任务墓碑'`

### Task 2: Durable terminal detail routing and capacity guard

**Files:**
- Modify: `omcgo/internal/task/service.go`
- Test: `omcgo/internal/task/service_pg_test.go`
- Test: `omcgo/internal/task/redis_queue_terminal_ttl_test.go`

**Interfaces:**
- Consumes: terminal tombstone returned by `RedisTaskQueue.GetByID`.
- Produces: `TaskService.GetTask` returns the PostgreSQL task for every terminal Redis state.

- [ ] **Step 1: Write failing tests**

Create a PostgreSQL-backed service test where Redis contains a compact terminal tombstone and PostgreSQL contains large `Params` and `Result`; assert `GetTask` returns the full durable object. Replace the obsolete 7,410/minute capacity assertion with 40,000/minute, 512-byte tombstones, 15-minute retention, and a 512 MiB ceiling.

- [ ] **Step 2: Verify RED**

Run: `go test ./omcgo/internal/task -run 'TestTaskService_GetTask.*Terminal|TestTerminalTaskCapacityModelAtObservedRate' -count=1`

Expected: FAIL because the service currently returns the Redis tombstone and the old model does not express the production rate.

- [ ] **Step 3: Implement minimal routing**

Change `TaskService.GetTask` so a non-terminal Redis task is returned directly, while a terminal Redis task is fetched from `s.repo.GetByID`; if the durable row is unexpectedly absent, return the tombstone rather than losing the terminal fence signal.

- [ ] **Step 4: Verify GREEN and regressions**

Run: `go test ./omcgo/internal/task -count=1`

Run: `go test ./... -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Run: `git add omcgo/internal/task/service.go omcgo/internal/task/service_pg_test.go omcgo/internal/task/redis_queue_terminal_ttl_test.go && git commit -m 'fix(task): 终态详情回源主库'`

### Task 3: Release, deploy, and production verification

**Files:**
- Modify: `docs/superpowers/evidence/2026-08-03-pm-slot-health-minio-headroom.md`

**Interfaces:**
- Consumes: tested task compaction and current release build workflow.
- Produces: deployed version, bounded Redis memory evidence, and MR-ready branch.

- [ ] **Step 1: Run release verification**

Run the repository release gate, full Go suite, shell syntax checks, and package metadata/checksum checks. Expected: all pass with no uncommitted generated changes.

- [ ] **Step 2: Deploy without migration**

Deploy the new offline package to `172.24.224.197`, preserving the explicitly scoped OMC stores during the upgrade. Persist `redis-core` at a 4 GiB container limit and 3 GiB `maxmemory` as online protection; do not touch non-OMC data.

- [ ] **Step 3: Verify business and performance**

Verify health checks, ACS 503/admission rejection/session metrics, native NATS/Redis queues, dead letters, task transition backlog trend, Redis memory and OOM errors, PostgreSQL/TimescaleDB locks and slow queries, host CPU/memory/iowait, K900010006/K900010076/K900010040/K900010041, the 12-minute hourly publication gate, and Dashboard hourly/daily/weekly requests including partial coverage.

- [ ] **Step 4: Update evidence and commit**

Record exact version, checksum, test output, production measurements, and request timings in the evidence document, then commit with `docs: 补充Redis终态压缩验收证据`.

- [ ] **Step 5: Submit and merge MR**

Push `codex/pm-kpi-closure-reconcile`, create a ready GitLab MR, verify pipeline and review status, merge it, and confirm `origin/main` contains the merged commits.

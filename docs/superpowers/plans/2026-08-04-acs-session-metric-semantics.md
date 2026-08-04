# ACS Session Metric Semantics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `acs_active_sessions` a deprecated compatibility alias of the authoritative global admission count and expose the process-local five-minute tracking set as `acs_local_tracked_sessions`.

**Architecture:** Keep the admission controller and session lifecycle unchanged. A single `AdmissionController.Current` sample refreshes both global gauges, while local track/untrack/reap operations update only a new diagnostic gauge. Current operational scripts and documents use the authoritative global metric for concurrency and capacity decisions.

**Tech Stack:** Go, Prometheus client_golang, testify, Bash, Markdown

## Global Constraints

- Do not change the 30,000 global session limit, rate limits, session TTL, Redis structures, or ACS request flow.
- Keep the process-local tracking set and its five-minute reaping behavior.
- Keep `acs_active_sessions` exported for compatibility, with a deprecated HELP contract.
- One admission sample must set both `acs_global_active_sessions` and `acs_active_sessions` to the same value.
- Current dashboards, alerts, health checks, load tests, and operational docs must use `acs_global_active_sessions` for concurrency and capacity decisions.
- Do not rewrite archived or historical design records.

---

### Task 1: Lock the three-metric contract

**Files:**
- Modify: `omcgo/internal/acs/metrics_test.go`
- Modify: `omcgo/internal/acs/handler_test.go`
- Modify: `omcgo/internal/acs/metrics.go`
- Modify: `omcgo/internal/acs/handler.go`

**Interfaces:**
- Consumes: `AdmissionController.Current(context.Context) int64`, `Handler.localActiveSessions`
- Produces: `ACSMetrics.LocalTrackedSessions prometheus.Gauge`; `acs_active_sessions`, `acs_global_active_sessions`, and `acs_local_tracked_sessions` collectors

- [ ] **Step 1: Write failing metric construction tests**

Add assertions that all three gauges are registered and gathered with these contracts:

```go
assert.Equal(t, "Deprecated compatibility alias for acs_global_active_sessions; current number of globally admitted TR069 sessions", metricHelp(t, reg, "acs_active_sessions"))
assert.Equal(t, "Current number of globally admitted TR069 sessions from the shared admission controller", metricHelp(t, reg, "acs_global_active_sessions"))
assert.Equal(t, "Current number of session IDs retained by this ACS process for up to five minutes; not real-time concurrency", metricHelp(t, reg, "acs_local_tracked_sessions"))
```

- [ ] **Step 2: Run the construction tests and verify RED**

Run: `env GOCACHE=/tmp/goomc-go-cache go test ./internal/acs -run 'TestNewACSMetrics' -count=1`

Expected: FAIL because `LocalTrackedSessions` and `acs_local_tracked_sessions` do not exist and the old HELP text describes local tracking.

- [ ] **Step 3: Write failing refresh and local lifecycle tests**

Update the refresh test to assert both global gauges equal the same authoritative admission value. Update local track, duplicate-completion, cross-instance completion, and reaper tests to assert only `LocalTrackedSessions` changes; seed both global gauges with sentinel values and prove local lifecycle operations do not change them.

- [ ] **Step 4: Run the lifecycle tests and verify RED**

Run: `env GOCACHE=/tmp/goomc-go-cache go test ./internal/acs -run 'Test(CompleteSessionOnlyDecrementsSessionsTrackedByThisProcess|ReapLocalActiveSessionsRemovesCrossInstanceOrphans|RefreshGlobalActiveSessionsUsesAdmissionSourceOfTruth)' -count=1`

Expected: FAIL because local tracking still changes `ActiveSessions` and refresh does not update the compatibility alias.

- [ ] **Step 5: Implement the minimal metric wiring**

Add `LocalTrackedSessions` to `ACSMetrics`, construct/register `acs_local_tracked_sessions`, mark the old gauge HELP deprecated, and update `refreshGlobalActiveSessions` to sample `Current` once before setting both global gauges. Make track/untrack update only `LocalTrackedSessions`, and update the nearby field comment to describe local tracking accurately.

- [ ] **Step 6: Run focused tests and verify GREEN**

Run: `env GOCACHE=/tmp/goomc-go-cache go test ./internal/acs -run 'Test(NewACSMetrics|CompleteSessionOnlyDecrementsSessionsTrackedByThisProcess|ReapLocalActiveSessionsRemovesCrossInstanceOrphans|RefreshGlobalActiveSessionsUsesAdmissionSourceOfTruth)' -count=1`

Expected: PASS.

### Task 2: Migrate active operational consumers

**Files:**
- Modify: `omcgo/scripts/health-check.sh`
- Modify: `omcgo/scripts/loadtest-benchmark.sh`
- Modify: `docs/operations/OMC可观测性使用手册.md`
- Modify: `omcgo/docs/acs-stress-test-readiness.md`

**Interfaces:**
- Consumes: Prometheus sample `acs_global_active_sessions`
- Produces: Current health, load-test, observability, and stress-test guidance that treats the global metric as authoritative

- [ ] **Step 1: Verify the current consumers fail the new contract**

Run:

```bash
rg -n 'acs_active_sessions' omcgo/scripts/health-check.sh omcgo/scripts/loadtest-benchmark.sh docs/operations/OMC可观测性使用手册.md omcgo/docs/acs-stress-test-readiness.md
```

Expected: matches in all four current consumers.

- [ ] **Step 2: Update scripts and current documentation**

Change health and benchmark metric selectors to exact `acs_global_active_sessions` samples. Document `acs_global_active_sessions` as the concurrency/capacity source, retain `acs_active_sessions` only in a compatibility note, and describe `acs_local_tracked_sessions` only as a local diagnostic that may exceed real-time concurrency.

- [ ] **Step 3: Verify current consumers use the authoritative metric**

Run:

```bash
rg -n 'acs_global_active_sessions' omcgo/scripts/health-check.sh omcgo/scripts/loadtest-benchmark.sh docs/operations/OMC可观测性使用手册.md omcgo/docs/acs-stress-test-readiness.md
rg -n 'acs_active_sessions' omcgo/scripts/health-check.sh omcgo/scripts/loadtest-benchmark.sh
```

Expected: the first command matches all four files; the second command has no matches.

### Task 3: Verify the complete change

**Files:**
- Verify only: all files from Tasks 1 and 2

**Interfaces:**
- Consumes: completed implementation and operational migration
- Produces: fresh test, formatting, static-contract, and diff evidence

- [ ] **Step 1: Format and inspect the diff**

Run: `gofmt -w internal/acs/metrics.go internal/acs/metrics_test.go internal/acs/handler.go internal/acs/handler_test.go`

Run: `git diff --check && git diff --stat && git diff`

- [ ] **Step 2: Run ACS package tests**

Run: `env GOCACHE=/tmp/goomc-go-cache go test ./internal/acs -count=1`

Expected: PASS.

- [ ] **Step 3: Run the complete Go test suite**

Run: `env GOCACHE=/tmp/goomc-go-cache go test ./... -count=1`

Expected: PASS.

- [ ] **Step 4: Recheck metric migration scope**

Run:

```bash
rg -n 'acs_active_sessions' --glob '!docs/archive/**' --glob '!docs/superpowers/**' --glob '!docs/perf/2026-07-18-kpi-overload-control-design.md' --glob '!omcgo/docs/go-zero-design/**' --glob '!omcgo/docs/phase4-analysis-report.md' --glob '!omcgo/docs/development-plan.md' --glob '!omcgo/docs/design/**' --glob '!omcgo/docs/detailed-design/**' --glob '!omcgo/docs/operations/deployment-guide.md' .
```

Expected: only the compatibility metric declaration and explicit compatibility documentation remain; no active script uses it for capacity decisions.

- [ ] **Step 5: Review requirements and commit**

Compare the final diff to every design acceptance criterion, then commit with:

```bash
git add omcgo/internal/acs/metrics.go omcgo/internal/acs/metrics_test.go omcgo/internal/acs/handler.go omcgo/internal/acs/handler_test.go omcgo/scripts/health-check.sh omcgo/scripts/loadtest-benchmark.sh docs/operations/OMC可观测性使用手册.md omcgo/docs/acs-stress-test-readiness.md docs/superpowers/plans/2026-08-04-acs-session-metric-semantics.md
git commit -m "fix(acs): 修正会话指标语义"
```

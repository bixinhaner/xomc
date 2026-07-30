# UECount Delivery Reliability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent online devices from permanently losing the hourly UECount GPV during high-concurrency CWMP session churn, and expose every global ACS admission rejection.

**Architecture:** Keep the bounded asynchronous UECount scheduler and shared task state machine. Give only `UECountPolicy:GPV` tasks a 12-minute bounded delivery window, ten retries, and a 30-second minimum retry interval; add a low-cardinality ACS admission counter and Prometheus alert without changing other task types.

**Tech Stack:** Go, Prometheus, PostgreSQL/Redis-backed device tasks, shell release tests.

## Global Constraints

- Inform processing must remain independent of PostgreSQL, Redis, and path translation latency.
- Only UECount system tasks receive the enlarged retry budget.
- Retry lifetime is bounded to 12 minutes.
- Global admission rejection metrics must contain no device identity.

---

### Task 1: Bound and strengthen UECount delivery

**Files:**
- Modify: `omcgo/internal/acs/ue_count_policy.go`
- Test: `omcgo/internal/acs/ue_count_policy_test.go`

**Interfaces:**
- Consumes: `task.CreateTaskRequest`.
- Produces: UECount requests with `MaxRetries=10`, `RetryIntervalSeconds=30`, and `ExpiresIn=720`.

- [x] **Step 1: Add failing request-contract assertions**

```go
require.NotNil(t, req.MaxRetries)
assert.Equal(t, 10, *req.MaxRetries)
assert.Equal(t, 30, req.RetryIntervalSeconds)
assert.Equal(t, 12*60, req.ExpiresIn)
```

- [x] **Step 2: Run the focused test and confirm failure**

Run: `go test ./internal/acs -run TestUECountPolicy_ProcessCreatesDirectGPVForSupportedPaths -count=1`

Expected: FAIL because the current request uses task defaults.

- [x] **Step 3: Add the scoped constants and request fields**

```go
const (
    defaultUECountTaskMaxRetries    = 10
    defaultUECountTaskRetryInterval = 30
    defaultUECountTaskExpiresIn     = 12 * 60
)
maxRetries := defaultUECountTaskMaxRetries
```

Set the three fields only in `UECountPolicy.process`.

- [x] **Step 4: Run focused and package tests**

Run: `go test ./internal/acs -count=1`

Expected: PASS.

### Task 2: Observe and alert on ACS global admission 503

**Files:**
- Modify: `omcgo/internal/acs/metrics.go`
- Modify: `omcgo/internal/acs/handler.go`
- Modify: `omcgo/internal/acs/metrics_test.go`
- Modify: `omcgo/internal/acs/handler_test.go`
- Modify: `deployments/monitoring/alerts/runtime-alerts.yml`
- Modify: `deployments/monitoring/tests/promql-probes.txt`
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`

**Interfaces:**
- Produces: counter `acs_admission_rejected_total`.
- Produces: alert `ACSGlobalAdmissionRejected`.

- [x] **Step 1: Add failing metric and handler assertions**

Assert `AdmissionRejected` is non-nil and that one denied Inform increments it to one.

- [x] **Step 2: Add the counter and increment it at the rejection branch**

```go
AdmissionRejected prometheus.Counter
```

Increment it immediately before returning HTTP 503 when global admission acquisition fails.

- [x] **Step 3: Add the bounded alert**

```yaml
- alert: ACSGlobalAdmissionRejected
  expr: increase(acs_admission_rejected_total[5m]) > 0
  for: 1m
```

- [x] **Step 4: Run all validation**

Run: `go build ./... && go test ./...`

Run: `bash deployments/release/bundle/deploy/storage-compose_test.sh`

Expected: all pass.

### Task 3: Submit, deploy, and validate the next hourly cohort

**Files:**
- No source changes expected.

**Interfaces:**
- Produces: merged MR, immutable release, and live evidence from the next UECount cycle.

- [ ] **Step 1: Commit, push, create, and merge the MR**

Commit message: `fix: 提升UECount任务交付可靠性`

- [ ] **Step 2: Build and deploy the merged main commit**

Require package outer and inner SHA256 checks and deployment healthcheck 77/77.

- [ ] **Step 3: Validate live outcomes**

Confirm no ACS admission rejects, no new UECount task expiry, no queue growth, and a materially lower terminal failure ratio across the next hourly cohort while CPU, memory, I/O wait, Redis, and databases remain within limits.

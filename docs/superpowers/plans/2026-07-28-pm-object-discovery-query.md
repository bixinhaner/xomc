# PM Object Discovery Query Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace heavyweight PM compatibility-view object discovery and pivot-skeleton reads with indexed source-table queries while preserving page and export contracts.

**Architecture:** Keep `Aggregator.DiscoverObjectLDNs(context.Context, QueryRequest) ([]string, error)` as the shared page/export boundary. Route 15-minute object and pivot-key discovery to measurement anchors and rolled-up discovery to aggregation results, then add Docker shared-memory headroom as defense in depth.

**Tech Stack:** Go, Squirrel, pgx, PostgreSQL/TimescaleDB, Docker Compose.

## Global Constraints

- Do not change `/api/v1/pm/metrics/aggregated` request or response fields.
- Do not change export task params or CSV layout.
- Preserve device visibility, technology, OUI/SN, granularity and half-open time-window filters.
- Preserve existing uncommitted query prefilter changes.

---

### Task 1: Add source-routing regression tests

**Files:**
- Modify: `omcgo/internal/pm/aggregator/count_test.go`

**Interfaces:**
- Consumes: `Aggregator.DiscoverObjectLDNs(context.Context, QueryRequest) ([]string, error)`
- Produces: regression expectations for raw and rolled-up discovery SQL

- [ ] **Step 1: Write a failing 15-minute test**

Assert that generated SQL reads `pm_measurement_anchors`, uses `device_dim_id`, applies the
half-open time window, and does not reference `pm_metrics`, `pm_files`, or `metric_path`.

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd omcgo && go test ./internal/pm/aggregator -run DiscoverObjectLDNs -count=1`

Expected: FAIL because the current implementation contains `FROM pm_metrics`.

- [ ] **Step 3: Write a failing rolled-up test**

Assert that hourly discovery reads `pm_aggregation_results` with
`dimension='device'`, `granularity='hourly'`, device, technology and time filters.

- [ ] **Step 4: Run the focused tests and verify RED**

Run: `cd omcgo && go test ./internal/pm/aggregator -run DiscoverObjectLDNs -count=1`

Expected: FAIL because the current implementation routes through `pm_metrics_hourly`.

### Task 2: Implement indexed discovery

**Files:**
- Modify: `omcgo/internal/pm/aggregator/query.go`

**Interfaces:**
- Consumes: `QueryRequest`
- Produces: unchanged `DiscoverObjectLDNs(context.Context, QueryRequest) ([]string, error)`

- [ ] **Step 1: Add raw and rolled-up query builders**

Build raw discovery from `pm_measurement_anchors a` joined to `device_dim`, and rolled-up
discovery from `pm_aggregation_results r`. Reuse the existing visibility helpers instead of
duplicating authorization semantics.

- [ ] **Step 2: Route by granularity**

Use the raw builder only for `15min`; use the rolled-up builder for hourly, daily, weekly and
monthly. Return the existing table-selection error for unsupported granularity/dimension pairs.

- [ ] **Step 3: Run focused tests and verify GREEN**

Run: `cd omcgo && go test ./internal/pm/aggregator -run DiscoverObjectLDNs -count=1`

Expected: PASS.

- [ ] **Step 4: Run export contract tests**

Run: `cd omcgo && go test ./internal/pm/export -run 'KpiQuery.*Skeleton|Dashboard' -count=1`

Expected: PASS without frontend or export interface changes.

### Task 3: Add shared-memory defense in depth

**Files:**
- Modify: `deployments/docker/docker-compose.yml`

**Interfaces:**
- Consumes: `TSDB_SHM_SIZE`
- Produces: `postgres-tsdb` `/dev/shm` size, default 512 MiB

- [ ] **Step 1: Add the configurable Compose setting**

Add `shm_size: ${TSDB_SHM_SIZE:-512m}` to `postgres-tsdb`. Environment-specific
deployments may override `TSDB_SHM_SIZE` in their ignored `resources.env`.

- [ ] **Step 2: Render and validate Compose**

Run: `docker compose --env-file deployments/docker/resources.env -f deployments/docker/docker-compose.yml config`

Expected: exit 0 and rendered `shm_size` of 512 MiB.

### Task 4: Route pivot skeleton keys to the same lightweight sources

**Files:**
- Modify: `omcgo/internal/pm/aggregator/query.go`
- Modify: `omcgo/internal/pm/aggregator/query_test.go`
- Modify: `omcgo/internal/pm/aggregator/count_test.go`

- [x] **Step 1: Add failing pivot-key and skeleton-count tests**

Cover 15-minute anchors and rolled-up aggregation results, and reject metric dictionary/value
expansion in the skeleton branch.

- [x] **Step 2: Reuse the minimal object source**

Use the minimal source for page keys, standalone pivot-row keys and skeleton counts while keeping
the selected-metric data query unchanged.

- [x] **Step 3: Verify the real page request**

Deploy and repeat the same single-device/single-indicator request. Confirm HTTP 200, identical
18-row result and sub-second response.

### Task 5: Unified verification

**Files:**
- Verify all files above.

**Interfaces:**
- Consumes: completed implementation
- Produces: evidence that the fix is safe to deploy

- [ ] **Step 1: Format Go files**

Run: `gofmt -w omcgo/internal/pm/aggregator/query.go omcgo/internal/pm/aggregator/count_test.go`

- [ ] **Step 2: Run PM-focused tests**

Run: `cd omcgo && go test ./internal/pm/aggregator ./internal/pm/export ./internal/pm -count=1`

- [ ] **Step 3: Run backend build and full tests**

Run: `cd omcgo && go build ./... && go test ./...`

- [ ] **Step 4: Review the final diff**

Confirm no frontend source, API field, export parameter or unrelated user change was modified.

# PM Sparse Storage and Rollup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace expanded-null PM storage with sparse values plus measurement anchors, preserve existing query results, and publish bounded hourly rollups atomically.

**Architecture:** Keep the existing public `pm_metrics` and `pm_metrics_hourly` read contracts as compatibility views. Store physical values in narrow hypertables and reconstruct supported-but-missing values by expanding immutable metric sets referenced by measurement anchors. Ingestion resolves dictionaries and metric sets inside the existing file transaction; hourly rollup writes a new bucket version in device batches and exposes it only after an atomic active-version switch.

**Tech Stack:** Go 1.x, pgx v5, Squirrel, PostgreSQL 16, TimescaleDB, Prometheus.

## Global Constraints

- Do not migrate old PM load-test data; the PM TimescaleDB is rebuilt during deployment.
- Keep PM API, Counter, KPI, Dashboard, CSV, pagination, and `fill_empty` logical results compatible.
- Store only actually reported finite values in physical value hypertables.
- Preserve measurement-present/missing-counter semantics with immutable historical metric sets.
- Unknown reported metrics must be registered and stored, never silently discarded.
- File marker, anchors, values, and ingestion batch state commit in one TimescaleDB transaction.
- Only aggregate closed hours; use advisory locking, configurable late-data handling, bounded device batches, and atomic whole-bucket publication.
- Do not rely on `SET LOCAL work_mem=512MB` for unbounded aggregation.
- Keep PM disk backpressure at 70%; do not change retention periods.

---

### Task 1: Sparse TimescaleDB schema

**Files:**
- Modify: `omcgo/migrations/tsdb/000001_tsdb_schema.sql`
- Modify: `omcgo/migrations/README.md`
- Test: `omcgo/test/integration/pm_sparse_schema_test.go`

**Interfaces:**
- Produces: `pm_metric_dictionary`, `pm_metric_sets`, `pm_measurement_anchors`, `pm_metric_values`, `pm_ingest_batches`, `pm_hourly_bucket_versions`, `pm_hourly_anchors`, and `pm_hourly_values`.
- Produces: compatibility views `pm_metrics` and `pm_metrics_hourly` with the current 14-column row contract.

- [x] **Step 1: Write a failing static migration test**

```go
func TestTSDBBaselineDefinesSparsePMStorage(t *testing.T) {
    sql := readTSDBBaseline(t)
    for _, object := range []string{
        "pm_metric_dictionary", "pm_metric_sets", "pm_measurement_anchors",
        "pm_metric_values", "pm_hourly_bucket_versions", "pm_hourly_anchors",
        "pm_hourly_values",
    } {
        require.Contains(t, sql, "CREATE TABLE public."+object)
    }
    require.Contains(t, sql, "CREATE VIEW public.pm_metrics AS")
    require.NotContains(t, tableBlock(sql, "pm_metric_values"), "extra jsonb")
}
```

- [x] **Step 2: Run it and verify it fails**

Run: `cd omcgo && go test ./test/integration -run TestTSDBBaselineDefinesSparsePMStorage -v`

Expected: FAIL because the sparse tables and compatibility views do not exist.

- [x] **Step 3: Replace the PM baseline DDL**

Create immutable metric dictionary/set tables, raw/hourly anchors and narrow value hypertables, batch/version control tables, minimal indexes, matching chunk intervals, and compatibility views that derive legacy fields from anchors, dictionary, `pm_files`, and `device_dim`.

- [x] **Step 4: Run the test**

Run: `cd omcgo && go test ./test/integration -run TestTSDBBaselineDefinesSparsePMStorage -v`

Expected: PASS.

### Task 2: Atomic sparse file ingestion

**Files:**
- Create: `omcgo/internal/pm/metrics/sparse_ingest.go`
- Create: `omcgo/internal/pm/metrics/sparse_ingest_test.go`
- Modify: `omcgo/internal/pm/metrics/copy_ingest.go`
- Modify: `omcgo/internal/pm/metrics/pg_repository.go`

**Interfaces:**
- Consumes: `FileMarker`, `[]model.PMCounter`, and `[]model.KPIValue`.
- Produces: `SparseMeasurement`, `SparseMetric`, deterministic metric-set hashes, and `CopyIngest` transactional writes.

- [x] **Step 1: Add failing tests for sparse projection**

Test that reported finite values become `SparseMetric` rows, NaN placeholders contribute to metric sets but not value rows, empty measurement groups still create anchors, and identical sorted sets have identical SHA-256 hashes.

- [x] **Step 2: Verify RED**

Run: `cd omcgo && go test ./internal/pm/metrics -run 'TestBuildSparse|TestMetricSetHash' -v`

Expected: FAIL because sparse projection does not exist.

- [x] **Step 3: Implement pure sparse projection**

Group by device/object/time/group/granularity, deduplicate metric paths last-wins, sort set members, retain every anchor, and exclude NaN/Inf from physical values.

- [x] **Step 4: Verify GREEN**

Run the same focused command and expect PASS.

- [x] **Step 5: Make `CopyIngest` transactional**

Insert the file marker and `building` batch, upsert dictionary metadata, resolve IDs, reuse/create metric sets by content hash, insert anchors with returned identities, COPY finite values, and mark the batch committed before committing. A marker conflict returns `ingested=false`.

- [x] **Step 6: Run metrics tests**

Run: `cd omcgo && go test ./internal/pm/metrics -v`

Expected: PASS.

### Task 3: Query compatibility and unknown metrics

**Files:**
- Modify: `omcgo/internal/pm/collector/collector.go`
- Modify: `omcgo/internal/pm/collector/filter_test.go`
- Modify: `omcgo/internal/pm/collector/result_normalization.go`
- Modify: `omcgo/internal/pm/collector/result_normalization_test.go`
- Modify: `omcgo/internal/tsdbsync/sync.go`
- Test: `omcgo/internal/pm/metrics/pg_repository_test.go`

**Interfaces:**
- Compatibility views expose current row columns; existing repository/query builders continue to read `pm_metrics`.
- Unknown metrics retain the original report key/path and raw finite value with nullable metadata.

- [x] **Step 1: Write failing unknown-metric tests**

Assert whitelist misses remain in the output, known metrics are still rewritten to indicator IDs, and normalization leaves unknown unit/statis metadata unchanged while still normalizing known metrics.

- [x] **Step 2: Verify RED**

Run: `cd omcgo && go test ./internal/pm/collector -run 'Unknown|Whitelist' -v`

Expected: FAIL because whitelist misses are currently dropped.

- [x] **Step 3: Implement fail-open unknown registration semantics**

Retain unknown rows, record the existing observable metric/log, skip configured normalization only when both metadata fields are absent, and let sparse ingestion register the minimal dictionary entry.

- [x] **Step 4: Mirror `report_key`**

Add `report_key` to TSDB indicator mirrors so later dictionary refreshes can enrich minimal unknown entries.

- [x] **Step 5: Run collector, repository, and query regression tests**

Run: `cd omcgo && go test ./internal/pm/collector ./internal/pm/metrics ./internal/pm/aggregator ./internal/pm/counter ./internal/pm/kpi ./internal/dashboard ./internal/pm/export`

Expected: PASS.

### Task 4: Bounded, versioned hourly rollup

**Files:**
- Create: `omcgo/internal/pm/aggregator/hourly_versioned.go`
- Create: `omcgo/internal/pm/aggregator/hourly_versioned_test.go`
- Modify: `omcgo/internal/pm/aggregator/runner.go`
- Modify: `omcgo/internal/pm/aggregator/aggregator.go`
- Modify: `omcgo/cmd/app/provider/pm.go`

**Interfaces:**
- Produces: `RunHourlyVersioned(ctx, WindowSpec, batchSize) (RollupStats, error)`.
- Uses: a session advisory lock held by a dedicated connection across all batch transactions, bucket version states `building|active|superseded|failed`, and configurable `PM_HOURLY_BATCH_DEVICES`.

- [x] **Step 1: Add failing SQL/state-machine tests**

Assert closed-hour validation, advisory locking, device-batch predicates, no `ON CONFLICT` on hourly values, no 512 MB work-memory override, and active-version publication only after validation.

- [x] **Step 2: Verify RED**

Run: `cd omcgo && go test ./internal/pm/aggregator -run VersionedHourly -v`

Expected: FAIL because the versioned runner does not exist.

- [x] **Step 3: Implement building-version batches**

Create one version per attempt, enumerate stable device-ID batches, build hourly metric-set unions and finite aggregates into the version, persist batch progress, and mark late writes dirty.

- [x] **Step 4: Implement atomic publication and cleanup**

Validate batch/anchor/value counts, supersede the previous active version and activate the new version in one short transaction, update the watermark, and delete failed/superseded versions only in cleanup.

- [x] **Step 5: Wire only the hourly runner**

Route `pm.rollup.hourly` through the new implementation; daily/weekly/monthly continue consuming the hourly compatibility view.

- [x] **Step 6: Run aggregator tests**

Run: `cd omcgo && go test ./internal/pm/aggregator ./cmd/app/provider -v`

Expected: PASS.

### Task 5: Operations defaults, observability, and full regression

**Files:**
- Modify: `omcgo/internal/pm/aggregator/metrics.go`
- Modify: `omcgo/internal/pm/retention/service.go`
- Modify: `deployments/docker/docker-compose.yml`
- Modify: `deployments/docker/README.md`
- Modify: `omcgo/migrations/README.md`
- Create: `omcgo/scripts/pm_sparse_compare.go`

**Interfaces:**
- Produces metrics for queue oldest age, rollup watermark/lag, batch rows, dirty buckets, and sparse amplification.
- Compression eligibility requires late window elapsed, successful rollup, and no dirty/failed state.

- [x] **Step 1: Add failing configuration/retention tests**

Verify 70% PM backpressure remains, batch and late-window defaults are present, release DB diagnostic logging defaults are disabled, and retention/compression SQL checks active hourly versions.

- [x] **Step 2: Implement defaults and metrics**

Add bounded batch configuration, late window, state gauges/counters, container log rotation, and watermarked compression eligibility without changing retention durations.

- [x] **Step 3: Add the offline golden comparator**

Compare normalized logical rows, null/filled counts, stable ordering, historical set changes, unknown metrics, hourly output, and CSV content between two JSON/CSV exports.

- [x] **Step 4: Run fresh verification**

Run:

```bash
cd omcgo
go build ./...
go test ./...
```

Expected: exit 0 with no failing packages.

- [x] **Step 5: Review the design requirement-by-requirement**

Confirm every development split item A-F is either implemented and tested or explicitly identified as environment-only acceptance work (real `EXPLAIN`, 4/12/24-hour chunk benchmark, 10,000-station overnight run, and destructive PM volume rebuild).

## Acceptance Status

- Development and automated regression are complete.
- Sparse ingestion and versioned hourly publication were validated against an isolated TimescaleDB instance on the target server.
- Environment-only acceptance remains: production-data `EXPLAIN`, 4/12/24-hour chunk benchmark, 10,000-station overnight run, golden export comparison using production samples, and the destructive PM volume rebuild/deployment.
- The destructive rebuild requires a separate explicit confirmation naming the production PM volume before execution.

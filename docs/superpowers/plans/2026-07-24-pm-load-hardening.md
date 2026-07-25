# PM Load Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate every confirmed P0/P1/P2 issue from the 10,000-device PM load test, prove sparse storage correctness and value, deploy the result to `172.24.224.197`, and update the existing merge request with reproducible evidence.

**Architecture:** Keep PM ingestion, rollup, and queue control independently testable, but enforce one database lock order: bucket/version state → dictionary → metric set → anchors/values. Stable metadata becomes insert-only on the hot path. Retriable transaction failures are handled at the smallest safe transaction boundary. Queue pressure combines JetStream depth, age, and rate signals with hysteresis. Operations evidence is produced by first-class metrics, alerts, health checks, and an isolated old/new comparison tool.

**Tech Stack:** Go 1.x, pgx v5, Squirrel, PostgreSQL/TimescaleDB, NATS JetStream, Prometheus, Grafana, Docker Compose, GitLab.

## Global Constraints

- Follow `AGENTS.md` and `omcgo/AGENTS.md`; all Git operations run at repository root.
- Use TDD for every behavioral change: failing test first, minimal implementation, focused test, then broader test.
- Never update stable dictionary or metric-set rows merely to obtain their identifiers.
- Never retry a transaction object after PostgreSQL aborts it; every retry creates a new transaction.
- Retry only SQLSTATE `40P01` and `40001`, at most three attempts, with bounded jitter and context cancellation.
- Preserve the approved lock order in ingestion, hourly rollup, maintenance, and recovery paths.
- Keep all existing user changes and avoid unrelated refactors.
- Do not delete indexes until production-shaped `EXPLAIN (ANALYZE, BUFFERS)` evidence proves they are redundant.
- A task is complete only after its focused tests pass and its checkbox/evidence is updated.
- Final completion requires local verification, server deployment, 10,000-device validation, commit, push, and update of MR `!324`.

---

## Task 1: Make PM Metadata Resolution Insert-Only

**Files:**

- Create: `omcgo/internal/pm/metrics/sparse_dictionary.go`
- Create: `omcgo/internal/pm/metrics/sparse_dictionary_test.go`
- Modify: `omcgo/internal/pm/metrics/sparse_ingest.go`
- Modify: `omcgo/internal/pm/metrics/sparse_ingest_test.go`
- Modify: `omcgo/internal/pm/metrics/sparse_ingest_integration_test.go`

- [ ] **Step 1: Add failing SQL-shape tests**

Test that dictionary resolution:

1. queries existing paths first;
2. inserts only missing paths with `ON CONFLICT (metric_path) DO NOTHING`;
3. never contains `DO UPDATE`;
4. re-queries and returns every requested ID;
5. reports duplicate/missing/inconsistent metadata as an error.

Test that metric-set resolution:

1. computes a deterministic hash from sorted metric IDs;
2. queries by `(product_key, counter_group, content_hash)` first;
3. inserts missing sets with `DO NOTHING`;
4. never updates `metric_ids`;
5. rejects an existing row whose stored IDs do not match the hash input.

Run:

```bash
cd omcgo
go test ./internal/pm/metrics -run 'TestResolveMetric(Dictionary|Set)' -count=1
```

Expected: FAIL because the resolver helpers do not exist and current SQL uses `DO UPDATE`.

- [ ] **Step 2: Implement dictionary resolution**

Add focused helpers with transaction-scoped interfaces:

```go
type sparseMetadataQuerier interface {
    Query(context.Context, string, ...any) (pgx.Rows, error)
    QueryRow(context.Context, string, ...any) pgx.Row
    Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func resolveMetricDictionary(
    ctx context.Context,
    tx sparseMetadataQuerier,
    defs []metricDefinition,
) (map[string]int64, error)
```

The helper must sort/deduplicate paths, query existing rows, insert only missing definitions, then re-query. If an existing metric has incompatible immutable identity, return a wrapped error instead of mutating it.

- [ ] **Step 3: Implement immutable metric-set resolution**

Add:

```go
func resolveMetricSet(
    ctx context.Context,
    tx sparseMetadataQuerier,
    productKey, counterGroup, contentHash string,
    metricIDs []int64,
) (int64, error)
```

Use `SELECT`, `INSERT ... ON CONFLICT DO NOTHING`, then `SELECT`. Validate sorted `metric_ids` against the requested IDs.

- [ ] **Step 4: Route sparse ingestion through the helpers**

Remove both hot-path `DO UPDATE` clauses from `writeSparseMeasurements`. Keep the final anchor/value write behavior unchanged.

- [ ] **Step 5: Add integration regression coverage**

Run two identical ingests and assert:

- dictionary `xmin`/`updated_at` is unchanged after the second ingest;
- metric-set row is unchanged;
- anchor/value idempotency still holds;
- no duplicate dictionary or metric-set rows exist.

Run:

```bash
cd omcgo
go test ./internal/pm/metrics -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the isolated metadata fix**

```bash
git add omcgo/internal/pm/metrics
git commit -m "fix(pm): 消除稀疏写入元数据热更新"
```

---

## Task 2: Enforce One Lock Order Across Ingestion and Formula KPI Rollup

**Files:**

- Modify: `omcgo/internal/pm/metrics/sparse_ingest.go`
- Modify: `omcgo/internal/pm/metrics/sparse_ingest_test.go`
- Modify: `omcgo/internal/pm/aggregator/hourly_versioned.go`
- Modify: `omcgo/internal/pm/aggregator/hourly_versioned_test.go`
- Modify: `omcgo/internal/pm/aggregator/hourly_versioned_integration_test.go`

- [ ] **Step 1: Add failing lock-order tests**

Add recording transaction fakes that assert the statement order:

```text
ingest: bucket dirty/version lock → dictionary → set → anchor/value
rollup: formula dictionary registration → bucket/version transaction → rollup values
```

Run:

```bash
cd omcgo
go test ./internal/pm/metrics ./internal/pm/aggregator \
  -run 'Test.*LockOrder|TestFormulaDictionaryRegisteredBeforeVersionLock' -count=1
```

Expected: FAIL because ingestion currently marks dirty after metadata/value writes and formula registration currently runs while holding version/bucket work.

- [ ] **Step 2: Move dirty marking to the beginning of ingestion**

Call `markHourlyBucketsDirty` before dictionary resolution. Keep it in the same transaction so rollback remains atomic. Document the lock-order invariant immediately above the call.

- [ ] **Step 3: Split formula preparation from batch execution**

Refactor hourly formula work into:

```go
type preparedFormulaKPI struct {
    MetricID  int64
    // expression inputs required by the batch query
}

func prepareVersionedHourlyFormulaKPIs(
    ctx context.Context,
    db DBTX,
    productKey string,
) ([]preparedFormulaKPI, error)
```

Resolve/register formula dictionary entries before opening the transaction that locks and writes the hourly bucket/version. The batch transaction receives immutable metric IDs and must not write dictionary rows.

- [ ] **Step 4: Prove the deadlock cycle is gone**

Add an integration test with two connections and barriers: one ingestion transaction and one hourly rollup transaction target the same bucket. The test must complete within a deadline and leave one active hourly version.

Run:

```bash
cd omcgo
go test ./internal/pm/metrics ./internal/pm/aggregator -count=1
```

Expected: PASS with no deadlock.

- [ ] **Step 5: Commit lock-order enforcement**

```bash
git add omcgo/internal/pm/metrics omcgo/internal/pm/aggregator
git commit -m "fix(pm): 统一写入与聚合锁顺序"
```

---

## Task 3: Retry Only Retriable Hourly Batch Transactions

**Files:**

- Create: `omcgo/internal/pm/aggregator/tx_retry.go`
- Create: `omcgo/internal/pm/aggregator/tx_retry_test.go`
- Modify: `omcgo/internal/pm/aggregator/hourly_versioned.go`
- Modify: `omcgo/internal/pm/aggregator/metrics.go`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`

- [ ] **Step 1: Add failing retry classification and lifecycle tests**

Cover:

- SQLSTATE `40P01` and `40001` retry;
- all other errors return immediately;
- each attempt begins a new transaction;
- failed transaction is rolled back;
- successful transaction is committed once;
- maximum three attempts;
- context cancellation stops backoff;
- jitter stays within the configured bound.

Run:

```bash
cd omcgo
go test ./internal/pm/aggregator -run 'TestRunTransactionWithRetry' -count=1
```

Expected: FAIL because the helper does not exist.

- [ ] **Step 2: Implement the retry helper**

Add:

```go
func runTransactionWithRetry(
    ctx context.Context,
    begin func(context.Context) (pgx.Tx, error),
    maxAttempts int,
    run func(context.Context, pgx.Tx) error,
) error
```

Extract PostgreSQL code via `errors.As(err, *pgconn.PgError)`. Use bounded exponential delay such as 25–100 ms plus injectable jitter for deterministic tests.

- [ ] **Step 3: Wrap each hourly batch, not the whole job**

Move `Begin`, batch marker insert, aggregation statements, and `Commit` inside the retry closure. Do not retry formula metadata registration or an already committed batch.

- [ ] **Step 4: Add retry metrics and alerts**

Expose counters by SQLSTATE and exhausted outcome:

```text
omc_pm_hourly_tx_retries_total{sqlstate}
omc_pm_hourly_tx_retry_exhausted_total{sqlstate}
```

Alert on any exhausted retry and sustained retry rate.

- [ ] **Step 5: Verify**

```bash
cd omcgo
go test ./internal/pm/aggregator -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add omcgo/internal/pm/aggregator deployments/monitoring/alerts/omc-rules.yml
git commit -m "fix(pm): 增加小时聚合事务级重试"
```

---

## Task 4: Recover Failed Hourly Buckets Without Infinite Replay

**Files:**

- Create: `omcgo/migrations/000005_async_job_recovery.sql`
- Modify: `omcgo/migrations/000001_init_schema.sql`
- Modify: `omcgo/internal/core/asyncjob/model.go`
- Modify: `omcgo/internal/core/asyncjob/repository.go`
- Modify: `omcgo/internal/core/asyncjob/repository_test.go`
- Modify: `omcgo/internal/pm/aggregator/runner.go`
- Modify: `omcgo/internal/pm/aggregator/sparse_maintenance.go`
- Modify: `omcgo/internal/pm/aggregator/sparse_maintenance_test.go`
- Modify: `omcgo/internal/pm/aggregator/metrics.go`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`

- [ ] **Step 1: Add failing migration and repository tests**

Add `recovery_count integer NOT NULL DEFAULT 0` and `last_recovered_at timestamptz`. Test:

- only failed natural-bucket jobs with retriable SQLSTATE markers are requeued;
- recovery count remains monotonic;
- maximum recovery count is respected;
- cooldown is respected;
- successful, running, pending, and non-retriable failed jobs are untouched;
- concurrent recovery calls requeue once.

Proposed API:

```go
type FailedBucketRecoveryRequest struct {
    JobType      string
    BucketStart  time.Time
    BucketEnd    time.Time
    Payload      json.RawMessage
    MaxRecoveries int
    Cooldown     time.Duration
}

func (r *PgRepository) RequeueRetriableFailedBucket(
    ctx context.Context,
    req FailedBucketRecoveryRequest,
) (uuid.UUID, bool, error)
```

- [ ] **Step 2: Implement one atomic conditional update**

Use Squirrel or a constant SQL statement with a guarded `UPDATE ... RETURNING`. Never reset `recovery_count` in generic `Insert`. Update scanners and model fields.

- [ ] **Step 3: Add maintenance recovery and orphan cleanup**

During sparse maintenance:

- discover failed hourly buckets within a bounded recent horizon;
- call the guarded recovery API;
- mark stale `building` versions failed when their job is terminal or absent beyond timeout;
- enqueue dirty active buckets only after cleanup;
- emit recovery, exhausted, stale-building, failed-bucket, and watermark-lag metrics.

- [ ] **Step 4: Add alert coverage**

Alert for:

- failed hourly bucket older than one maintenance interval;
- `recovery_count` exhausted;
- stale `building` version;
- waterline lag above two hourly buckets.

- [ ] **Step 5: Verify migration idempotency and packages**

```bash
cd omcgo
go test ./internal/core/asyncjob ./internal/pm/aggregator ./test/integration -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add omcgo/migrations omcgo/internal/core/asyncjob \
  omcgo/internal/pm/aggregator deployments/monitoring/alerts/omc-rules.yml
git commit -m "fix(pm): 自动恢复失败小时桶"
```

---

## Task 5: Observe JetStream Queue Depth, Age, Redelivery, and Rates

**Files:**

- Modify: `omcgo/internal/core/event/nats_bus.go`
- Modify: `omcgo/internal/core/event/nats_bus_test.go`
- Modify: `omcgo/internal/core/event/metrics.go`
- Modify: `omcgo/cmd/acs/main.go`

- [ ] **Step 1: Add failing queue-stat tests**

Define:

```go
type QueueStats struct {
    Pending          uint64
    AckPending       int
    Redelivered      int
    OldestPendingAge time.Duration
    LastSequence     uint64
    AckSequence      uint64
    SampledAt        time.Time
}

func (b *NATSBus) QueueStats(
    ctx context.Context,
    subject, durable string,
) (QueueStats, error)
```

Use consumer info for pending/ack/redelivery/delivery floors and stream info for last sequence and oldest retained timestamp. Test empty streams, pending streams, deleted/advanced first sequence, missing consumers, and context cancellation.

Run:

```bash
cd omcgo
go test ./internal/core/event -run 'TestNATSBusQueueStats' -count=1
```

Expected: FAIL because only aggregate `PendingCount` exists.

- [ ] **Step 2: Implement queue stats and keep compatibility**

Implement `PendingCount` in terms of `QueueStats` for one release. Timestamp every successful sample at the end of collection.

- [ ] **Step 3: Export Prometheus gauges/counters**

Expose:

```text
omc_pm_queue_pending
omc_pm_queue_ack_pending
omc_pm_queue_redelivered
omc_pm_queue_oldest_age_seconds
omc_pm_queue_last_sequence
omc_pm_queue_ack_sequence
omc_pm_queue_sample_timestamp_seconds
```

- [ ] **Step 4: Wire ACS startup**

Replace the pending-count adapter with the full queue-stat adapter. Do not change stream or durable names.

- [ ] **Step 5: Verify**

```bash
cd omcgo
go test ./internal/core/event ./cmd/acs -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add omcgo/internal/core/event omcgo/cmd/acs
git commit -m "feat(pm): 补齐队列健康指标"
```

---

## Task 6: Make Backpressure React to Queue Risk With Hysteresis

**Files:**

- Modify: `omcgo/internal/acs/upload/backpressure.go`
- Modify: `omcgo/internal/acs/upload/backpressure_test.go`
- Modify: `omcgo/internal/acs/upload/backpressure_integration_test.go`
- Modify: `omcgo/internal/core/appconfig/config.go`
- Modify: `omcgo/configs/config.yaml`
- Modify: `omcgo/configs/config.prod.yaml`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`
- Modify: `deployments/monitoring/grafana/dashboards/omc-overview.json`

- [ ] **Step 1: Add failing pure-decision tests**

Extend configuration:

```go
QueuePendingHigh int
QueuePendingLow  int
QueueOldestHigh  time.Duration
QueueOldestLow   time.Duration
QueueSlopeWindow time.Duration
```

Defaults: pending high/low `2000/500`, age high/low `10m/2m`.

Test:

- high pending or old queue activates pressure;
- pressure releases only when all available signals are below low watermarks;
- positive pending slope prevents release;
- queue read failure preserves the current state;
- startup failure remains fail-open only when no pressure has previously been observed;
- counter reset and stale sample do not produce false rates.

Run:

```bash
cd omcgo
go test ./internal/acs/upload -run 'TestDecideBackpressure|TestQueueSignal' -count=1
```

Expected: FAIL.

- [ ] **Step 2: Implement sampled signal state**

Store the previous successful `QueueStats` sample and derive:

```go
type QueueRates struct {
    PendingPerSecond     float64
    AckAdvancePerSecond  float64
    DeliveryPerSecond    float64
}
```

Make the decision function pure and keep I/O in the watchdog loop.

- [ ] **Step 3: Export reason and rate metrics**

Add pressure-state gauge, reason-labelled transition counter, queue rates, sample failures, and last-success timestamp.

- [ ] **Step 4: Update alerts and dashboard**

Visualize pending, ack pending, oldest age, redelivery, input/ack rate, pressure state, and transitions. Alert if age/pending remains high, queue sample is stale, or ack rate is saturated below arrival rate.

- [ ] **Step 5: Verify**

```bash
cd omcgo
go test ./internal/acs/upload ./internal/core/appconfig ./cmd/acs -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add omcgo/internal/acs/upload omcgo/internal/core/appconfig \
  omcgo/configs deployments/monitoring
git commit -m "fix(pm): 按队列风险启停上传背压"
```

---

## Task 7: Correct Counter Semantics and Operational Gaps

**Files:**

- Modify: `omcgo/internal/pm/metrics.go`
- Modify: PM collector call sites and tests located with `rg 'DroppedCountersTotal' omcgo`
- Modify: `deployments/monitoring/alerts/omc-rules.yml`
- Modify: `deployments/monitoring/grafana-dashboard.json`
- Modify: `deployments/monitoring/grafana/dashboards/omc-overview.json`
- Modify: `omcgo/internal/backup/restore_service.go`
- Modify: `omcgo/internal/backup/policy_storage_monitor.go`
- Modify: `omcgo/internal/backup/restore_service_test.go`
- Modify: `omcgo/internal/backup/policy_storage_monitor_test.go`
- Modify: `deployments/release/bundle/deploy/docker-compose.monitoring.yml`
- Modify: `deployments/release/bundle/deploy/healthcheck.sh`
- Modify: `deployments/release/bundle/deploy/README.md`

- [ ] **Step 1: Rename the misleading PM metric with compatibility**

Add `omc_pm_discovered_counters_total`. Increment it where unknown counters are preserved/discovered. Keep the old dropped metric as a deprecated alias for one release, with identical value and explicit HELP text. Update queries and alerts to use the new name.

Add failing tests proving no counter is described as dropped when it is persisted.

- [ ] **Step 2: Fix the backup bucket name**

Use the valid physical S3 bucket name `config-backup`. Accept legacy logical `config_backup` input only as a compatibility alias at API/config boundaries. Add tests that every bucket sent to MinIO passes S3 naming rules.

If a legacy physical bucket cannot exist because of S3 validation, no object migration is required; document that conclusion and ensure startup creates/uses the valid bucket.

- [ ] **Step 3: Make tracing deployment-consistent**

Retain the existing no-op tracer when `tracing.enabled=false`. Ensure release deployment sets tracing enabled only with a configured collector and starts the complete monitoring profile for production validation.

Enable the OTEL Collector health-check extension and have `healthcheck.sh` verify it externally rather than assuming a shell exists in the distroless image.

- [ ] **Step 4: Add context-cancellation distinction**

At dashboard and monitoring query boundaries, treat `context.Canceled`/`DeadlineExceeded` as canceled requests, not database/service failures. Add tests that error counters and warning logs are not incremented for client cancellation.

- [ ] **Step 5: Verify**

```bash
cd omcgo
go test ./internal/pm/... ./internal/backup/... ./internal/dashboard/... -count=1
cd ..
bash deployments/release/bundle/deploy/storage-compose_test.sh
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add omcgo deployments/monitoring deployments/release/bundle/deploy
git commit -m "fix(ops): 修正PM指标与发布健康检查"
```

---

## Task 8: Prove Index Choices and Sparse Storage Value

**Files:**

- Create: `omcgo/internal/pm/compare/compare.go`
- Create: `omcgo/internal/pm/compare/compare_test.go`
- Modify: `omcgo/scripts/pm_sparse_compare.go`
- Create: `omcgo/scripts/pm_explain_core_queries.sql`
- Create: `docs/superpowers/evidence/2026-07-24-pm-index-analysis.md`
- Create: `docs/superpowers/evidence/2026-07-24-pm-dual-run.md`

- [ ] **Step 1: Add failing logical/physical comparison tests**

Test:

- stable logical comparison independent of ordering;
- exact mismatch reporting by device/time/counter;
- duplicate-key detection;
- physical byte and ratio calculation;
- division-by-zero/empty-run handling;
- failure when sparse physical bytes exceed 20% of old representation.

- [ ] **Step 2: Extract a testable comparison package**

Expose:

```go
type Result struct {
    LogicalEqual  bool
    Mismatches    []Mismatch
    OldBytes      int64
    SparseBytes   int64
    SparseRatio   float64
}

func Compare(oldRows, sparseRows []Row, oldBytes, sparseBytes int64) Result
```

Keep `scripts/pm_sparse_compare.go` as a thin CLI wrapper.

- [ ] **Step 3: Capture production-shaped query plans**

Add a read-only SQL script for core ingest lookup, latest metric lookup, hourly aggregation, dashboard range queries, and cleanup queries. Run with `EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)` against a copied/isolated dataset.

Record plan, row estimates, actual time, buffers, index size, scan/write frequency, and decision for each candidate index.

- [ ] **Step 4: Apply only evidence-backed index changes**

If no index is proven redundant, make no schema deletion and explicitly record “retained”. If an index is proven redundant, add a dedicated migration plus before/after plan evidence and focused migration test.

- [ ] **Step 5: Run isolated same-input dual execution**

Feed the exact same captured raw PM files into old/main and new sparse paths in separate schemas/databases. Compare logical rows and measure table+index bytes after checkpoint/analyze. Require:

```text
logical_equal = true
sparse_bytes / old_bytes <= 0.20
```

- [ ] **Step 6: Verify and commit**

```bash
cd omcgo
go test ./internal/pm/compare -count=1
go run ./scripts/pm_sparse_compare.go --help
```

Expected: PASS.

```bash
git add omcgo/internal/pm/compare omcgo/scripts \
  docs/superpowers/evidence
git commit -m "test(pm): 增加稀疏存储对照验证"
```

---

## Task 9: Full Local Verification and Independent Review

**Files:**

- Modify only files required by discovered defects.
- Update: `docs/superpowers/plans/2026-07-24-pm-load-hardening.md`

- [x] **Step 1: Inspect the complete diff**

```bash
git status --short
git diff --check
git diff --stat origin/main...HEAD
git log --oneline --decorate -12
```

Confirm local AI configuration files are not tracked.

- [x] **Step 2: Run backend verification**

```bash
cd omcgo
gofmt -w <changed-go-files>
go build ./...
go test ./...
```

If local-listener tests are sandbox-blocked, rerun with the required permission and distinguish environment failures from real failures.

- [x] **Step 3: Run frontend verification**

```bash
cd omcmb
npm run typecheck
```

- [x] **Step 4: Run deployment/static validation**

```bash
bash deployments/release/bundle/deploy/storage-compose_test.sh
docker compose -f deployments/docker/docker-compose.yml config
```

- [x] **Step 5: Review every P0/P1/P2 acceptance item**

Use `superpowers:requesting-code-review`. Resolve every actionable finding, rerun affected tests, then repeat the full verification. Do not waive an item without evidence.

- [x] **Step 6: Commit verification-only fixes if necessary**

Use a Conventional Commit message describing the actual fix.

**Task 9 evidence (2026-07-25):** final local verification passed after independent-review fixes, including `go build ./...`, `go test ./...`, `npm run typecheck`, release static validation, and Compose configuration parsing. Verification-only follow-up commits are `a141a9868` and the subsequent metadata-builder review fix. The detailed local report is intentionally ignored at `.superpowers/sdd/pm-task-9-report.md`. Task 8's isolated EXPLAIN/dual-run evidence and every deployment/load threshold remain Task 10 live gates and are not completed here.

---

## Task 10: Deploy and Validate on the 10,000-Device Server

**Target:** `172.24.224.197`

**Files:**

- Create: `docs/superpowers/evidence/2026-07-24-pm-10000-validation.md`
- Update: `docs/superpowers/plans/2026-07-24-pm-load-hardening.md`

- [ ] **Step 1: Capture a pre-deployment baseline**

Record timestamped:

- host CPU, load, memory, swap, disk space, disk latency/IO pressure;
- container health/restarts/resources;
- PostgreSQL connections, waits, deadlocks, rollbacks, table/index bytes, dead tuples, autovacuum, active/blocked queries;
- NATS pending, ack pending, redelivery, oldest age, last/ack sequence;
- async jobs by state, oldest pending, failed bucket details;
- hourly versions by state, stale building rows, watermark;
- service logs and current application version.

- [ ] **Step 2: Deploy the exact verified commit**

Push the branch only after local verification, build/deploy the exact commit through the repository release/Compose workflow, run migrations, and start application, infrastructure, and monitoring services. Never use the bare-process restart scripts.

- [ ] **Step 3: Run release health checks**

Require healthy application endpoints, PostgreSQL, TimescaleDB, NATS, MinIO, Prometheus, Grafana, Loki, Tempo, OTEL Collector, and exporters. Require no recurring OTEL endpoint or invalid backup-bucket warning.

- [ ] **Step 4: Observe the active 10,000-device load**

Sample every 30–60 seconds and record:

- pending/ack/redelivery/oldest/rates and pressure state;
- PM ingest throughput and latency;
- CPU/memory/disk/DB locks/deadlocks/autovacuum;
- hourly jobs/versions/watermark;
- error and retry counters.

Do not stop observation until the burst is drained or the 30-minute steady-state window completes.

- [ ] **Step 5: Enforce acceptance thresholds**

Require all:

- 10,000-device burst drains within 15 minutes;
- at 11.1 files/s for 30 minutes, pending slope is non-positive;
- oldest pending age stays below 5 minutes after warm-up;
- ack pending is not continuously saturated;
- zero new PostgreSQL deadlocks;
- no persistent lock wait queue;
- hourly bucket succeeds with exactly one active version and advancing watermark;
- host CPU retains at least 20% idle during steady state;
- no OOM, container restart, or unhealthy dependency;
- sparse logical output equals old output;
- sparse physical storage is at most 20% of old storage.

If any threshold fails, return to the responsible task, add a failing regression test, fix it, repeat local verification, redeploy, and repeat this task.

- [ ] **Step 6: Write evidence**

Include commands/queries, timestamps, before/after values, graphs or compact tables, exact commit SHA, and an explicit pass/fail row for every acceptance threshold. Do not include credentials.

- [ ] **Step 7: Commit evidence**

```bash
git add docs/superpowers/evidence docs/superpowers/plans/2026-07-24-pm-load-hardening.md
git commit -m "docs(pm): 记录10000基站负载验证"
```

---

## Task 11: Push and Update the Existing Merge Request

- [ ] **Step 1: Final clean-tree verification**

```bash
git diff --check
git status --short
cd omcgo && go build ./... && go test ./...
cd ../omcmb && npm run typecheck
```

Require a clean working tree after the evidence commit.

- [ ] **Step 2: Push the verified branch**

```bash
git push origin codex/pm-sparse-storage-rollup
```

- [ ] **Step 3: Update MR `!324`**

Update title/description/checklist with:

- P0/P1/P2 fixes mapped to commits;
- schema/migration notes;
- local test evidence;
- server deployment SHA;
- 10,000-device CPU/memory/database/queue results;
- sparse correctness and storage ratio;
- rollback notes;
- explicit statement that no P0/P1/P2 item remains.

- [ ] **Step 4: Check MR pipeline and discussion state**

Require pipeline success or report the exact failing job. Address all unresolved actionable discussions. Re-run affected verification after any MR-driven change.

- [ ] **Step 5: Final handoff**

Report the MR URL, commit SHA, deployment health, load-test threshold table, and any operational follow-up that is informational rather than an unresolved defect.

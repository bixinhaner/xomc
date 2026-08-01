# Runtime Backlog and Hour-Close Root Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** Eliminate invalid startup PM dead letters, periodic-event amplification, parameter-sync result write amplification, and the 20,000-entity hourly publication tail while preserving the 12-minute late-data grace.

**Architecture:** Keep the existing durable consumers and streaming aggregation. Terminate unregistered PM files at the collector boundary, remove redundant heartbeat grouping, make parameter-sync counters incremental with one set-based full-run cleanup, and stage hourly revisions before an atomic publication-watermark switch.

**Tech Stack:** Go 1.24, pgx v5, Squirrel, PostgreSQL 16, TimescaleDB 2.25, Redis 7, NATS JetStream, MinIO, Prometheus, Docker Compose.

## Global Constraints

- A PM file whose device is still absent after the fixed 30-minute registration grace is illegal data.
- Illegal PM deletes the MinIO object, creates no dead letter, creates no PM/KPI/aggregation data, and records only low-cardinality metrics plus redacted logs.
- The hourly late-data grace remains exactly 12 minutes.
- Dashboard continues to refresh only on its existing five-minute timer and reads stored hourly/daily/weekly results.
- SQL uses Squirrel + pgx; schema changes fold into the tracked 000001 baselines.
- Every behavior change follows RED-GREEN-REFACTOR and gets an independently reviewable Conventional Commit.
- No MR is created until full tests, clean deployment, and 20,000-device production validation pass.

---

### Task 1: Synchronize Main and Establish the Baseline

**Files:**
- Inspect: AGENTS.md
- Inspect: docs/ref/pm-metrics-knowledge.md
- Inspect: deployments/docker/README.md

**Interfaces:**
- Consumes: current branch codex/pm-dashboard-snapshot-scan-root-fix with seven local commits.
- Produces: branch merged with latest origin/main and clean baseline evidence.

- [ ] **Step 1: Fetch and merge latest main without rewriting local commits**

Run:

~~~bash
git fetch origin main
git merge --no-edit origin/main
~~~

Expected: merge succeeds without dropping existing dashboard/parameter-sync fixes. Resolve only true overlaps; never reset or discard user work.

- [ ] **Step 2: Verify workspace and dependencies**

~~~bash
git status --short --branch
cd omcgo
go mod download
~~~

Expected: no uncommitted source changes and dependencies resolve.

- [ ] **Step 3: Run targeted baseline**

~~~bash
cd omcgo
go test ./internal/pm/collector ./internal/device ./internal/paramsync ./internal/pm/stream -count=1
~~~

Expected: PASS. Diagnose any baseline failure before implementation.

---

### Task 2: Discard Illegal Unregistered PM Without DLQ

**Files:**
- Modify: omcgo/internal/pm/collector/collector.go
- Modify: omcgo/internal/pm/collector/subscribe_test.go
- Modify: omcgo/internal/pm/metrics.go
- Modify: omcgo/internal/pm/metrics_test.go

**Interfaces:**
- Consumes: FileReceivedPayload, DeviceRegistrationGrace, reliability.ErrDeferred, MinIO RemoveObject.
- Produces: rawObjectDiscarder, PMCollector.SetRawObjectDiscarder, PMMetrics.FilesDiscardedTotal.

- [ ] **Step 1: Write failing collector tests**

Add a fake implementing:

~~~go
type rawObjectDiscarder interface {
    RemoveObject(context.Context, string, string, minio.RemoveObjectOptions) error
}
~~~

Required tests:

~~~go
func TestHandleFileReceivedExpiredUnregisteredDeviceDeletesRawAndACKs(t *testing.T)
func TestHandleFileReceivedExpiredUnregisteredDeviceMissingObjectACKs(t *testing.T)
func TestHandleFileReceivedExpiredUnregisteredDeviceDeleteFailureDefersWithoutDLQ(t *testing.T)
func TestHandleFileReceivedWithinRegistrationGraceStillDefers(t *testing.T)
~~~

Assert exact bucket/object, nil only after successful/idempotent deletion, and errors.Is(err, reliability.ErrDeferred) on storage failure.

- [ ] **Step 2: Run RED**

~~~bash
cd omcgo
go test ./internal/pm/collector -run 'TestHandleFileReceived(ExpiredUnregistered|WithinRegistration)' -count=1
~~~

Expected: FAIL because the current collector returns ErrPermanent and never removes the object.

- [ ] **Step 3: Implement the terminal discard path**

Add rawObjectDiscarder to PMCollector, initialize it from the MinIO client, and expose a test setter. Extract:

~~~go
func (c *PMCollector) discardUnregisteredRaw(
    ctx context.Context,
    payload FileReceivedPayload,
) error
~~~

Use payload bucket when present, otherwise the configured PM bucket. RemoveObject success or NoSuchKey increments omc_pm_files_discarded_total{reason="device_not_registered"} and returns nil. NoSuchBucket or any other error wraps reliability.ErrDeferred. Logs contain only a short hash of SN/object.

- [ ] **Step 4: Add the metric and test registration/increment**

~~~go
FilesDiscardedTotal *prometheus.CounterVec
~~~

Use exactly one label, reason.

- [ ] **Step 5: Run GREEN**

~~~bash
cd omcgo
go test ./internal/pm/collector ./internal/pm -count=1
~~~

- [ ] **Step 6: Commit**

~~~bash
git add omcgo/internal/pm/collector omcgo/internal/pm/metrics.go omcgo/internal/pm/metrics_test.go
git commit -m "fix(pm): 丢弃未注册设备非法文件"
~~~

---

### Task 3: Remove Periodic Heartbeat Group-Match Amplification

**Files:**
- Modify: omcgo/internal/device/inform_handler.go
- Modify: omcgo/internal/device/inform_handler_test.go
- Modify: omcgo/cmd/app/provider/modules.go
- Test: omcgo/internal/topology/group_match_engine_test.go

**Interfaces:**
- Consumes: device.registered, device.attributes.changed, rule-edit, and hourly GroupMatchEngine triggers.
- Produces: Periodic Inform path with no topology query or fire-and-forget goroutine.

- [ ] **Step 1: Write RED test**

~~~go
func TestHandlePeriodicDoesNotReassignStableDeviceGroup(t *testing.T) {
    require.NoError(t, handler.handlePeriodic(context.Background(), evt))
    require.Never(t, func() bool { return assigner.Calls() != 0 }, 50*time.Millisecond, 5*time.Millisecond)
}
~~~

Retain/prove tests for registered and attribute-change grouping.

- [ ] **Step 2: Run RED**

~~~bash
cd omcgo
go test ./internal/device -run TestHandlePeriodicDoesNotReassignStableDeviceGroup -count=1
~~~

Expected: FAIL with one assignment.

- [ ] **Step 3: Remove redundant heartbeat grouping**

Delete both triggerGroupAssign calls, the trigger method, heartbeat-only interface/DTO/setter, provider adapter and wiring. Do not change GroupMatchEngine event/cron paths.

- [ ] **Step 4: Run GREEN**

~~~bash
cd omcgo
go test ./internal/device ./internal/topology ./cmd/app/provider -count=1
~~~

- [ ] **Step 5: Commit**

~~~bash
git add omcgo/internal/device/inform_handler.go omcgo/internal/device/inform_handler_test.go omcgo/cmd/app/provider/modules.go omcgo/internal/topology/group_match_engine_test.go
git commit -m "fix(device): 移除周期心跳重复归组"
~~~

---

### Task 4: Make Parameter-Sync Result Counts Incremental

**Files:**
- Modify: omcgo/internal/paramsync/result_processor.go
- Modify: omcgo/internal/paramsync/result_processor_integration_test.go
- Modify: omcgo/internal/paramsync/reconciler.go
- Modify: omcgo/internal/paramsync/metrics.go
- Modify: omcgo/cmd/app/etc/config.prod.yaml
- Modify: omcgo/cmd/app/etc/config.dev.yaml
- Modify: omcgo/cmd/app/etc/config.test.yaml

**Interfaces:**
- Consumes: unique run_id/task_id result insert and durable terminal task validation.
- Produces: advanceRunCountsForInsertedResult and authoritative fallback only at convergence/recovery.

- [ ] **Step 1: Write failing exact-once tests**

~~~go
func TestResultProcessorInsertedResultAdvancesStoredCountsOnce(t *testing.T)
func TestResultProcessorDuplicateResultDoesNotAdvanceStoredCounts(t *testing.T)
func TestResultProcessorIntermediateResultSkipsAuthoritativeAggregate(t *testing.T)
func TestResultProcessorNearCompletionUsesAuthoritativeFallback(t *testing.T)
~~~

Use a transaction recorder or pg_stat_statements substitute in the test fixture to prove the ordinary intermediate path does not run WITH actual AS.

- [ ] **Step 2: Run RED**

~~~bash
cd omcgo
go test ./internal/paramsync -run 'TestResultProcessor(Inserted|Duplicate|Intermediate|NearCompletion)' -count=1
~~~

Expected: FAIL because every unique result calls loadAuthoritativeRunCounts.

- [ ] **Step 3: Implement guarded incremental counts**

After result insertion and terminal durability validation, issue one UPDATE ... RETURNING:

~~~sql
UPDATE parameter_sync_runs
SET terminal_task_count = LEAST(expected_task_count, terminal_task_count + 1),
    processed_task_count = LEAST(expected_task_count, processed_task_count + 1),
    failed_task_count = LEAST(expected_task_count, failed_task_count + $2),
    version = version + 1
WHERE id = $1
RETURNING expected_task_count, terminal_task_count,
          processed_task_count, failed_task_count
~~~

Duplicate results skip it. If counts are invalid, at expected boundary, cancelling, or under recovery, run the authoritative aggregate and repair. Preserve request/run/outbox atomicity.

- [ ] **Step 4: Add low-cardinality mode/duration metrics**

Record result_count_mode={incremental,authoritative} and full-run finalize duration.

- [ ] **Step 5: Set explicit GPV batch size 100**

Add gpv_batch_size: 100 under provision.auto_sync in prod/dev/test and add a config/planner test.

- [ ] **Step 6: Run GREEN**

~~~bash
cd omcgo
go test ./internal/paramsync ./cmd/app/provider ./internal/core/appconfig -count=1
~~~

- [ ] **Step 7: Commit**

~~~bash
git add omcgo/internal/paramsync omcgo/cmd/app/etc/config.prod.yaml omcgo/cmd/app/etc/config.dev.yaml omcgo/cmd/app/etc/config.test.yaml omcgo/cmd/app/provider
git commit -m "perf(paramsync): 增量收敛任务结果计数"
~~~

---

### Task 5: Collapse Full-Sync Cleanup to One Indexed DELETE

**Files:**
- Modify: omcgo/internal/paramsync/result_processor.go
- Modify: omcgo/internal/paramsync/result_processor_integration_test.go
- Create: omcgo/internal/paramsync/full_cleanup_query_test.go

**Interfaces:**
- Consumes: CoverageScope, frozenCoveragePathPredicate, staging values.
- Produces: completeCoveragePredicate([]CoverageScope) sq.Sqlizer and one cleanup DELETE per successful full run.

- [ ] **Step 1: Write failing SQL-shape/integration tests**

Require one DELETE containing device_id, one OR tree for exact/prefix+regex coverage, and the staging NOT EXISTS anti-join. Prove incomplete coverage retains old values.

- [ ] **Step 2: Run RED**

~~~bash
cd omcgo
go test ./internal/paramsync -run 'TestCompleteCoveragePredicate|TestFullSyncCleanup' -count=1
~~~

- [ ] **Step 3: Implement one set-based cleanup**

Build one sq.Or across complete non-empty coverage predicates. Skip when empty. Preserve the LIKE prefix before regex for instance paths.

- [ ] **Step 4: Run GREEN and commit**

~~~bash
cd omcgo
go test ./internal/paramsync -count=1
cd ..
git add omcgo/internal/paramsync
git commit -m "perf(paramsync): 合并全量同步收尾清理"
~~~

---

### Task 6: Add Hourly Revision Preparation and Publication Watermark

**Files:**
- Modify: omcgo/migrations/tsdb/000001_tsdb_schema.sql
- Create: omcgo/internal/pm/stream/publication_repository.go
- Create: omcgo/internal/pm/stream/publication_repository_test.go
- Modify: omcgo/internal/pm/stream/window_repository.go
- Modify: omcgo/internal/pm/stream/window_repository_test.go
- Modify: omcgo/internal/pm/stream/finalizer.go
- Modify: omcgo/internal/pm/stream/finalize_scheduler_test.go
- Modify: omcgo/internal/pm/stream/consumer.go
- Modify: omcgo/internal/pm/stream/result_repository.go
- Modify: omcgo/internal/pm/stream/result_repository_test.go
- Modify: omcgo/internal/pm/stream/rollup_outbox.go
- Create: omcgo/internal/pm/stream/rollup_outbox_test.go
- Modify: omcgo/internal/pm/stream/metrics.go
- Modify: deployments/docker/docker-compose.yml

**Interfaces:**
- Produces:

~~~go
type PublicationKey struct {
    TaskVersionID uuid.UUID
    Granularity   Granularity
    WindowStart   time.Time
}
type PublicationRepository interface {
    RegisterEntity(context.Context, pgx.Tx, PublicationKey) error
    MarkPrepared(context.Context, pgx.Tx, PublicationKey, int) error
    MarkDirty(context.Context, pgx.Tx, PublicationKey) error
    PublishReady(context.Context, time.Time, uint64) ([]PublicationRecord, error)
}
~~~

- [ ] **Step 1: Write RED schema/SQL tests**

Require pm_aggregation_publications with revision, status preparing/published, expected/prepared/dirty counters, watermark/published timestamps, primary key and partial due index. Rebuild uq_pm_aggregation_results_business with revision in the unique key so a prepared revision cannot overwrite the currently published revision.

- [ ] **Step 2: Run RED**

~~~bash
cd omcgo
go test ./cmd/migrate ./internal/pm/stream -run 'Publication|Prepare' -count=1
~~~

- [ ] **Step 3: Implement idempotent publication registration**

First window insert registers one expected entity in the same transaction. Duplicate window open cannot increment twice. First preparation increments prepared once. A contribution to prepared/published state marks that entity dirty and advances revision only once.

- [ ] **Step 4: Split preparation from publication**

~~~go
func (f *Finalizer) PrepareClaimed(ctx context.Context, window WindowRecord, leaseOwner uuid.UUID) error
func (f *Finalizer) PublishRevision(ctx context.Context, publication PublicationRecord) error
~~~

Preparation writes target-revision results and revision-tagged rollup outbox rows but keeps both invisible. ReplaceWindowResults conflicts and stale-result deletion are scoped to the target revision. Publication performs only an indexed readiness check, publication switch, and rollup eligibility update in one short transaction. Redis state is removed only after publication.

- [ ] **Step 5: Prepare at hour end and publish at +12 minutes**

Hourly preparation eligibility: window_end <= now. Publication eligibility: window_end + CloseGrace <= now. Keep 30-second scan cadence, yielding the +12m30s bound. Daily/weekly/monthly also register publications, but prepare and publish in the same due cycle after their existing grace; this gives every granularity one reader contract without changing long-period timing.

- [ ] **Step 6: Add revision behavior tests**

~~~go
func TestPreparedHourlyWindowIsInvisibleBeforeWatermark(t *testing.T)
func TestPublicationSwitchMakesWholeRevisionVisible(t *testing.T)
func TestLateEventBeforeWatermarkRepreparesOnlyDirtyEntity(t *testing.T)
func TestLateEventAfterPublicationCreatesNextRevision(t *testing.T)
func TestDuplicateContributionDoesNotAdvanceRevision(t *testing.T)
~~~

- [ ] **Step 7: Raise bounded preparation concurrency**

Change PM_AGGREGATION_FINALIZE_CONCURRENCY default from 4 to 32. Keep claim batch 32 and TSDB connection budget 96; update budget/config tests.

- [ ] **Step 8: Run GREEN and e2e**

~~~bash
cd omcgo
go test ./internal/pm/stream ./cmd/worker ./cmd/migrate -count=1
go run ./cmd/pm-stream-e2etest
~~~

- [ ] **Step 9: Commit**

~~~bash
git add omcgo/migrations/tsdb/000001_tsdb_schema.sql omcgo/internal/pm/stream omcgo/cmd/worker deployments/docker/docker-compose.yml
git commit -m "feat(pm): 原子发布小时聚合版本"
~~~

---

### Task 7: Make Dashboard and Rollups Read One Published Revision

**Files:**
- Modify: omcgo/internal/pm/adhoc/repository.go
- Modify: omcgo/internal/pm/adhoc/results_query_test.go
- Modify: omcgo/internal/pm/adhoc/handler.go
- Modify: omcgo/internal/pm/adhoc/filter_options_query_test.go
- Modify: omcgo/internal/pm/stream/progress_service.go
- Modify: omcgo/internal/pm/stream/progress_service_test.go
- Test: omcgo/internal/pm/stream/rollup_outbox_test.go

**Interfaces:**
- Consumes: published pm_aggregation_publications.revision.
- Produces: results/count/filter/progress/rollup paths resolving exactly that revision.

- [ ] **Step 1: Write failing query-shape tests**

Every reader joins publication on task version, granularity and window start, requiring publication.status='published' and result.revision=publication.revision. Prohibit per-entity MAX(published_at).

- [ ] **Step 2: Run RED**

~~~bash
cd omcgo
go test ./internal/pm/adhoc ./internal/pm/stream -run 'PublishedRevision|ResultsQuery|FilterOptions|Progress' -count=1
~~~

- [ ] **Step 3: Update all PM entry-matrix reads**

Update Query, Count, filter options, dashboard exports, current daily/weekly progress and rollup selection. Rollup relay may publish only rows whose publication revision is current and published. Preserve permission, dimension, technology, object and time filters.

- [ ] **Step 4: Run GREEN and commit**

~~~bash
cd omcgo
go test ./internal/pm/... -count=1
cd ..
git add omcgo/internal/pm/adhoc omcgo/internal/pm/stream
git commit -m "fix(pm): 统一读取已发布聚合版本"
~~~

---

### Task 8: Full Verification and One Immutable Release Build

**Files:**
- Modify only after a new failing test proves a real regression.
- Record evidence in the existing validation evidence location.

- [ ] **Step 1: Format/static/full tests**

~~~bash
cd omcgo
gofmt -w omcgo/internal/pm/collector/*.go omcgo/internal/pm/*.go \
  omcgo/internal/device/inform_handler*.go omcgo/internal/paramsync/*.go \
  omcgo/internal/pm/stream/*.go omcgo/internal/pm/adhoc/*.go
go vet ./...
go build ./...
go test ./... -count=1
~~~

Expected: PASS. Rerun sandbox-blocked listener tests with permission; never alter tests to hide environment failures.

- [ ] **Step 2: Review complete diff**

~~~bash
git diff origin/main...HEAD --check
git status --short --branch
~~~

Exclude credentials, private config, binaries, temporary evidence and unrelated files.

- [ ] **Step 3: Build release once**

Follow deployments/docker/README.md. Record immutable version, commit and image digests. Testing and MR use the same artifact.

---

### Task 9: Clean Deploy and 20,000-Device Acceptance

**Files:** No source change unless validation produces a failing test first.

- [ ] **Step 1: Stop Compose and remove all prior test data**

Use only documented Compose operations. The user explicitly authorized deletion of all test data. Resolve exact volumes/paths first; never use broad recursive deletion or bare-process restart scripts.

- [ ] **Step 2: Deploy immutable release and check every service**

~~~bash
docker compose -f deployments/docker/docker-compose.yml up -d
docker compose -f deployments/docker/docker-compose.yml ps
~~~

All app/web/worker/ACS versions and image digests must match.

- [ ] **Step 3: Verify the clean deployment contains no historical invalid PM**

The authorized full test-data removal must leave dead_letters, PM objects and aggregation tables empty before workload start. Record the zero baseline. During the new workload, any expired unregistered PM must delete its object and increment the discard metric without creating a dead-letter row.

- [ ] **Step 4: Run workload through one complete hourly boundary**

Observe ACS 503/session/rejections, PM/device/param-sync queues, tasks/runs/outbox/dead letters, K900010006/K900010076, PG/TSDB slow queries and locks, CPU/memory/swap/I/O, hourly publication and current daily/weekly coverage.

- [ ] **Step 5: Enforce exact gates**

~~~text
invalid PM dead-letter increment = 0
PM main/registration-wait steady pending = 0
device-periodic backlog slope <= 0 and historical pending clears < 1h
parameter-sync results clear < 1h after cold-start production stops
server-attributable task expiry < 0.1%
main PG steady CPU < 70% of assigned cores
main PG 5-minute buffer hit > 95%
lock waits = 0
host iowait P95 < 5%
hourly publication <= hour_end + 12m30s
current daily/weekly progress visible on next five-minute refresh
ACS/HTTP 502/503/504 = 0
container restart/OOM = 0
Prometheus business alerts = 0
~~~

A failed gate returns to RED-GREEN-REFACTOR, rebuild, redeploy and repeat. Never clear post-startup data to hide a reproducible defect.

---

### Task 10: MR, Merge, and Post-Merge Verification

- [ ] **Step 1: Re-run full verification and prove deployed commit equals HEAD**

Use Task 8 commands and compare immutable image labels/digests.

- [ ] **Step 2: Push and create MR**

~~~bash
git push -u origin codex/pm-dashboard-snapshot-scan-root-fix
glab mr create --fill --remove-source-branch=false
~~~

MR includes root-cause evidence, schema/data-flow changes, test commands, release version, queue slopes, DB CPU/I/O, hourly timing and rollback.

- [ ] **Step 3: Merge only after required checks pass**

Use normal non-force merge; never bypass hooks/checks.

- [ ] **Step 4: Deploy merged main artifact and repeat smoke/operational gates**

Confirm merged version, non-growing queues, illegal PM discard without DLQ, and the next hourly publication at +12m30s or earlier.

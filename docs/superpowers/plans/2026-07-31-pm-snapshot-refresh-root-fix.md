# PM Snapshot Refresh Root Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate unchanged PM aggregation catalog full scans from worker minute refreshes and Dashboard daily/weekly progress queries without weakening configuration freshness or aggregation correctness.

**Architecture:** `PgTaskRepository` exposes a cheap task-catalog revision. `SnapshotStore` serializes refreshes, caches immutable source versions, and performs a full `LoadMatchable` only when the revision changes or an explicit reload is requested. `ProgressService` shares the same store inside the app process, while worker polling uses conditional refresh and version-change events retain force reload.

**Tech Stack:** Go 1.24, pgx, Squirrel, testify, PostgreSQL, Redis-backed PM streaming aggregation.

## Global Constraints

- SQL must use Squirrel + pgx; no ORM or concatenated SQL.
- Explicit task-version-change events must still force immediate full reload.
- Revision/load failure must preserve the last good atomic snapshot.
- Loaders without revision support must retain legacy full-refresh behavior.
- The 45-day matchable history boundary and hourly/daily/weekly/monthly semantics must remain unchanged.
- No database clearing or production-data mutation is allowed during validation.

---

### Task 1: Conditional SnapshotStore refresh

**Files:**
- Modify: `omcgo/internal/pm/stream/snapshot.go`
- Test: `omcgo/internal/pm/stream/snapshot_test.go`

**Interfaces:**
- Consumes: existing `MatchableLoader.LoadMatchable(context.Context, time.Time)`.
- Produces: `MatchableRevision`, `MatchableRevisionLoader`, and `(*SnapshotStore).Refresh(context.Context) error`.

- [ ] **Step 1: Write failing unchanged-revision and changed-revision tests**

Add a thread-safe fake that implements both loader interfaces and tests equivalent to:

```go
func TestSnapshotRefreshSkipsFullLoadWhenRevisionIsUnchanged(t *testing.T) {
    loader := newRevisionLoader(MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)})
    loader.versions = []*TaskVersionSnapshot{{VersionID: uuid.New(), Members: map[uuid.UUID][]TaskMember{}}}
    store := NewSnapshotStore(loader, nil)

    require.NoError(t, store.Refresh(context.Background()))
    require.NoError(t, store.Refresh(context.Background()))
    require.Equal(t, 1, loader.loadCalls())
    require.Equal(t, 2, loader.revisionCalls())
}

func TestSnapshotRefreshReloadsAfterRevisionChanges(t *testing.T) {
    loader := newRevisionLoader(MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)})
    store := NewSnapshotStore(loader, nil)
    require.NoError(t, store.Refresh(context.Background()))

    loader.setRevision(MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(2, 0)})
    loader.setVersions([]*TaskVersionSnapshot{{VersionID: uuid.New(), Members: map[uuid.UUID][]TaskMember{}}})
    require.NoError(t, store.Refresh(context.Background()))
    require.Equal(t, 2, loader.loadCalls())
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `cd omcgo && go test ./internal/pm/stream -run 'TestSnapshotRefresh(Skips|Reloads)' -count=1`

Expected: compilation fails because `MatchableRevision` and `Refresh` do not exist.

- [ ] **Step 3: Implement the minimal revision-aware state machine**

Add the optional interface and mutex-protected cache:

```go
type MatchableRevision struct {
    TaskCount   int64
    UpdatedAt   time.Time
    Fingerprint string
}

type MatchableRevisionLoader interface {
    LoadMatchableRevision(context.Context) (MatchableRevision, error)
}

func (s *SnapshotStore) Refresh(ctx context.Context) error {
    s.reloadMu.Lock()
    defer s.reloadMu.Unlock()
    revisionLoader, ok := s.loader.(MatchableRevisionLoader)
    if !ok {
        return s.reloadLocked(ctx, nil)
    }
    revision, err := revisionLoader.LoadMatchableRevision(ctx)
    if err != nil {
        return fmt.Errorf("load PM aggregation task revision: %w", err)
    }
    if s.initialized && revision == s.revision {
        if s.nextBoundary.IsZero() || time.Now().UTC().Before(s.nextBoundary) {
            return nil
        }
        s.publishCached(time.Now().UTC())
        return nil
    }
    return s.reloadLocked(ctx, &revision)
}
```

`Reload` must take the same mutex and force `reloadLocked`. `reloadLocked` reads the revision before loading when the caller did not provide one, stores source versions only after successful load, and publishes `BuildTaskSnapshot(snapshotVersionsAt(versions, now))`. `snapshotVersionsAt` excludes versions whose non-nil `EffectiveTo` is before `now-45d` and returns shallow copies so `BuildTaskSnapshot` never mutates cached source objects. Store the earliest future `EffectiveFrom`, `EffectiveTo`, or `EffectiveTo+45d` as `nextBoundary`; unchanged refreshes before that instant return without rebuilding.

- [ ] **Step 4: Add failure, legacy-loader, and concurrent-first-load tests**

Tests must prove revision errors preserve `Current()`, a loader without the optional interface loads on every refresh, 16 concurrent first refreshes result in one `LoadMatchable` call, and an unchanged revision rebuilds exactly once after a controlled version time boundary without calling `LoadMatchable`.

- [ ] **Step 5: Run focused tests and verify GREEN**

Run: `cd omcgo && go test ./internal/pm/stream -run 'TestSnapshot' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add omcgo/internal/pm/stream/snapshot.go omcgo/internal/pm/stream/snapshot_test.go
git commit -m "fix(pm): 缓存未变更的聚合任务快照"
```

### Task 2: PostgreSQL catalog revision

**Files:**
- Modify: `omcgo/internal/pm/stream/task_repository.go`
- Test: `omcgo/internal/pm/stream/task_repository_test.go`

**Interfaces:**
- Consumes: `MatchableRevision` from Task 1.
- Produces: `(*PgTaskRepository).LoadMatchableRevision(context.Context) (MatchableRevision, error)`.

- [ ] **Step 1: Write a failing SQL-contract test**

Extract a pure builder and assert SQL/arguments:

```go
func TestBuildLoadMatchableRevisionSQLIncludesDeletedTasks(t *testing.T) {
    query, args, err := buildLoadMatchableRevisionSQL()
    require.NoError(t, err)
    require.Empty(t, args)
    require.Contains(t, query, "COUNT(*)")
    require.Contains(t, query, "MAX(updated_at)")
    require.NotContains(t, query, "deleted_at")
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd omcgo && go test ./internal/pm/stream -run TestBuildLoadMatchableRevisionSQLIncludesDeletedTasks -count=1`

Expected: compilation fails because the builder is undefined.

- [ ] **Step 3: Implement the revision query with Squirrel + pgx**

Use `storage.Psql.Select("COUNT(*)", "COALESCE(MAX(updated_at), to_timestamp(0))", "COALESCE(md5(string_agg(row_to_json(t)::text, ',' ORDER BY t.id)), md5(''))").From("pm_aggregation_tasks t")`; scan into `TaskCount`, `UpdatedAt`, and `Fingerprint`, and wrap builder/query errors with PM aggregation revision context. The fingerprint is required so an older long transaction that commits after the current maximum timestamp cannot evade change detection.

- [ ] **Step 4: Run repository and package tests**

Run: `cd omcgo && go test ./internal/pm/stream -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add omcgo/internal/pm/stream/task_repository.go omcgo/internal/pm/stream/task_repository_test.go
git commit -m "fix(pm): 增加聚合目录轻量修订水位"
```

### Task 3: Reuse the cache in worker and Dashboard progress

**Files:**
- Modify: `omcgo/internal/pm/stream/snapshot.go`
- Modify: `omcgo/internal/pm/stream/progress_service.go`
- Test: `omcgo/internal/pm/stream/progress_service_test.go`

**Interfaces:**
- Consumes: `SnapshotStore.Refresh` and `SnapshotStore.Current` from Task 1.
- Produces: progress queries that load version details once per revision.

- [ ] **Step 1: Write a failing progress snapshot reuse test**

Add a private `loadSnapshot` method contract and test it without PostgreSQL/Redis:

```go
func TestProgressServiceLoadSnapshotReusesUnchangedCatalog(t *testing.T) {
    loader := newRevisionLoader(MatchableRevision{TaskCount: 1, UpdatedAt: time.Unix(1, 0)})
    service := NewProgressService(nil, nil, loader)

    _, err := service.loadSnapshot(context.Background())
    require.NoError(t, err)
    _, err = service.loadSnapshot(context.Background())
    require.NoError(t, err)
    require.Equal(t, 1, loader.loadCalls())
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd omcgo && go test ./internal/pm/stream -run TestProgressServiceLoadSnapshotReusesUnchangedCatalog -count=1`

Expected: compilation fails because `loadSnapshot` does not exist.

- [ ] **Step 3: Wire ProgressService to one shared SnapshotStore**

Construct `snapshot: NewSnapshotStore(loader, nil)` in `NewProgressService`. Add `NewProgressServiceWithSnapshot` for app startup to inject the already loaded metadata-backfill snapshot. Replace direct `LoadMatchable` and `BuildTaskSnapshot` in `Query` with `loadSnapshot`, and compute metric intervals by iterating `snapshot.ByVersion` while filtering the requested real task ID. Update `cmd/app/provider/pm.go` so startup backfill and Dashboard progress share one store.

- [ ] **Step 4: Change worker polling to conditional refresh**

In `SnapshotStore.RunRefresh`, call `s.Refresh(ctx)` instead of `s.Reload(ctx)`. Do not change worker startup or event callbacks; they continue calling force `Reload`.

- [ ] **Step 5: Run PM stream tests and race tests**

Run: `cd omcgo && go test ./internal/pm/stream -count=1`

Run: `cd omcgo && go test -race ./internal/pm/stream -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add omcgo/internal/pm/stream/snapshot.go omcgo/internal/pm/stream/progress_service.go omcgo/internal/pm/stream/progress_service_test.go
git commit -m "fix(pm): 复用日周进度聚合目录快照"
```

### Task 4: Full verification, MR, deployment, and production proof

**Files:**
- Create: `docs/superpowers/evidence/2026-07-31-pm-snapshot-refresh-root-fix-validation.md`

**Interfaces:**
- Consumes: merged commits from Tasks 1–3.
- Produces: reviewed MR, deployed release, and production acceptance evidence.

- [ ] **Step 1: Run complete backend verification**

Run: `cd omcgo && go build ./...`

Run: `cd omcgo && go test ./... -count=1`

Run: `cd omcgo && go vet ./...`

Run: `git diff --check`

Expected: all commands exit 0.

- [ ] **Step 2: Review the final diff against the design**

Confirm no migration is added, explicit `Reload` callers remain forceful, conditional `Refresh` is the only timer/query path, and error paths never overwrite a good snapshot.

- [ ] **Step 3: Push and create a ready MR**

```bash
git push -u origin codex/pm-snapshot-refresh-root-fix
glab mr create --source-branch codex/pm-snapshot-refresh-root-fix --target-branch main --title "fix(pm): 根治聚合任务快照周期空扫" --description "根治 worker 分钟刷新和 Dashboard 日/周查询对聚合成员目录的重复全量读取。使用轻量任务修订水位与进程内不可变快照缓存；保留显式事件强制刷新、45 天版本范围和全部聚合口径。未清理任何业务数据。"
```

MR evidence must include the production query signature, RED/GREEN tests, full verification, and no-data-clear statement.

- [ ] **Step 4: Merge only after checks/review pass**

Verify MR pipeline/review state, merge without force or skipped checks, and confirm the merge commit is on `origin/main`.

- [ ] **Step 5: Build and deploy the merged immutable release**

Use the repository release workflow and `/opt/omc/current/deploy/healthcheck.sh`; do not use bare-process restart scripts and do not clear PostgreSQL, TSDB, Redis, NATS, or MinIO.

- [ ] **Step 6: Prove production behavior**

For at least three worker refresh minutes and repeated Dashboard daily/weekly queries, verify:

- no repeated `pm_aggregation_version_members` slow query after the first catalog load;
- healthcheck 90/90 and no container restart/OOM;
- ACS 503, admission reject, rate-limit reject, and backpressure reject are zero;
- `pm-workers` and `pm-registration-wait` pending/ack-pending/oldest/redelivery are zero;
- no new dead letters or post-deploy terminal device-task failures;
- K900010006/K900010076 remain fresh and complete;
- the 14:00 window evidence remains first-published at or after 15:12 with complete coverage;
- Dashboard daily and weekly in-progress coverage renders;
- PostgreSQL/TSDB have no lock waiters or active queries over five seconds;
- CPU, memory, disk I/O and Prometheus business alerts remain within established limits.

- [ ] **Step 7: Mark the root goal complete only if every production criterion passes**

If any criterion fails, return to systematic root-cause analysis and a new RED test; never clear data to obtain a green result.

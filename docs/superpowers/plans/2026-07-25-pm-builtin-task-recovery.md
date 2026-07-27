# PM Built-in Task Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the twelve built-in PM aggregation definitions after the streaming migration and automatically materialize idempotent executable versions whenever their resolved membership changes.

**Architecture:** A new idempotent seed migration restores only missing `pm_tasks` definitions and never overwrites edited metrics. The worker runs a built-in-task reconciler at startup and every five minutes; it resolves current rules and members, skips empty membership, and saves a new immutable stream version only when a deterministic content hash differs from the current version.

**Tech Stack:** Go 1.x, pgx/pgxpool, Squirrel SQL builder, PostgreSQL/Goose migrations, TimescaleDB dimension lookup, Zap, Prometheus.

## Global Constraints

- Restore exactly twelve built-in tasks: `network`, `device_group`, `product`, and `band` for LTE, NR, and GSM.
- Preserve the fixed IDs, default metric lists, technology, dimension, `is_builtin=true`, and default `hourly` granularity from `migrations/seed/000001_init_seed.sql`.
- Upgrade initialization inserts missing definitions only and never overwrites user-edited built-in metrics.
- Definitions remain visible when no devices match; no empty executable version is created.
- A changed definition, rule, or sorted membership snapshot creates one version effective at the next complete 15-minute boundary.
- Unchanged content creates no version, including concurrent reconciliation by multiple workers.
- Reconciliation runs once before the initial stream snapshot load and every five minutes afterward.
- Reconciliation failure keeps the last usable version and is retried on the next interval.
- Do not migrate user-created legacy tasks or historical aggregation results and do not backfill history.
- Write all behavior tests before production code, run the new tests once to observe the expected failures, then implement the whole batch and perform centralized verification.

---

### Task 1: Restore the twelve definitions with an incremental seed migration

**Files:**
- Modify: `omcgo/migrations/seed/000001_init_seed.sql`
- Modify: `omcgo/cmd/migrate/pm_streaming_aggregation_migration_test.go`

**Interfaces:**
- Consumes: The existing `public.pm_tasks` schema and the twelve fixed rows in `seed/000001_init_seed.sql`.
- Produces: Twelve visible legacy control-plane definitions for the API and reconciler.

- [ ] **Step 1: Write the failing migration contract test**

Add a test that reads the consolidated seed baseline, verifies every fixed task ID is represented, executes the seed twice when `OMCGO_TEST_DB_DSN` is available, and verifies edited `metric_paths` are unchanged.

```go
func TestBuiltinTaskRecoveryMigrationContract(t *testing.T) {
    sql := readMigration(t, filepath.Join("..", "..", "migrations", "seed", "000001_init_seed.sql"))
    for _, id := range builtinTaskIDs {
        require.Contains(t, sql, id)
    }
    require.Contains(t, sql, "ON CONFLICT (id) DO NOTHING")
    require.NotContains(t, sql, "DO UPDATE")
}
```

- [ ] **Step 2: Run the contract test and verify RED**

Run: `go test ./cmd/migrate -run TestBuiltinTaskRecoveryMigrationContract -count=1`

Expected: FAIL because the consolidated baseline does not yet satisfy the recovery contract.

- [ ] **Step 3: Add the incremental seed migration**

Keep the twelve canonical rows and their complete `pm_tasks` column list in the consolidated seed baseline. Use:

```sql
INSERT INTO public.pm_tasks (...) VALUES
    (...)
ON CONFLICT (id) DO NOTHING;
```

The Down migration must be intentionally non-destructive so rollback cannot remove an edited built-in task:

```sql
-- +goose Down
SELECT 1;
```

- [ ] **Step 4: Defer verification until the centralized test phase**

Do not run the test again here; continue through all implementation tasks.

---

### Task 2: Make immutable stream-version saves content-idempotent

**Files:**
- Modify: `omcgo/migrations/000001_init_schema.sql`
- Create: `omcgo/internal/pm/stream/task_content.go`
- Create: `omcgo/internal/pm/stream/task_content_test.go`
- Modify: `omcgo/internal/pm/stream/task_model.go`
- Modify: `omcgo/internal/pm/stream/task_repository.go`
- Modify: `omcgo/cmd/migrate/pm_streaming_aggregation_migration_test.go`

**Interfaces:**
- Consumes: `stream.SaveTaskRequest`.
- Produces: `ContentHash(req SaveTaskRequest) []byte` and idempotent `PgTaskRepository.Save`; an unchanged save returns the current snapshot without publishing a version-change event.

- [ ] **Step 1: Write deterministic-content failing tests**

Tests must prove that reordered granularities, object LDNs, metric rules, and members hash identically, while changing a rule, device, dimension key, or enabled flag changes the hash.

```go
func TestContentHashIgnoresCollectionOrder(t *testing.T) {
    left := sampleSaveRequest()
    right := permuteSaveRequest(left)
    require.Equal(t, contentHash(left), contentHash(right))
}

func TestContentHashChangesWithExecutableContent(t *testing.T) {
    before := sampleSaveRequest()
    after := before
    after.Members = append([]TaskMember(nil), before.Members...)
    after.Members[0].DimensionKey = "changed"
    require.NotEqual(t, contentHash(before), contentHash(after))
}
```

- [ ] **Step 2: Run stream tests and verify RED**

Run: `go test ./internal/pm/stream -run 'TestContentHash' -count=1`

Expected: FAIL because the hash implementation does not exist.

- [ ] **Step 3: Add the schema field**

Add a nullable `content_hash bytea` column and an index supporting current-version comparison. Existing versions remain readable; the first reconciliation creates a hashed version.

- [ ] **Step 4: Implement canonical hashing**

Canonicalize all executable fields independently of input order, encode with `encoding/json`, and hash with SHA-256. Include `enabled`, technology, dimension, granularities, object LDNs, metric ID/path/type/op, and member device/SN/dimension/object LDN. Exclude `Now`, version IDs, and display-only timestamps.

- [ ] **Step 5: Make Save transactionally idempotent**

Within the existing logical-task `FOR UPDATE` transaction:

1. Compute the content hash.
2. Load the current version and its hash.
3. If equal, commit no writes and load/return that version snapshot.
4. If different or missing, close the prior version, insert the new version with its hash, update `current_version_id`, commit, then publish the change event.

The task row lock must cover comparison and insertion so concurrent workers cannot create duplicate versions.

- [ ] **Step 6: Defer verification until the centralized test phase**

Continue without running green tests yet.

---

### Task 3: Add built-in definition reconciliation

**Files:**
- Create: `omcgo/internal/pm/adhoc/builtin_reconciler.go`
- Create: `omcgo/internal/pm/adhoc/builtin_reconciler_test.go`
- Modify: `omcgo/internal/pm/adhoc/stream_sync.go`
- Modify: `omcgo/internal/pm/adhoc/repository.go`

**Interfaces:**
- Consumes: `PgRepository.List`, existing rule/member resolvers, and `PgTaskRepository.Save`.
- Produces:

```go
type BuiltinReconcileResult struct {
    Definitions int
    Saved       int
    Empty       int
    Failed      int
}

func (r *PgRepository) ReconcileBuiltinStreamingTasks(
    ctx context.Context,
) (BuiltinReconcileResult, error)
```

- [ ] **Step 1: Write failing reconciler behavior tests**

Use narrow injectable collaborators around task listing, rule/member resolution, and stream saving. Tests must prove:

- only `is_builtin=true` continuous definitions are selected;
- empty membership is counted and does not call Save;
- one failing definition does not prevent later definitions from reconciling;
- a non-empty task passes its original fixed ID, edited metrics, technology, dimension, hourly granularity, and resolved members to Save.

```go
func TestBuiltinReconcilerKeepsEmptyDefinitionWithoutSavingVersion(t *testing.T) {
    result, err := reconcilerWithMembers(nil).Reconcile(context.Background())
    require.NoError(t, err)
    require.Equal(t, 1, result.Empty)
    require.Equal(t, 0, result.Saved)
}
```

- [ ] **Step 2: Run reconciler tests and verify RED**

Run: `go test ./internal/pm/adhoc -run TestBuiltinReconciler -count=1`

Expected: FAIL because the reconciler does not exist.

- [ ] **Step 3: Separate empty membership from resolver errors**

Make member resolvers return an empty slice without manufacturing an error. Keep normal user create/update validation by rejecting zero members in `syncStreamingTask`; let the built-in reconciler interpret zero members as a valid inactive definition.

- [ ] **Step 4: Implement reconciliation**

List all built-ins with `IncludeAll=true`, resolve each task’s current metric metadata and members, skip empty membership, and call the idempotent stream repository for non-empty definitions. Continue after per-task failures, aggregate them with task IDs/names, and return the result plus a joined error.

- [ ] **Step 5: Defer verification until the centralized test phase**

Continue to worker wiring.

---

### Task 4: Wire startup and periodic reconciliation with observability

**Files:**
- Modify: `omcgo/cmd/worker/pm_streaming.go`
- Modify: `omcgo/internal/pm/stream/metrics.go`
- Modify: `omcgo/internal/pm/stream/metrics_test.go`
- Create: `omcgo/cmd/worker/pm_builtin_reconcile_test.go`

**Interfaces:**
- Consumes: `adhoc.PgRepository.ReconcileBuiltinStreamingTasks`.
- Produces: immediate startup reconciliation before `SnapshotStore.Reload`, periodic five-minute reconciliation followed by snapshot reload, and counters/gauges for success/failure/empty definitions.

- [ ] **Step 1: Write failing wiring and metric tests**

Tests must prove the interval is five minutes, startup reconciliation happens before the first snapshot load, successful changed reconciliation reloads the snapshot, and errors do not terminate the worker loop.

- [ ] **Step 2: Run focused tests and verify RED**

Run: `go test ./cmd/worker ./internal/pm/stream -run 'Test.*Builtin|TestMetrics' -count=1`

Expected: FAIL because reconciliation is not wired and metrics are absent.

- [ ] **Step 3: Implement startup and loop wiring**

Construct an adhoc repository from the worker’s PostgreSQL and Timescale pools, attach the shared stream task repository, reconcile once, then load the stream snapshot. Start a five-minute ticker which reconciles and reloads the snapshot after successful saves. Log task counts and failures without clearing the prior snapshot.

- [ ] **Step 4: Add low-cardinality metrics**

Expose total reconcile runs, failures, created/changed versions, and definitions skipped for empty membership. Do not label metrics by task ID or name.

- [ ] **Step 5: Defer verification until the centralized test phase**

All production development is now complete; move to centralized verification.

---

### Task 5: Centralized verification and MR preparation

**Files:**
- Modify only files required by failures attributable to this feature.

**Interfaces:**
- Consumes: All changes from Tasks 1–4.
- Produces: A verified branch ready for one MR.

- [ ] **Step 1: Run focused tests**

Run:

```bash
cd omcgo
go test ./cmd/migrate ./internal/pm/stream ./internal/pm/adhoc ./cmd/worker -count=1
```

Expected: PASS.

- [ ] **Step 2: Run full backend verification**

Run:

```bash
cd omcgo
go build ./...
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend and diff verification**

Run:

```bash
cd omcmb
npm run typecheck
cd ..
git diff --check
```

Expected: PASS.

- [ ] **Step 4: Run migration and behavior validation in the compose environment**

Apply schema and seed migrations, then verify:

```sql
SELECT count(*) FROM pm_tasks WHERE is_builtin = true;
-- 12

SELECT count(*)
FROM pm_aggregation_tasks a
JOIN pm_tasks t ON t.id = a.id
WHERE t.is_builtin = true;
-- only built-ins with at least one resolved member have executable tasks
```

Run reconciliation twice without changing devices and verify the version count does not increase. Add or move one matching device and verify exactly one new version appears at the next 15-minute boundary.

- [ ] **Step 5: Review the complete diff**

Confirm there is no legacy history migration, raw PM database scan, empty stream version, unconditional overwrite of edited built-in metrics, or unrelated refactor.

- [ ] **Step 6: Commit and create the MR only after all verification passes**

```bash
git add docs/superpowers/plans/2026-07-25-pm-builtin-task-recovery.md \
  omcgo/migrations/seed/000001_init_seed.sql \
  omcgo/migrations/000001_init_schema.sql \
  omcgo/cmd/migrate/pm_streaming_aggregation_migration_test.go \
  omcgo/internal/pm/stream \
  omcgo/internal/pm/adhoc \
  omcgo/cmd/worker/pm_streaming.go \
  omcgo/cmd/worker/pm_builtin_reconcile_test.go
git commit -m "fix(pm): 恢复内置流式聚合任务"
git push -u origin codex/pm-builtin-task-recovery
glab mr create --target-branch main --title "fix(pm): 恢复内置流式聚合任务"
```

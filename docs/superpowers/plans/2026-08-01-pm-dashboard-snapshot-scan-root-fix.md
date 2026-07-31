# PM Dashboard Snapshot Scan Root Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent unchanged PM catalog reconciliation and Dashboard progress reads from scanning all aggregation-version members.

**Architecture:** Make task persistence idempotent at SQL level, compute catalog revisions from semantic fields, and inject a member-free catalog loader into the app-side progress service while preserving the worker's full matcher snapshot.

**Tech Stack:** Go, pgx, Squirrel, PostgreSQL, Redis, Docker Compose.

## Global Constraints

- Dashboard refresh remains timer-only every 5 minutes.
- Hourly, daily, and weekly series continue reading existing aggregation output.
- Current daily and weekly results retain progress and coverage metadata.
- No raw PM fallback or online KPI aggregation is introduced.

---

### Task 1: Idempotent task persistence and semantic catalog revision

**Files:**
- Modify: `omcgo/internal/pm/stream/task_repository.go`
- Test: `omcgo/internal/pm/stream/task_repository_test.go`

**Interfaces:**
- Produces: unchanged `PgTaskRepository.Save` and `LoadMatchableRevision` public signatures.

- [ ] Add failing SQL-builder tests proving unchanged task fields are protected by `IS DISTINCT FROM` and revision SQL excludes `updated_at`/`row_to_json` while including current-version semantics.
- [ ] Run `go test ./internal/pm/stream -run 'TestBuild(UpdateTaskMetadata|LoadMatchableRevision)SQL'` and confirm failure.
- [ ] Extract the guarded update SQL builder and replace the revision query with the semantic task/current-version fingerprint.
- [ ] Rerun the focused tests and confirm success.

### Task 2: Member-free progress catalog

**Files:**
- Modify: `omcgo/internal/pm/stream/task_repository.go`
- Modify: `omcgo/cmd/app/provider/pm.go`
- Test: `omcgo/internal/pm/stream/task_repository_test.go`
- Test: `omcgo/cmd/app/provider/pm_test.go`

**Interfaces:**
- Produces: `NewPgProgressTaskLoader(pool)` implementing `MatchableLoader` and `MatchableRevisionLoader` without querying version members.
- Consumes: existing `SnapshotStore` and `ProgressService` contracts.

- [ ] Add failing tests proving the progress loader excludes member details and the app provider selects it.
- [ ] Run the focused stream/provider tests and confirm failure.
- [ ] Split matchable detail loading into rule/counter loading plus optional member loading, add the progress loader, and wire it into app startup.
- [ ] Rerun the focused tests and confirm success.

### Task 3: Verification, deployment, and production acceptance

**Files:**
- Modify: deployment validation documentation only if observed behavior needs recording.

**Interfaces:**
- Consumes: Tasks 1-2 code and the existing Docker Compose release workflow.

- [ ] Run `gofmt` on changed Go files.
- [ ] Run `cd omcgo && go test ./internal/pm/stream ./cmd/app/provider`, `go build ./...`, and `go test ./...`.
- [ ] Review the complete diff for scope, migration compatibility, and unchanged Dashboard behavior.
- [ ] Stop the target OMC Compose stack, resolve its exact project volumes, remove all application data authorized by the user, build and deploy the new release.
- [ ] Verify services, ACS, queues, dead letters, PM/KPI freshness, K900010006/K900010076, PostgreSQL/TSDB locks and slow queries, CPU/memory/disk I/O, and hourly/daily/weekly coverage.
- [ ] Cross a 5-minute reconcile boundary, query Dashboard daily/weekly repeatedly, and prove no app/worker member-table full scan occurs.
- [ ] Commit, push, create MR, wait for pipeline, merge, and confirm merged main contains the fix.


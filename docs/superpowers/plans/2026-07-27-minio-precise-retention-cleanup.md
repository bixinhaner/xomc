# MinIO Precise Retention Cleanup Implementation Plan

> **For Codex:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace full-bucket ILM expiry scans for PM/MR raw objects with a continuously rate-limited worker that deletes exact database paths, scales with device/file volume, and yields to production load.

**Architecture:** A new `rawcleanup` package owns candidate selection, exact object deletion, retry state, rate calculation, pressure protection, advisory locking, and metrics. The worker runs one batch at a time; the app keeps or removes OMC lifecycle rules according to a rollout mode (`shadow`, `fallback`, `exclusive`). No database transaction spans a MinIO request and no MinIO object listing is used.

**Tech Stack:** Go, pgx, Squirrel, minio-go v7, NATS JetStream, Prometheus HTTP API, PostgreSQL/TimescaleDB migrations, Docker Compose.

---

### Task 1: Persist precise deletion state and rollout configuration

**Files:**
- Create: `omcgo/migrations/tsdb/000002_raw_object_cleanup.sql`
- Create: `omcgo/migrations/seed/000003_raw_object_cleanup_mode.sql`
- Modify: `omcgo/internal/core/appconfig/config.go`
- Modify: `omcgo/cmd/worker/etc/config.dev.yaml`
- Modify: `omcgo/cmd/worker/etc/config.test.yaml`
- Modify: `omcgo/cmd/worker/etc/config.prod.yaml`
- Test: `omcgo/internal/core/appconfig/config_test.go`

**Steps:**
1. Add a failing configuration-default test for batch size 100, rates 5–100/s, 1.5 headroom, 5-minute recalculation, 10-second object timeout, and safe `shadow` default.
2. Add `raw_deleted_at`, retry fields, candidate partial indexes, and `mr_files(created_at)` index to the baseline TSDB schema.
3. Add `RawCleanupConfig` to `WorkerConfig`, including a `Defaults()` normalizer and the three rollout modes.
4. Add explicit raw-cleanup settings to all worker YAML profiles.
5. Run the focused configuration test.

### Task 2: Implement and test automatic rate and pressure governance

**Files:**
- Create: `omcgo/internal/rawcleanup/model.go`
- Create: `omcgo/internal/rawcleanup/governor.go`
- Create: `omcgo/internal/rawcleanup/governor_test.go`
- Create: `omcgo/internal/rawcleanup/pressure.go`
- Create: `omcgo/internal/rawcleanup/pressure_test.go`

**Steps:**
1. Write failing table tests for predicted/observed maximum, headroom, min/max clamps, 10k-device 15-minute traffic, and batch interval.
2. Implement the pure rate calculation:
   `clamp(ceil(max(predicted, observed)*headroom), min, max)`.
3. Write failing tests for PM boundary slowdown, CPU/disk reduction, hard disk/NATS pause, slow MinIO batch reduction, and unavailable-monitoring 20/s cap.
4. Implement pressure evaluation as pure logic with a structured decision/reason.
5. Add a Prometheus pressure probe for CPU and disk await/queue. Invalid, stale, or unreachable responses return “unavailable”, never fabricated healthy values.

### Task 3: Implement exact candidate selection, deletion, and retry state

**Files:**
- Create: `omcgo/internal/rawcleanup/repository.go`
- Create: `omcgo/internal/rawcleanup/repository_test.go`
- Create: `omcgo/internal/rawcleanup/deleter.go`
- Create: `omcgo/internal/rawcleanup/deleter_test.go`

**Steps:**
1. Write failing repository tests that inspect generated/issued SQL for retention cutoff, retry readiness, deterministic keyset order, and partial-success updates.
2. Implement a dedicated pgx repository using Squirrel. Select at most one configured batch across PM/MR, count files created in the last hour, calculate oldest-expired age, and write per-object success/failure state.
3. Write failing deleter tests for exact PM/MR bucket routing, partial success, missing-object idempotency, error truncation, and exponential retry capped at one hour.
4. Implement MinIO `RemoveObjects` batching over exact keys only. Never call `ListObjects`.
5. Add dependency-safe, bounded metadata cleanup queries; retain `pm_files` while anchors/records or retained time-series dependencies still exist.

### Task 4: Implement the single-runner loop and observability

**Files:**
- Create: `omcgo/internal/rawcleanup/runner.go`
- Create: `omcgo/internal/rawcleanup/runner_test.go`
- Create: `omcgo/internal/rawcleanup/metrics.go`
- Create: `omcgo/cmd/worker/raw_cleanup.go`
- Modify: `omcgo/cmd/worker/main.go`
- Modify: `omcgo/cmd/app/provider/modules.go`
- Delete: `omcgo/internal/mr/task/cleaner.go`
- Delete: `omcgo/internal/mr/task/cleaner_test.go`
- Modify: `omcgo/internal/mr/store.go`
- Modify: `omcgo/internal/mr/pg_store.go`

**Steps:**
1. Write failing runner tests for shadow mode, one-batch-at-a-time behavior, advisory-lock exclusion, exact post-delete status updates, context cancellation, and conservative monitoring fallback.
2. Implement a continuously running loop with startup jitter, session advisory lock, rate recalculation, batch timeout, exponential MinIO backoff, and per-round wait.
3. Add the six approved Prometheus metrics and stable low-cardinality pause/failure reasons.
4. Wire the runner to the worker TSDB, MinIO client, current retention sys-config, recent file rate, device-count prediction, NATS pressure, and Prometheus pressure probe.
5. Remove the old daily MR metadata cleaner so it cannot delete `mr_files` rows before exact object cleanup.

### Task 5: Make ILM exit staged and lower scanner priority

**Files:**
- Modify: `omcgo/internal/core/components/minio/minio.go`
- Modify: `omcgo/internal/core/components/minio/minio_test.go`
- Modify: `omcgo/cmd/app/provider/minio_ilm.go`
- Modify: `omcgo/cmd/app/provider/minio_ilm_test.go`
- Modify: `deployments/docker/docker-compose.yml`
- Modify: `deployments/docker/.env.example`

**Steps:**
1. Write failing tests that `shadow`/`fallback` apply the OMC raw lifecycle rule while `exclusive` removes only OMC-owned raw-expiry rules and preserves unrelated lifecycle rules.
2. Implement lifecycle reconciliation and hot reload for `minio.retention.cleanup_mode`.
3. Change `MINIO_SCANNER_SPEED` default to `slowest`; do not disable MinIO scanner.
4. Keep `raw_object_days` as the only raw object retention duration and do not change TimescaleDB retention configuration.

### Task 6: Unified verification and MR

**Files:**
- Modify: `docs/superpowers/specs/2026-07-27-minio-precise-retention-cleanup-design.md`

**Steps:**
1. Update the design status to implemented and record the rollout controls.
2. Run formatting once for all changed Go files.
3. Run the focused `rawcleanup`, MinIO, provider, appconfig, worker, PM, and MR tests.
4. Run the complete backend gates: `go build ./...` and `go test ./...`.
5. Review the complete diff for accidental schema/data-retention changes, object listing, unbounded deletes, or unrelated edits.
6. Commit with a Conventional Commit message, push the branch, and create a ready MR only after all verification passes.

# PM Slot Health and MinIO Headroom Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist bounded 15-minute PM slot health summaries, expose reliable ingestion/coverage metrics and alerts, and give MinIO sufficient memory headroom without masking leaks.

**Architecture:** The PM copy-ingest transaction records the XML measurement window on `pm_files`. A worker-side observer evaluates only the latest eligible slot after the 12-minute grace, joins durable ingested-device counts with the active aggregation-task membership snapshot, persists one summary row per technology/carrier, and exports low-cardinality gauges. The dashboard and Prometheus read this summary rather than scanning PM metric detail. MinIO receives a 6 GiB limit on the 32 GiB profile plus pressure-aware alerts that distinguish process RSS from reclaimable cgroup working set.

**Tech Stack:** Go 1.x, pgx, Squirrel SQL builder, PostgreSQL/TimescaleDB, Prometheus/promtool, Docker Compose, React/TypeScript.

## Global Constraints

- All SQL must use Squirrel plus pgx; no ORM or interpolated SQL.
- Database changes must be folded into `omcgo/migrations/000001_init_schema.sql` or `omcgo/migrations/tsdb/000001_tsdb_schema.sql`; no `000002+` migration.
- PM slot evaluation uses the existing 15-minute source cadence and the configured 12-minute close grace.
- Metrics labels are limited to `technology` and `carrier`; slot timestamps and device identifiers are metric values, never labels.
- Dashboard reads `pm_slot_health`; it must not scan `pm_measurement_anchors`, `pm_metric_values`, or compatibility views.
- User-visible frontend text must use i18n and browser validation.
- Startup-crossing slots are recorded as `bootstrap_ignored`, not treated as complete and not alerted.

---

### Task 1: Persist the XML measurement window atomically

**Files:**
- Modify: `omcgo/migrations/tsdb/000001_tsdb_schema.sql`
- Modify: `omcgo/internal/pm/metrics/copy_ingest.go`
- Modify: `omcgo/internal/pm/metrics/copy_ingest_test.go`
- Modify: `omcgo/internal/pm/collector/collector.go`
- Modify: `omcgo/internal/pm/collector/result_normalization_test.go`

**Interfaces:**
- Produces: `FileMarker.MeasurementStart time.Time` and `FileMarker.MeasurementEnd time.Time`.
- Produces: indexed `pm_files.measurement_start` and `pm_files.measurement_end` columns.

- [ ] Write a failing copy-ingest SQL test proving both XML window timestamps are inserted independently from upload `collect_time`.
- [ ] Run the focused metrics test and confirm it fails because the fields/columns do not exist.
- [ ] Add the baseline columns/index and pass `PMFileContent.FileBeginTime/FileEndTime` into the marker.
- [ ] Run collector and metrics tests and confirm the measurement window is preserved.

### Task 2: Evaluate and persist bounded PM slot health

**Files:**
- Create: `omcgo/internal/pm/slothealth/model.go`
- Create: `omcgo/internal/pm/slothealth/repository.go`
- Create: `omcgo/internal/pm/slothealth/observer.go`
- Create: `omcgo/internal/pm/slothealth/observer_test.go`
- Create: `omcgo/internal/pm/slothealth/repository_test.go`
- Modify: `omcgo/migrations/tsdb/000001_tsdb_schema.sql`

**Interfaces:**
- Produces: `slothealth.Observer.Observe(context.Context) error` and `Run(context.Context)`.
- Produces: `slothealth.Snapshot` with slot, technology, carrier, expected, received, coverage, snapshot version, status, and evaluation time.
- Consumes: main DB active task versions/members at slot start and TSDB `pm_files.measurement_end` receipts.

- [ ] Write failing pure tests for latest eligible slot, 12-minute boundary, coverage classification, and startup-slot suppression.
- [ ] Run the tests and confirm the missing observer behavior causes the expected failure.
- [ ] Implement the pure slot and status logic.
- [ ] Write failing repository tests proving expected membership and received-device queries are bounded to one slot and use Squirrel placeholders.
- [ ] Implement the pgx/Squirrel repository and idempotent `pm_slot_health` upsert.
- [ ] Run the package tests and confirm all observer/repository cases pass.

### Task 3: Export stage timestamps and slot gauges

**Files:**
- Modify: `omcgo/internal/pm/metrics.go`
- Modify: `omcgo/internal/pm/metrics_test.go`
- Modify: `omcgo/internal/pm/collector/collector.go`
- Modify: `omcgo/internal/acs/upload/backpressure.go`
- Modify: `omcgo/internal/acs/upload/backpressure_test.go`
- Modify: `omcgo/internal/acs/upload/handler.go`
- Modify: `omcgo/internal/acs/upload/handler_test.go`
- Modify: `omcgo/cmd/worker/main.go`

**Interfaces:**
- Produces: `acs_pm_upload_last_accepted_timestamp_seconds`.
- Produces: `omc_pm_ingest_last_success_timestamp_seconds{technology,carrier}`.
- Produces: latest slot expected/received/coverage/end/evaluation gauges by technology/carrier.

- [ ] Write failing registry tests for all metric names and bounded label sets.
- [ ] Write a failing handler test proving rejected/failed uploads do not advance the accepted timestamp while a stored PM upload does.
- [ ] Implement ACS accepted and Worker successful-ingest timestamp updates.
- [ ] Write a failing observer metric test proving one observation replaces the current fixed-label gauge without a slot label.
- [ ] Implement metric publication and wire the observer into worker graceful shutdown.
- [ ] Run ACS upload, PM root, collector, slothealth, and worker package tests.

### Task 4: Read slot summaries from the dashboard without raw scans

**Files:**
- Modify: `omcgo/internal/dashboard/service.go`
- Modify: `omcgo/internal/dashboard/service_test.go`
- Modify: `omcgo/internal/dashboard/handler_test.go`
- Modify: `omcmb/frontend-core/src/types/dashboard.ts`
- Modify: `omcmb/webcode/src/pages/Dashboard/index.tsx`
- Modify: `omcmb/webcode/src/pages/Dashboard/index.test.tsx`
- Modify: relevant locale files discovered from the existing dashboard namespace.

**Interfaces:**
- Produces: `DashboardSummary.pm_slot_health`, sourced only from the latest `pm_slot_health` rows.

- [ ] Write a failing service test that rejects any summary query containing raw PM tables and expects the newest health row per technology/carrier.
- [ ] Implement the bounded summary query and response DTO.
- [ ] Write a failing frontend test for coverage/status display and no-data behavior.
- [ ] Implement the small homepage health card using existing dashboard refresh only and i18n text.
- [ ] Run focused Go and frontend tests.
- [ ] Validate the real page and `/api/v1/dashboard/summary` request in a browser after deployment.

### Task 5: Add actionable PM alerts

**Files:**
- Modify: `deployments/monitoring/alerts/omc-rules.yml`
- Create: `deployments/monitoring/alerts/pm-slot-health.test`

**Interfaces:**
- Consumes: ACS accepted timestamp, Worker ingest timestamp, observer freshness, expected count, and slot coverage gauges.

- [ ] Add promtool fixtures that fail because stalled uploads, ACS-to-Worker lag, stale observation, zero coverage, and low coverage are not yet alerted.
- [ ] Add warning/critical rules with active-device and startup guards.
- [ ] Run `promtool test rules` and alert syntax validation.

### Task 6: Give MinIO headroom and distinguish cache pressure from process growth

**Files:**
- Modify: `deployments/docker/plan-resources.sh`
- Modify: `deployments/release/bundle/deploy/plan-resources.sh`
- Modify: `deployments/release/bundle/deploy/docker-compose.infra.yml`
- Modify: `deployments/monitoring/alerts/host-container-alerts.yml`
- Modify: `deployments/monitoring/alerts/resource-plan-alerts.test`
- Modify: relevant resource-plan shell tests.

**Interfaces:**
- Produces: 6 GiB MinIO memory cap for the 32 GiB profile, capped at 8 GiB on larger hosts.
- Produces: MinIO working-set warning/critical and process-RSS growth/5xx/OOM coverage without weakening generic container alerts.

- [ ] Update behavioral resource-plan tests to expect 6 GiB at 32 GiB and confirm smaller profiles retain safe floors.
- [ ] Run the tests and confirm the old 4 GiB plan fails.
- [ ] Implement matching development and release resource formulas/defaults.
- [ ] Add promtool scenarios for MinIO reclaim pressure and process growth.
- [ ] Implement MinIO-specific alerts while keeping OOM critical.
- [ ] Run resource-plan, Compose rendering, promtool, and monitoring validation tests.

### Task 7: Full verification, deployment, and live acceptance

**Files:**
- Modify: `docs/superpowers/evidence/2026-08-03-pm-slot-health-minio-headroom.md`

**Interfaces:**
- Consumes all tasks above; produces reproducible test and live evidence.

- [ ] Run formatting, complete affected Go tests, frontend typecheck/tests, shell tests, Prometheus rule tests, and release build checks.
- [ ] Review the diff against every requirement and confirm no dashboard raw scan or high-cardinality label exists.
- [ ] Build the amd64 release, verify tarball checksum/images/VERSION, upload it, and validate the remote checksum.
- [ ] Deploy to `172.24.224.197`, run health checks, and confirm the MinIO limit is 6 GiB.
- [ ] Validate a real PM batch: ACS accepted timestamp, Worker success timestamp, one closed slot after +12 minutes, coverage values, queues, KPI freshness, no 503, no OOM, and dashboard response/page.
- [ ] Record CPU, memory, MinIO RSS/working-set, memory pressure, disk I/O, PostgreSQL/TSDB locks and slow queries, and Prometheus alerts.
- [ ] Commit with a Chinese Conventional Commit message, push the branch, create the MR, and merge only after all checks are green.

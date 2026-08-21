# Consolidate OMC Migrations Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace all incremental SQL migrations under `omcgo/migrations` with one final-state `000001` baseline per independent Goose stream.

**Architecture:** Apply every current migration to disposable PostgreSQL and TimescaleDB containers on an isolated Docker network. Regenerate the main schema and seed baselines from final PostgreSQL dumps, integrate the four TSDB deltas into its explicit baseline, delete superseded SQL, then compare fresh-baseline databases with the golden state.

**Tech Stack:** PostgreSQL 16, TimescaleDB 2.25.2-pg16, Goose v3 through `omcgo-migrate`, Docker, `pg_dump`, SQL.

## Global Constraints

- Keep exactly `omcgo/migrations/000001_init_schema.sql`, `omcgo/migrations/seed/000001_init_seed.sql`, and `omcgo/migrations/tsdb/000001_tsdb_schema.sql`.
- Keep schema, seed, and TSDB as independent Goose streams and version tables.
- Target clean installations; versions above `000001` are not upgrade-compatible.
- Do not start application services against disposable databases.
- Do not modify existing developer containers or volumes; disposable databases use dedicated names, a dedicated network, no host ports, and tmpfs storage.
- Do not export TimescaleDB extension catalog objects into repository SQL.
- Keep scripts outside `omcgo/migrations` out of scope.

---

### Task 1: Capture the current golden final state

**Files:**
- Read: `deployments/docker/Dockerfile.app`
- Read: `omcgo/migrations/**/*.sql`
- Create outside repository: `/tmp/omc-migration-baseline/*`

**Interfaces:**
- Consumes: 24 schema, 13 seed, and 5 TSDB migration files.
- Produces: golden main schema/data dumps and TSDB inventories.

- [ ] **Step 1: Assert the source inventory**

Run `find omcgo/migrations -type f -name '*.sql' -print | sort`.

Expected: 42 files: 24 schema, 13 seed, and 5 TSDB.

- [ ] **Step 2: Build the migrator and isolated Docker resources**

Run:

```bash
mkdir -p /tmp/omc-migration-baseline
docker build -f deployments/docker/Dockerfile.app -t omcgo-migration-baseline:20260716 .
docker network create --driver bridge --subnet 173.21.0.0/16 --gateway 173.21.0.1 omc-migration-baseline
docker run --rm -d --name omc-baseline-main-golden --network omc-migration-baseline --tmpfs /var/lib/postgresql/data:rw -e POSTGRES_USER=omcgo -e POSTGRES_PASSWORD=omcgo123 -e POSTGRES_DB=omcgo postgres:16-alpine
docker run --rm -d --name omc-baseline-tsdb-golden --network omc-migration-baseline --tmpfs /var/lib/postgresql/data:rw -e POSTGRES_USER=omcgo -e POSTGRES_PASSWORD=omcgo123 -e POSTGRES_DB=omcgo timescale/timescaledb:2.25.2-pg16
```

Poll `docker exec <name> pg_isready -U omcgo -d omcgo` until both report `accepting connections`.

- [ ] **Step 3: Apply every migration with separate version tables**

Run:

```bash
docker run --rm --network omc-migration-baseline -v "$PWD/omcgo/migrations:/etc/omcgo/migrations:ro" --entrypoint omcgo-migrate omcgo-migration-baseline:20260716 --dsn 'postgres://omcgo:omcgo123@omc-baseline-main-golden:5432/omcgo?sslmode=disable' --path /etc/omcgo/migrations up
docker run --rm --network omc-migration-baseline -v "$PWD/omcgo/migrations:/etc/omcgo/migrations:ro" --entrypoint omcgo-migrate omcgo-migration-baseline:20260716 --dsn 'postgres://omcgo:omcgo123@omc-baseline-main-golden:5432/omcgo?sslmode=disable' --path /etc/omcgo/migrations/seed --table goose_db_version_seed up
docker run --rm --network omc-migration-baseline -v "$PWD/omcgo/migrations:/etc/omcgo/migrations:ro" --entrypoint omcgo-migrate omcgo-migration-baseline:20260716 --dsn 'postgres://omcgo:omcgo123@omc-baseline-tsdb-golden:5432/omcgo?sslmode=disable' --path /etc/omcgo/migrations/tsdb --table goose_db_version_tsdb up
```

Expected versions: `24`, `13`, and `5`.

- [ ] **Step 4: Capture deterministic golden outputs**

Run:

```bash
docker exec omc-baseline-main-golden pg_dump -U omcgo -d omcgo --schema-only --no-owner --no-privileges --no-tablespaces --no-publications --no-subscriptions --exclude-table='goose_db_version*' > /tmp/omc-migration-baseline/golden-main-schema.sql
docker exec omc-baseline-main-golden pg_dump -U omcgo -d omcgo --data-only --inserts --rows-per-insert=500 --on-conflict-do-nothing --no-owner --no-privileges --disable-triggers --exclude-table='goose_db_version*' > /tmp/omc-migration-baseline/golden-main-seed.sql
docker exec omc-baseline-tsdb-golden psql -U omcgo -d omcgo -Atc "SELECT table_schema, table_name, column_name, data_type, is_nullable, coalesce(column_default, '') FROM information_schema.columns WHERE table_schema = 'public' ORDER BY table_name, ordinal_position" > /tmp/omc-migration-baseline/golden-tsdb-columns.txt
docker exec omc-baseline-tsdb-golden psql -U omcgo -d omcgo -Atc "SELECT schemaname, tablename, indexname, indexdef FROM pg_indexes WHERE schemaname = 'public' ORDER BY tablename, indexname" > /tmp/omc-migration-baseline/golden-tsdb-indexes.txt
docker exec omc-baseline-tsdb-golden psql -U omcgo -d omcgo -Atc "SELECT view_schema, view_name, materialized_only FROM timescaledb_information.continuous_aggregates ORDER BY view_schema, view_name" > /tmp/omc-migration-baseline/golden-tsdb-caggs.txt
```

Expected: all outputs are non-empty and cagg inventory contains `pm_metrics_hourly_cagg`.

### Task 2: Regenerate main schema and seed baselines

**Files:**
- Modify: `omcgo/migrations/000001_init_schema.sql`
- Modify: `omcgo/migrations/seed/000001_init_seed.sql`

**Interfaces:**
- Consumes: Task 1 main schema and seed dumps.
- Produces: two direct final-state Goose baselines.

- [ ] **Step 1: Normalize the schema dump into the baseline**

Remove any `\\restrict` and `\\unrestrict` lines. Prefix the dump with:

```sql
-- +goose Up
-- 主库 consolidated baseline（2026-07-16；纯 PostgreSQL 16；时序对象位于 migrations/tsdb）。
```

Wrap each PL/pgSQL function in matching Goose StatementBegin/StatementEnd directives. Restore `search_path` and add the Down section:

```sql
SELECT pg_catalog.set_config('search_path', 'public', false);
-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
-- +goose StatementEnd
```

Expected: direct final definitions with no historical version comments or appended ALTER chain.

- [ ] **Step 2: Normalize the data dump into the seed baseline**

Remove any `\\restrict` and `\\unrestrict` lines. Prefix the dump with:

```sql
-- +goose Up
-- 主库 consolidated seed baseline（2026-07-16；由完整迁移后的干净库最终态导出）。
```

Restore `search_path` and add:

```sql
SELECT pg_catalog.set_config('search_path', 'public', false);
-- +goose Down
-- consolidated seed 无安全的逐行回滚；重置请重建数据库。
SELECT 1;
```

Expected: final data only; generated inserts use untargeted `ON CONFLICT DO NOTHING`.

- [ ] **Step 3: Check directives and whitespace**

Run `rg -n '^-- \+goose (Up|Down|StatementBegin|StatementEnd)$'` on both files and `git diff --check -- omcgo/migrations/000001_init_schema.sql omcgo/migrations/seed/000001_init_seed.sql`.

Expected: one Up and Down per file, balanced StatementBegin/End counts, silent whitespace check.

### Task 3: Fold TSDB deltas into its explicit baseline

**Files:**
- Modify: `omcgo/migrations/tsdb/000001_tsdb_schema.sql`
- Read: `omcgo/migrations/tsdb/000002_create_pm_metrics_hourly_cagg.sql`
- Read: `omcgo/migrations/tsdb/000003_allow_pm_metrics_metric_value_null.sql`
- Read: `omcgo/migrations/tsdb/000004_allow_pm_rollup_metric_value_null.sql`
- Read: `omcgo/migrations/tsdb/000005_allow_pm_adhoc_metric_value_null.sql`

**Interfaces:**
- Consumes: four TSDB increments and Task 1 inventories.
- Produces: explicit version-5 final state in `000001`.

- [ ] **Step 1: Encode final metric nullability**

Remove `NOT NULL` from `metric_value` in `pm_metrics`, the four `pm_metrics_*` rollups, the four `pm_group_metrics_*` rollups, and `pm_adhoc_aggregation_results`.

Expected: these ten columns match version 5 without later ALTER statements.

- [ ] **Step 2: Integrate the continuous aggregate**

Insert the exact `public.pm_metrics_hourly_cagg` definition and `add_continuous_aggregate_policy` call from `000002` after the `pm_metrics` hypertable setup, inside matching StatementBegin/End directives. In Down, remove its policy and materialized view before dropping `pm_metrics`.

Expected: Up and Down are dependency-safe and contain no TimescaleDB internal catalog DDL.

- [ ] **Step 3: Check final definitions**

Run `rg -n 'pm_metrics_hourly_cagg|metric_value'`, compare StatementBegin/End counts, and run `git diff --check` on the TSDB baseline.

Expected: cagg has Up/Down handling, ten columns are nullable, directives balance, whitespace check is silent.

### Task 4: Delete increments and update documentation

**Files:**
- Delete: schema `000002` through `000024`
- Delete: seed `000002` through `000013`
- Delete: TSDB `000002` through `000005`
- Modify: `omcgo/migrations/README.md`

**Interfaces:**
- Consumes: three completed baselines.
- Produces: exactly three SQL files and accurate operator guidance.

- [ ] **Step 1: Delete the 39 superseded SQL files with `apply_patch`**

Expected: `find omcgo/migrations -type f -name '*.sql' -print | sort` lists only the three version-1 files.

- [ ] **Step 2: Align README with the new baseline**

Record consolidation date `2026-07-16`, included ranges schema `000001..000024`, seed `000001..000013`, TSDB `000001..000005`, clean-install compatibility, and next version `000002` for each independent stream.

Expected: README state, inventory, and next-version guidance match the filesystem.

- [ ] **Step 3: Check the scoped diff**

Run `git diff --check -- omcgo/migrations`, `find omcgo/migrations -type f -name '*.sql' -print | sort`, and `git status --short`.

Expected: no whitespace errors, exactly three SQL files, unrelated `.codex/` untouched.

### Task 5: Verify fresh install and final-state equivalence

**Files:**
- Test: all three remaining `000001` files
- Test: backend packages under `omcgo/`

**Interfaces:**
- Consumes: consolidated baselines and Task 1 golden outputs.
- Produces: executable evidence of equivalence and clean install.

- [ ] **Step 1: Start fresh disposable `omc-baseline-main-final` and `omc-baseline-tsdb-final` containers**

Use the Task 1 `docker run` commands with the final names.

Expected: both databases become ready without host ports or persistent volumes.

- [ ] **Step 2: Apply the three baselines**

Use the Task 1 migrator commands against final container names.

Expected: all three streams report version `1`.

- [ ] **Step 3: Compare main final state**

Repeat the Task 1 dumps, normalize only `Dumped from/by` header lines, and run `diff -u` against golden schema and seed dumps.

Expected: both diffs are empty.

- [ ] **Step 4: Compare TSDB final state**

Repeat the Task 1 column, index, and cagg inventory queries and run `diff -u` against golden inventories.

Expected: all three diffs are empty.

- [ ] **Step 5: Verify Down**

Run Goose `down` for seed, TSDB, and schema in that order.

Expected: commands exit zero, cagg is absent, and main `public` schema is empty.

- [ ] **Step 6: Run backend verification**

Run:

```bash
cd omcgo
go build ./...
go test ./...
```

Expected: both exit zero; if sandbox blocks local listeners, rerun with repository-prescribed escalation.

- [ ] **Step 7: Remove only disposable resources**

Run:

```bash
docker stop omc-baseline-main-golden omc-baseline-tsdb-golden omc-baseline-main-final omc-baseline-tsdb-final
docker network rm omc-migration-baseline
```

Expected: only dedicated `omc-baseline-*` resources are removed.

- [ ] **Step 8: Commit**

Run:

```bash
git add omcgo/migrations docs/superpowers/plans/2026-07-16-consolidate-migrations-baseline.md
git commit -m "chore(migrations): 合并数据库迁移基线"
```

Expected: implementation and plan are committed; `.codex/` is not staged.

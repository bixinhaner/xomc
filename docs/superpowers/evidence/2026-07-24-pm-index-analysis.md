# PM sparse index analysis — 2026-07-24

## Acceptance status

**NOT PASSED — Task 10 blocking evidence.**

No plan or index measurement was taken on `172.24.224.197`. The host may still
be serving the 10,000-device workload, and this task was not given an isolated
copied database target. `EXPLAIN ANALYZE` executes its statement; the cleanup
case executes `DELETE` and generates WAL even when later rolled back. Running
that harness on the active database would therefore be unsafe.

No index is deleted by Task 8. Every candidate is **retained** until the
isolated evidence below is captured. An absent or zero `idx_scan` value alone
is not proof that an index is redundant.

## Reproducible evidence command

Use a copied, production-shaped TimescaleDB only. Do not put credentials in the
repository or captured output.

Create the sentinel only after independently cloning the database into the
dedicated identity `omc_task8_sparse_clone`. Never run this block on the
production database:

```sql
\set ON_ERROR_STOP on
SELECT current_database() = 'omc_task8_sparse_clone' AS is_expected_clone
\gset
\if :is_expected_clone
\else
  \echo 'refusing sentinel creation: not the exact isolated clone database'
  SELECT 1 / 0 AS task8_sentinel_creation_refusal;
\endif

CREATE SCHEMA task8_validation;
CREATE TABLE task8_validation.pm_explain_clone_sentinel (
    database_name name PRIMARY KEY,
    purpose text NOT NULL CHECK (purpose = 'pm-task8-isolated-clone'),
    created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO task8_validation.pm_explain_clone_sentinel
    (database_name, purpose)
VALUES (current_database(), 'pm-task8-isolated-clone');
```

The sentinel is clone-local evidence, not a general configuration table. Do
not migrate it and do not create it in any live database.

```bash
cd omcgo
set -o pipefail
psql "$TASK8_TSDB_DSN" \
  -v task8_isolated_safe_environment=on \
  -v task8_expected_database=omc_task8_sparse_clone \
  -f scripts/pm_explain_core_queries.sql 2>&1 \
  | tee pm-task8-explain.txt
```

Before any `EXPLAIN ANALYZE` or `DELETE`, the script requires the client flag,
an exact `current_database()` match, and a matching server-side clone
sentinel. Every refusal raises an `ON_ERROR_STOP` error, so `psql` exits 3.
`set -o pipefail` is mandatory: without it, `tee` can turn that refusal into
an apparent zero exit status.
The script pins `search_path`, fully qualifies PM relations, requires
representative joined rows, executes every plan in one transaction, and ends
with `ROLLBACK`. It covers:

- ingest idempotency lookup by `pm_ingest_batches.source_file_id`;
- latest device/counter lookup;
- raw-to-hourly aggregation;
- dashboard device/time/counter range;
- active-hourly latest and dashboard range reads;
- object/time range;
- source-file value and anchor cleanup;
- obsolete hourly-version cleanup;
- index definition, physical bytes, cumulative scans and fetched tuples;
- per-candidate chunk-index scans mapped to each hypertable index;
- full hypertable/chunk table, index and TOAST bytes;
- chunk and ordinary-relation scans plus insert/update/delete counts.

PostgreSQL does not expose a cumulative per-index write counter. Table
`n_tup_ins`, `n_tup_upd`, and `n_tup_del` are recorded as the write-frequency
proxy, while each statement's `WAL` block is the write-cost evidence.
`hypertable_index_size` and `hypertable_detailed_size` are used so chunk bytes
are not silently omitted; TimescaleDB chunk statistics are summed separately
so parent-table zeros are not mistaken for an idle workload. The installed
TimescaleDB catalog maps each chunk index back to its candidate hypertable
index for cumulative per-candidate scans.

Guard behavior can be verified without TimescaleDB data. The test starts a
temporary Unix-socket-only PostgreSQL instance and proves false flag, wrong
database, and absent sentinel all exit 3 before a query marker:

```bash
cd omcgo
./scripts/pm_explain_core_queries_guard_test.sh
```

## Candidate evidence register

The following fields must be replaced with captured values in Task 10. “Not
measured” is deliberately not an acceptance claim.

| Candidate | Core query or constraint | Plan / estimates / actual time | Buffers / WAL | Size / scans / write proxy | Decision |
|---|---|---|---|---|---|
| `pm_metric_dictionary_pkey` | dictionary joins by `metric_id` | Not measured | Not measured | Not measured | **Retained** |
| `pm_metric_dictionary_metric_path_key` | exact counter-path resolution and uniqueness | Not measured | Not measured | Not measured | **Retained** |
| `pm_metric_sets_pkey` | anchor-to-set joins | Not measured | Not measured | Not measured | **Retained** |
| `uq_pm_metric_sets_content` | ingest metric-set reuse and uniqueness | Not measured | Not measured | Not measured | **Retained** |
| `pm_ingest_batches_pkey` | anchor-to-batch joins | Not measured | Not measured | Not measured | **Retained** |
| `pm_ingest_batches_source_file_id_key` | ingest idempotency lookup | Not measured | Not measured | Not measured | **Retained** |
| `pm_measurement_anchors_pkey` | value joins and time uniqueness | Not measured | Not measured | Not measured | **Retained** |
| `idx_pm_anchors_device_time` | latest metric, hourly aggregation, dashboard range | Not measured | Not measured | Not measured | **Retained** |
| `idx_pm_anchors_object_time` | object/time range | Not measured | Not measured | Not measured | **Retained** |
| `uq_pm_metric_values` | value uniqueness and anchor/value joins | Not measured | Not measured | Not measured | **Retained** |
| `idx_pm_metric_values_metric_time` | latest metric, hourly aggregation, dashboard counter filter | Not measured | Not measured | Not measured; specifically verify the previously observed ~145 MiB class of this index | **Retained** |
| `pm_hourly_bucket_versions_pkey` | version publication and cleanup | Not measured | Not measured | Not measured | **Retained** |
| `uq_pm_hourly_active_bucket` | one active version per bucket | Not measured | Not measured | Not measured | **Retained** |
| `pm_hourly_rollup_batches_pkey` | batch progress and cleanup | Not measured | Not measured | Not measured | **Retained** |
| `pm_hourly_anchors_pkey` | hourly value joins | Not measured | Not measured | Not measured | **Retained** |
| `idx_pm_hourly_anchors_version_device` | active hourly device/range reads and cleanup | Not measured | Not measured | Not measured | **Retained** |
| `uq_pm_hourly_values` | hourly value uniqueness and joins | Not measured | Not measured | Not measured | **Retained** |
| `idx_pm_hourly_values_version_metric_time` | active hourly counter/range reads | Not measured | Not measured | Not measured | **Retained** |

TimescaleDB-created time indexes and chunk-local indexes must also be copied
from the harness output. They are **retained** until before/after plans prove
that removing one does not regress ingest lookup, latest metric, hourly
aggregation, dashboard range, object range, CSV/detail reads, or cleanup.

## Required Task 10 decision procedure

For each candidate, record the full plan node using it or the competing plan,
estimated versus actual rows, execution time, shared/temp buffers, WAL, index
bytes, cumulative scan counts, and table write proxy. Compare representative
cold and warm executions on the same copied dataset. Only a dedicated
migration with before/after focused tests may remove an index.

Until that evidence exists, the schema remains unchanged.

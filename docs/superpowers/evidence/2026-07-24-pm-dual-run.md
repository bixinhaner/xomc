# PM legacy versus sparse dual run — 2026-07-24

## Acceptance status

**NOT PASSED — Task 10 blocking acceptance.**

The same-input dual run was not executed on `172.24.224.197`. The host may
still be under the 10,000-device workload and no isolated legacy/sparse
database pair, MinIO pair, NATS pair, or port allocation was provided to this
task. No production table was read with an expensive plan or mutated.

Task 8 supplies the runnable logical/physical comparator and the exact capture
contract. Actual values for `logical_equal` and the sparse physical ratio must
not be reported as passing until the isolated run below is completed.

## Normalized comparison contract

Both implementations export the same logical columns:

| Column | Meaning |
|---|---|
| `device` | stable device key, normally `device_oui/device_sn` |
| `object` | object LDN; optional only for objectless counters |
| `time` | RFC3339 timestamp |
| `counter` | metric path |
| `value` | numeric value or null/empty |

The comparator treats `(device, object, time, counter)` as the logical key.
Rows may be in any order. Missing rows, changed values, and duplicate keys are
reported with the exact key and both sides' rows.

Accepted inputs are a JSON array of normalized rows or CSV with
`device,time,counter,value` and optional `object`. Example:

```csv
device,object,time,counter,value
001122/serial-1,Cell=1,2026-07-24T00:00:00Z,Signal.RSRP,-95
001122/serial-1,Cell=1,2026-07-24T00:00:00Z,Traffic.Bytes,
```

## Isolated same-input procedure

1. Prepare two isolated environments:
   - the pinned `main` legacy commit with its legacy schema;
   - the Task 8 branch with its sparse schema.
   Database, MinIO, NATS, ports, and data directories must not be shared.
2. Copy one immutable captured raw PM corpus into a neutral directory. Record
   its manifest before either run:

   ```bash
   find "$TASK8_RAW_PM_DIR" -type f -print0 \
     | sort -z \
     | xargs -0 shasum -a 256 \
     > task8-raw-pm.sha256
   ```

3. Feed that directory, without rewriting files, to the normal PM ingestion
   entry point of each isolated environment. Repeat the manifest command after
   each run and require an unchanged manifest.
4. Wait for both ingestion queues to drain. Run `CHECKPOINT` and `ANALYZE` in
   each isolated database before measuring. These commands must not be run on
   the active production database for Task 8.
5. Export the identical logical scope from each database. The compatibility
   view makes the query shape the same:

   ```sql
   \copy (
     SELECT concat_ws('/', device_oui, device_sn) AS device,
            object_ldn AS object,
            to_char(
              "time" AT TIME ZONE 'UTC',
              'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ) AS time,
            metric_path AS counter,
            metric_value AS value
       FROM pm_metrics
      ORDER BY 1, 2, 3, 4
   ) TO 'pm-logical.csv' WITH (FORMAT csv, HEADER true)
   ```

   Use separate output paths such as `legacy/pm-logical.csv` and
   `sparse/pm-logical.csv`. Export Counter, KPI, Dashboard, CSV, and hourly
   scopes separately when their query contracts differ; do not concatenate
   granularities into a key space that would create artificial duplicates.

6. Measure physical bytes after checkpoint/analyze. Legacy bytes include the
   legacy PM detail table, every attached index, and TOAST. Sparse bytes
   include dictionary, metric sets, ingest batches, anchors, values, and every
   attached index and TOAST:

   ```sql
   WITH ordinary(bytes) AS (
     SELECT sum(pg_total_relation_size(table_name::regclass))::bigint
       FROM unnest(ARRAY[
         'pm_metric_dictionary',
         'pm_metric_sets',
         'pm_ingest_batches'
       ]) AS table_name
   ),
   sparse_hypertables(bytes) AS (
     SELECT total_bytes
       FROM hypertable_detailed_size(
         'public.pm_measurement_anchors'::regclass
       )
     UNION ALL
     SELECT total_bytes
       FROM hypertable_detailed_size(
         'public.pm_metric_values'::regclass
       )
   )
   SELECT ordinary.bytes + sum(sparse_hypertables.bytes) AS bytes
     FROM ordinary CROSS JOIN sparse_hypertables
    GROUP BY ordinary.bytes;
   ```

   Run `hypertable_detailed_size` for the actual legacy `pm_metrics`
   hypertable. Do not use `pg_total_relation_size` on only the hypertable
   parent, because that omits TimescaleDB chunks, and do not count the sparse
   compatibility view.

7. Apply the acceptance gate:

   ```bash
   cd omcgo
   go run ./scripts/pm_sparse_compare.go \
     --old "$TASK8_LEGACY_EXPORT" \
     --sparse "$TASK8_SPARSE_EXPORT" \
     --old-bytes "$TASK8_LEGACY_BYTES" \
     --sparse-bytes "$TASK8_SPARSE_BYTES"
   ```

The command exits zero only when both conditions hold:

```text
logical_equal = true
sparse_bytes / old_bytes <= 0.20
```

It exits one for any logical mismatch, duplicate key, or ratio above 20%.
Zero/zero is a valid empty measurement; positive sparse bytes with a zero
legacy baseline cannot pass.

## Evidence still required

Task 10 must attach:

- immutable raw corpus manifest and exact baseline/branch commits;
- isolated environment identifiers without secrets;
- queue-drained, checkpoint, and analyze timestamps;
- comparator stdout and exit code for Counter, KPI, Dashboard, CSV, and hourly;
- mismatch output, if any;
- table, index, TOAST, and total bytes for both implementations;
- old/sparse WAL bytes for the same ingestion;
- anchor/value counts and legacy logical row counts;
- the final ratio and an explicit pass/fail conclusion.

Until all of the above is present, the sparse storage ratio and logical
equivalence are unverified.

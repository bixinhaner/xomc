\set ON_ERROR_STOP on
\pset pager off

-- OFFLINE ONLY. EXPLAIN ANALYZE executes every statement, including DELETE.
-- Run only against a copied, isolated Task 8 database:
--   psql "$TASK8_TSDB_DSN" \
--     -v task8_isolated_safe_environment=on \
--     -f scripts/pm_explain_core_queries.sql \
--     | tee pm-task8-explain.txt
\if :{?task8_isolated_safe_environment}
\else
  \echo 'refusing to run: pass -v task8_isolated_safe_environment=on for an isolated copied database'
  \quit
\endif
\if :task8_isolated_safe_environment
\else
  \echo 'refusing to run: task8_isolated_safe_environment must be on'
  \quit
\endif

BEGIN;
SET LOCAL application_name = 'pm-task8-isolated-explain';
SET LOCAL statement_timeout = '5min';
SET LOCAL lock_timeout = '1s';

-- Require production-shaped seed data before collecting misleading empty plans.
SELECT 1 / CASE
             WHEN EXISTS (
                    SELECT 1
                      FROM pm_ingest_batches b
                      JOIN pm_measurement_anchors a
                        ON a.source_file_id = b.source_file_id
                  )
              AND EXISTS (SELECT 1 FROM pm_metric_values)
              AND EXISTS (SELECT 1 FROM pm_hourly_bucket_versions)
             THEN 1
             ELSE 0
           END AS populated_dataset_required;

SELECT b.source_file_id::text AS sample_source_file_id
  FROM pm_ingest_batches b
 WHERE EXISTS (
       SELECT 1
         FROM pm_measurement_anchors a
        WHERE a.source_file_id = b.source_file_id
   )
 ORDER BY b.started_at DESC
 LIMIT 1
\gset

SELECT a.device_dim_id::text AS sample_device_id,
       v.metric_id::text AS sample_metric_id,
       a.object_ldn AS sample_object_ldn,
       (a."time" - interval '24 hours')::text AS sample_range_start,
       (a."time" + interval '1 second')::text AS sample_range_end
  FROM pm_measurement_anchors a
  JOIN pm_metric_values v
    ON v."time" = a."time" AND v.anchor_id = a.anchor_id
 ORDER BY a."time" DESC
 LIMIT 1
\gset

\echo 'QUERY ingest lookup by source_file_id'
EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)
SELECT ingest_batch_id, status, started_at, committed_at
  FROM pm_ingest_batches
 WHERE source_file_id = :'sample_source_file_id'::uuid;

\echo 'QUERY latest metric for one device and counter'
EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)
SELECT a."time", a.anchor_id, v.metric_value
  FROM pm_measurement_anchors a
  JOIN pm_metric_values v
    ON v."time" = a."time" AND v.anchor_id = a.anchor_id
 WHERE a.device_dim_id = :'sample_device_id'::uuid
   AND v.metric_id = :'sample_metric_id'::bigint
 ORDER BY a."time" DESC, a.anchor_id DESC
 LIMIT 1;

\echo 'QUERY raw hourly aggregate'
EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)
SELECT date_trunc('hour', a."time") AS bucket_start,
       avg(v.metric_value) AS metric_value
  FROM pm_measurement_anchors a
  JOIN pm_metric_values v
    ON v."time" = a."time" AND v.anchor_id = a.anchor_id
 WHERE a.device_dim_id = :'sample_device_id'::uuid
   AND v.metric_id = :'sample_metric_id'::bigint
   AND a."time" >= :'sample_range_start'::timestamptz
   AND a."time" < :'sample_range_end'::timestamptz
 GROUP BY 1
 ORDER BY 1;

\echo 'QUERY dashboard device/time/counter range'
EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)
SELECT a."time", a.start_time, a.end_time, a.object_ldn,
       d.metric_path, d.metric_type, v.metric_value
  FROM pm_measurement_anchors a
  JOIN pm_metric_sets s ON s.metric_set_id = a.metric_set_id
  JOIN pm_metric_dictionary d
    ON d.metric_id = ANY (s.metric_ids)
   AND d.metric_id = :'sample_metric_id'::bigint
  LEFT JOIN pm_metric_values v
    ON v."time" = a."time"
   AND v.anchor_id = a.anchor_id
   AND v.metric_id = d.metric_id
 WHERE a.device_dim_id = :'sample_device_id'::uuid
   AND a."time" >= :'sample_range_start'::timestamptz
   AND a."time" < :'sample_range_end'::timestamptz
 ORDER BY a."time" DESC, a.anchor_id DESC;

\echo 'QUERY object/time range'
EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)
SELECT a."time", a.anchor_id, a.device_dim_id
  FROM pm_measurement_anchors a
 WHERE a.object_ldn = :'sample_object_ldn'
   AND a."time" >= :'sample_range_start'::timestamptz
   AND a."time" < :'sample_range_end'::timestamptz
 ORDER BY a."time" DESC, a.anchor_id DESC;

\echo 'QUERY source-file value cleanup (rolled back)'
SAVEPOINT before_source_value_cleanup;
EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)
DELETE FROM pm_metric_values v
 USING pm_measurement_anchors a
 WHERE a.source_file_id = :'sample_source_file_id'::uuid
   AND v."time" = a."time"
   AND v.anchor_id = a.anchor_id;
ROLLBACK TO SAVEPOINT before_source_value_cleanup;

\echo 'QUERY source-file anchor cleanup (rolled back)'
SAVEPOINT before_source_anchor_cleanup;
EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)
DELETE FROM pm_measurement_anchors
 WHERE source_file_id = :'sample_source_file_id'::uuid;
ROLLBACK TO SAVEPOINT before_source_anchor_cleanup;

-- This statement executes and can generate WAL. The savepoint and final
-- transaction rollback make it non-persistent, but do not make it suitable for
-- a live system. Its guard therefore requires an isolated copied database.
\echo 'QUERY obsolete hourly-version cleanup (rolled back)'
SAVEPOINT before_cleanup_explain;
EXPLAIN (ANALYZE, BUFFERS, WAL, SETTINGS)
WITH doomed AS (
    SELECT bucket_version
      FROM pm_hourly_bucket_versions
     WHERE status IN ('failed', 'superseded')
       AND created_at < now() - interval '24 hours'
),
deleted_values AS (
    DELETE FROM pm_hourly_values v
     USING pm_hourly_anchors a, doomed d
     WHERE a.bucket_version = d.bucket_version
       AND v."time" = a."time"
       AND v.anchor_id = a.anchor_id
),
deleted_anchors AS (
    DELETE FROM pm_hourly_anchors a
     USING doomed d
     WHERE a.bucket_version = d.bucket_version
),
deleted_batches AS (
    DELETE FROM pm_hourly_rollup_batches b
     USING doomed d
     WHERE b.bucket_version = d.bucket_version
)
DELETE FROM pm_hourly_bucket_versions v
 USING doomed d
 WHERE v.bucket_version = d.bucket_version;
ROLLBACK TO SAVEPOINT before_cleanup_explain;

\echo 'EVIDENCE index definitions, size, and cumulative scans'
SELECT i.relname AS table_name,
       s.indexrelname AS index_name,
       CASE
         WHEN EXISTS (
             SELECT 1
               FROM timescaledb_information.hypertables h
              WHERE h.hypertable_schema = s.schemaname
                AND h.hypertable_name = i.relname
         )
         THEN hypertable_index_size(s.indexrelid::regclass)
         ELSE pg_relation_size(s.indexrelid)
       END AS index_bytes_including_chunks,
       s.idx_scan,
       s.idx_tup_read,
       s.idx_tup_fetch,
       pg_get_indexdef(s.indexrelid) AS definition
  FROM pg_stat_user_indexes s
  JOIN pg_class i ON i.oid = s.relid
 WHERE s.schemaname = 'public'
   AND i.relname IN (
       'pm_metric_dictionary',
       'pm_metric_sets',
       'pm_ingest_batches',
       'pm_measurement_anchors',
       'pm_metric_values',
       'pm_hourly_bucket_versions',
       'pm_hourly_rollup_batches',
       'pm_hourly_anchors',
       'pm_hourly_values'
   )
 ORDER BY i.relname, s.indexrelname;

\echo 'EVIDENCE cumulative scans mapped from chunk indexes to each hypertable index'
SELECT ci.hypertable_index_name AS index_name,
       sum(COALESCE(s.idx_scan, 0)) AS idx_scan,
       sum(COALESCE(s.idx_tup_read, 0)) AS idx_tup_read,
       sum(COALESCE(s.idx_tup_fetch, 0)) AS idx_tup_fetch
  FROM _timescaledb_catalog.chunk_index ci
  JOIN _timescaledb_catalog.chunk ch ON ch.id = ci.chunk_id
  JOIN _timescaledb_catalog.hypertable h ON h.id = ci.hypertable_id
  LEFT JOIN pg_stat_all_indexes s
    ON s.schemaname = ch.schema_name
   AND s.indexrelname = ci.index_name
 WHERE h.schema_name = 'public'
   AND h.table_name IN (
       'pm_measurement_anchors',
       'pm_metric_values',
       'pm_hourly_anchors',
       'pm_hourly_values'
   )
 GROUP BY ci.hypertable_index_name
 ORDER BY ci.hypertable_index_name;

\echo 'EVIDENCE full hypertable bytes including chunks'
SELECT 'pm_measurement_anchors' AS table_name, sizes.*
  FROM hypertable_detailed_size('public.pm_measurement_anchors'::regclass) sizes
UNION ALL
SELECT 'pm_metric_values' AS table_name, sizes.*
  FROM hypertable_detailed_size('public.pm_metric_values'::regclass) sizes
UNION ALL
SELECT 'pm_hourly_anchors' AS table_name, sizes.*
  FROM hypertable_detailed_size('public.pm_hourly_anchors'::regclass) sizes
UNION ALL
SELECT 'pm_hourly_values' AS table_name, sizes.*
  FROM hypertable_detailed_size('public.pm_hourly_values'::regclass) sizes
ORDER BY table_name;

\echo 'EVIDENCE hypertable chunk scan/write frequency'
SELECT ch.hypertable_name AS table_name,
       sum(s.seq_scan) AS seq_scan,
       sum(s.idx_scan) AS idx_scan,
       sum(s.n_tup_ins) AS n_tup_ins,
       sum(s.n_tup_upd) AS n_tup_upd,
       sum(s.n_tup_del) AS n_tup_del
  FROM timescaledb_information.chunks ch
  JOIN pg_stat_all_tables s
    ON s.schemaname = ch.chunk_schema
   AND s.relname = ch.chunk_name
 WHERE ch.hypertable_schema = 'public'
   AND ch.hypertable_name IN (
       'pm_measurement_anchors',
       'pm_metric_values',
       'pm_hourly_anchors',
       'pm_hourly_values'
   )
 GROUP BY ch.hypertable_name
 ORDER BY ch.hypertable_name;

\echo 'EVIDENCE ordinary/parent relation stats and write frequency'
SELECT s.relname AS table_name,
       pg_relation_size(s.relid) AS table_bytes,
       pg_indexes_size(s.relid) AS index_bytes,
       CASE
         WHEN c.reltoastrelid = 0 THEN 0
         ELSE pg_total_relation_size(c.reltoastrelid)
       END AS toast_bytes,
       pg_total_relation_size(s.relid) AS total_bytes,
       s.seq_scan,
       s.idx_scan,
       s.n_tup_ins,
       s.n_tup_upd,
       s.n_tup_del
  FROM pg_stat_user_tables s
  JOIN pg_class c ON c.oid = s.relid
 WHERE s.schemaname = 'public'
   AND s.relname IN (
       'pm_metric_dictionary',
       'pm_metric_sets',
       'pm_ingest_batches',
       'pm_measurement_anchors',
       'pm_metric_values',
       'pm_hourly_bucket_versions',
       'pm_hourly_rollup_batches',
       'pm_hourly_anchors',
       'pm_hourly_values'
   )
 ORDER BY s.relname;

ROLLBACK;

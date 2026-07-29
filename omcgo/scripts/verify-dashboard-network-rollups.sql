\set ON_ERROR_STOP on
\pset pager off

\echo '=== built-in network rollup row counts ==='
WITH tasks(technology, task_id) AS (
  VALUES
    ('lte',  '0184dddd-0001-4000-8000-000000000001'::uuid),
    ('nr',   '0184dddd-0001-4000-8000-000000000002'::uuid),
    ('gsm',  '0184dddd-0001-4000-8000-000000000003'::uuid)
)
SELECT t.technology, r.granularity, count(*) AS row_count
FROM tasks t
LEFT JOIN public.pm_aggregation_results r
  ON r.task_id = t.task_id
 AND r.technology = t.technology
 AND r.dimension = 'network'
 AND r.metric_type = 'kpi'
GROUP BY t.technology, r.granularity
ORDER BY t.technology, r.granularity;

\echo '=== latest window and lag by technology/granularity ==='
SELECT technology,
       granularity,
       max(window_start) AS latest_window_start,
       extract(epoch FROM now() - max(window_end))::bigint AS lag_seconds
FROM public.pm_aggregation_results
WHERE dimension = 'network'
  AND metric_type = 'kpi'
  AND task_id IN (
    '0184dddd-0001-4000-8000-000000000001'::uuid,
    '0184dddd-0001-4000-8000-000000000002'::uuid,
    '0184dddd-0001-4000-8000-000000000003'::uuid
  )
  AND granularity IN ('hourly', 'daily', 'weekly')
GROUP BY technology, granularity
ORDER BY technology, granularity;

\echo '=== missing dashboard metric windows (empty result means covered) ==='
WITH expected_metrics(technology, metric_path) AS (
  VALUES
    ('lte','K900010015'), ('lte','K900010016'), ('lte','K900010040'),
    ('lte','K900010041'), ('lte','K900010076'), ('lte','K900010014'),
    ('lte','K900010013'), ('lte','K900010006'), ('lte','K900010002'),
    ('lte','K900010005'), ('lte','K900010029'), ('lte','K900010027'),
    ('lte','K900010017'), ('lte','K900010022'), ('lte','K900010021'),
    ('lte','K900010026'),
    ('nr','KGNB0511'), ('nr','KGNB0510'), ('nr','KGNB0517'),
    ('nr','KGNB0516'), ('nr','KGNB0506'), ('nr','KGNB0505'),
    ('gsm','KGSM0102'), ('gsm','KGSM0103'), ('gsm','KGSM0101')
),
windows(granularity, start_time, end_time, expected_buckets) AS (
  VALUES
    ('hourly', date_trunc('hour', now()) - interval '24 hours',
               date_trunc('hour', now()), 24::bigint),
    ('daily',  date_trunc('day', now()) - interval '30 days',
               date_trunc('day', now()), 30::bigint),
    ('weekly', date_trunc('week', now()) - interval '12 weeks',
               date_trunc('week', now()), 12::bigint)
),
coverage AS (
  SELECT m.technology,
         m.metric_path,
         w.granularity,
         w.expected_buckets,
         count(DISTINCT r.window_start)::bigint AS observed_buckets
  FROM expected_metrics m
  CROSS JOIN windows w
  LEFT JOIN public.pm_aggregation_results r
    ON r.technology = m.technology
   AND r.metric_path = m.metric_path
   AND r.granularity = w.granularity
   AND r.dimension = 'network'
   AND r.metric_type = 'kpi'
   AND r.window_start >= w.start_time
   AND r.window_start < w.end_time
  GROUP BY m.technology, m.metric_path, w.granularity, w.expected_buckets
)
SELECT technology,
       granularity,
       metric_path,
       expected_buckets,
       observed_buckets,
       expected_buckets - observed_buckets AS missing_buckets
FROM coverage
WHERE observed_buckets < expected_buckets
ORDER BY technology, granularity, metric_path;

\echo '=== incomplete windows (empty result means complete) ==='
SELECT technology,
       granularity,
       metric_path,
       window_start,
       complete,
       missing_slots
FROM public.pm_aggregation_results
WHERE dimension = 'network'
  AND metric_type = 'kpi'
  AND granularity IN ('hourly', 'daily', 'weekly')
  AND (NOT complete OR missing_slots > 0)
  AND window_start >= date_trunc('week', now()) - interval '12 weeks'
ORDER BY window_start DESC, technology, granularity, metric_path;

\echo '=== overlapping logical window versions (empty result means unique) ==='
SELECT technology,
       granularity,
       metric_path,
       window_start,
       count(*) AS version_count
FROM public.pm_aggregation_results
WHERE dimension = 'network'
  AND metric_type = 'kpi'
  AND granularity IN ('hourly', 'daily', 'weekly')
  AND window_start >= date_trunc('week', now()) - interval '12 weeks'
GROUP BY technology, granularity, metric_path, window_start
HAVING count(*) > 1
ORDER BY version_count DESC, window_start DESC;

\echo '=== dashboard representative query plan ==='
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT DISTINCT ON (r.technology, r.metric_path, r.window_start)
       r.technology,
       r.metric_path,
       r.granularity,
       r.window_start,
       r.window_end,
       r.metric_value,
       r.complete,
       r.missing_slots,
       r.created_at
FROM public.pm_aggregation_results r
WHERE r.dimension = 'network'
  AND r.metric_type = 'kpi'
  AND r.task_id = '0184dddd-0001-4000-8000-000000000001'::uuid
  AND r.technology = 'lte'
  AND r.granularity = 'hourly'
  AND r.metric_path = ANY(ARRAY['K900010015','K900010016'])
  AND r.window_start >= date_trunc('hour', now()) - interval '24 hours'
  AND r.window_start < date_trunc('hour', now()) + interval '1 hour'
ORDER BY r.technology, r.metric_path, r.window_start, r.created_at DESC;

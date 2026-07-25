-- +goose Up

-- 旧扫描式聚合结果不迁移。
DROP VIEW IF EXISTS public.pm_metrics_hourly;
DROP TABLE IF EXISTS public.pm_adhoc_aggregation_results;
DROP TABLE IF EXISTS public.pm_group_metrics_monthly;
DROP TABLE IF EXISTS public.pm_group_metrics_weekly;
DROP TABLE IF EXISTS public.pm_group_metrics_daily;
DROP TABLE IF EXISTS public.pm_group_metrics_hourly;
DROP TABLE IF EXISTS public.pm_metrics_monthly;
DROP TABLE IF EXISTS public.pm_metrics_weekly;
DROP TABLE IF EXISTS public.pm_metrics_daily;
DROP TABLE IF EXISTS public.pm_hourly_values;
DROP TABLE IF EXISTS public.pm_hourly_anchors;
DROP TABLE IF EXISTS public.pm_hourly_rollup_batches;
DROP TABLE IF EXISTS public.pm_hourly_bucket_versions;

CREATE TABLE public.pm_aggregation_outbox (
    event_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id uuid NOT NULL UNIQUE,
    ingest_batch_id uuid NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    publish_attempts integer NOT NULL DEFAULT 0,
    last_error text
);
CREATE INDEX idx_pm_aggregation_outbox_pending
    ON public.pm_aggregation_outbox (created_at, event_id)
    WHERE published_at IS NULL;

CREATE TABLE public.pm_aggregation_windows (
    task_id uuid NOT NULL,
    task_version_id uuid NOT NULL,
    granularity varchar(16) NOT NULL,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'open',
    expected_slots bigint NOT NULL,
    received_slots bigint NOT NULL DEFAULT 0,
    missing_slots bigint NOT NULL DEFAULT 0,
    close_reason varchar(16),
    result_count integer NOT NULL DEFAULT 0,
    opened_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    last_error text,
    PRIMARY KEY (task_version_id, granularity, window_start),
    CONSTRAINT chk_pm_aggregation_windows_granularity
        CHECK (granularity IN ('hourly', 'daily', 'weekly', 'monthly')),
    CONSTRAINT chk_pm_aggregation_windows_status
        CHECK (status IN ('open', 'finalizing', 'published', 'failed')),
    CONSTRAINT chk_pm_aggregation_windows_close_reason
        CHECK (close_reason IS NULL OR close_reason IN ('complete', 'timeout')),
    CONSTRAINT chk_pm_aggregation_windows_time CHECK (window_end > window_start)
);
CREATE INDEX idx_pm_aggregation_windows_due
    ON public.pm_aggregation_windows (window_end, status)
    WHERE status IN ('open', 'failed', 'finalizing');
CREATE INDEX idx_pm_aggregation_windows_task_time
    ON public.pm_aggregation_windows (task_id, window_start DESC);

CREATE TABLE public.pm_aggregation_results (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    task_id uuid NOT NULL,
    task_version_id uuid NOT NULL,
    granularity varchar(16) NOT NULL,
    dimension varchar(32) NOT NULL,
    dimension_key text NOT NULL,
    dimension_name text NOT NULL DEFAULT '',
    object_ldn text NOT NULL DEFAULT '',
    device_oui text NOT NULL DEFAULT '',
    device_sn text NOT NULL DEFAULT '',
    technology varchar(16) NOT NULL DEFAULT '',
    metric_id text NOT NULL,
    metric_path text NOT NULL,
    metric_type varchar(16) NOT NULL,
    aggregation_op varchar(8) NOT NULL,
    metric_value double precision NOT NULL,
    sample_count bigint NOT NULL,
    complete boolean NOT NULL,
    missing_slots bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_pm_aggregation_results_granularity
        CHECK (granularity IN ('hourly', 'daily', 'weekly', 'monthly')),
    CONSTRAINT chk_pm_aggregation_results_dimension
        CHECK (dimension IN ('device', 'aggregate_group', 'device_group', 'product', 'band', 'network')),
    CONSTRAINT chk_pm_aggregation_results_metric_type
        CHECK (metric_type IN ('counter', 'kpi')),
    CONSTRAINT chk_pm_aggregation_results_op
        CHECK (aggregation_op IN ('sum', 'avg', 'min', 'max')),
    CONSTRAINT chk_pm_aggregation_results_time CHECK (window_end > window_start)
);

SELECT create_hypertable(
    'public.pm_aggregation_results',
    by_range('window_start', INTERVAL '30 days'),
    if_not_exists => TRUE
);

CREATE UNIQUE INDEX uq_pm_aggregation_results_business
    ON public.pm_aggregation_results (
        task_version_id, granularity, window_start,
        dimension_key, object_ldn, technology, metric_id
    );
CREATE INDEX idx_pm_aggregation_results_task_time
    ON public.pm_aggregation_results (task_id, window_start DESC);
CREATE INDEX idx_pm_aggregation_results_dimension_time
    ON public.pm_aggregation_results (dimension, dimension_key, window_start DESC);
CREATE INDEX idx_pm_aggregation_results_metric_time
    ON public.pm_aggregation_results (metric_id, window_start DESC);

ALTER TABLE public.pm_aggregation_results SET (
    timescaledb.compress,
    timescaledb.compress_segmentby =
        'task_id,task_version_id,granularity,dimension,dimension_key,technology,metric_id'
);
SELECT add_compression_policy(
    'public.pm_aggregation_results',
    INTERVAL '90 days',
    if_not_exists => TRUE
);
SELECT add_retention_policy(
    'public.pm_aggregation_results',
    INTERVAL '365 days',
    if_not_exists => TRUE
);

-- 现有报表/导出 API 的只读投影。它不保留旧结果，也不触发原始 PM 查询。
CREATE VIEW public.pm_adhoc_aggregation_results AS
SELECT
    r.id,
    r.task_id,
    r.device_oui,
    CASE WHEN r.dimension = 'device' THEN r.device_sn ELSE 'AGGREGATED' END::text AS device_sn,
    CASE WHEN r.dimension = 'product' THEN r.dimension_key::uuid ELSE NULL::uuid END AS product_id,
    r.metric_path,
    r.metric_type,
    r.metric_value,
    r.aggregation_op::text AS statis_type,
    r.granularity::text,
    r.window_start AS "time",
    r.window_start AS start_time,
    r.window_end AS end_time,
    r.created_at AS ingest_time,
    CASE
        WHEN r.dimension = 'device' THEN NULLIF(r.object_ldn, '')
        WHEN r.dimension = 'network' THEN 'Network'
        ELSE r.dimension_key
    END::text AS object_ldn,
    jsonb_build_object(
        'task_version_id', r.task_version_id,
        'complete', r.complete,
        'missing_slots', r.missing_slots,
        'dimension', r.dimension
    ) AS extra
FROM public.pm_aggregation_results r;

-- 旧报表查询名称改为统一结果表的只读投影。投影不扫描原始 PM 表；当多个逻辑任务
-- 覆盖同一设备/指标/窗口时，仅暴露最新写入的一份，避免报表重复计数。
CREATE VIEW public.pm_metrics_hourly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.object_ldn)
    r.id, r.device_oui, r.device_sn, r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time, r.window_end AS end_time,
    r.created_at AS ingest_time, r.object_ldn,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra
FROM public.pm_aggregation_results r
WHERE r.dimension = 'device' AND r.granularity = 'hourly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.object_ldn, r.created_at DESC;

CREATE VIEW public.pm_metrics_daily AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.object_ldn)
    r.device_oui, r.device_sn, r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time, r.object_ldn,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra
FROM public.pm_aggregation_results r
WHERE r.dimension = 'device' AND r.granularity = 'daily'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.object_ldn, r.created_at DESC;

CREATE VIEW public.pm_metrics_weekly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.object_ldn)
    r.device_oui, r.device_sn, r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time, r.object_ldn,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra
FROM public.pm_aggregation_results r
WHERE r.dimension = 'device' AND r.granularity = 'weekly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.object_ldn, r.created_at DESC;

CREATE VIEW public.pm_metrics_monthly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.object_ldn)
    r.device_oui, r.device_sn, r.metric_path, r.metric_type, r.metric_value,
    r.aggregation_op::text AS statis_type, r.granularity::text,
    r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time, r.object_ldn,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra
FROM public.pm_aggregation_results r
WHERE r.dimension = 'device' AND r.granularity = 'monthly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.object_ldn, r.created_at DESC;

CREATE VIEW public.pm_group_metrics_hourly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.technology)
    r.id,
    replace(r.dimension_key, 'DeviceGroup=', '')::uuid AS device_group_id,
    r.metric_path, r.metric_type, r.metric_value, r.aggregation_op::text AS statis_type,
    r.granularity::text, r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra,
    r.technology
FROM public.pm_aggregation_results r
WHERE r.dimension = 'device_group' AND r.granularity = 'hourly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.technology, r.created_at DESC;

CREATE VIEW public.pm_group_metrics_daily AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.technology)
    replace(r.dimension_key, 'DeviceGroup=', '')::uuid AS device_group_id,
    r.metric_path, r.metric_type, r.metric_value, r.aggregation_op::text AS statis_type,
    r.granularity::text, r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra,
    r.technology
FROM public.pm_aggregation_results r
WHERE r.dimension = 'device_group' AND r.granularity = 'daily'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.technology, r.created_at DESC;

CREATE VIEW public.pm_group_metrics_weekly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.technology)
    replace(r.dimension_key, 'DeviceGroup=', '')::uuid AS device_group_id,
    r.metric_path, r.metric_type, r.metric_value, r.aggregation_op::text AS statis_type,
    r.granularity::text, r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra,
    r.technology
FROM public.pm_aggregation_results r
WHERE r.dimension = 'device_group' AND r.granularity = 'weekly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.technology, r.created_at DESC;

CREATE VIEW public.pm_group_metrics_monthly AS
SELECT DISTINCT ON (r.dimension_key, r.metric_id, r.window_start, r.technology)
    replace(r.dimension_key, 'DeviceGroup=', '')::uuid AS device_group_id,
    r.metric_path, r.metric_type, r.metric_value, r.aggregation_op::text AS statis_type,
    r.granularity::text, r.window_start AS "time", r.window_start AS start_time,
    r.window_end AS end_time, r.created_at AS ingest_time,
    jsonb_build_object('task_id', r.task_id, 'task_version_id', r.task_version_id,
                       'complete', r.complete, 'missing_slots', r.missing_slots) AS extra,
    r.technology
FROM public.pm_aggregation_results r
WHERE r.dimension = 'device_group' AND r.granularity = 'monthly'
ORDER BY r.dimension_key, r.metric_id, r.window_start, r.technology, r.created_at DESC;

-- +goose Down

DROP VIEW IF EXISTS public.pm_group_metrics_monthly;
DROP VIEW IF EXISTS public.pm_group_metrics_weekly;
DROP VIEW IF EXISTS public.pm_group_metrics_daily;
DROP VIEW IF EXISTS public.pm_group_metrics_hourly;
DROP VIEW IF EXISTS public.pm_metrics_monthly;
DROP VIEW IF EXISTS public.pm_metrics_weekly;
DROP VIEW IF EXISTS public.pm_metrics_daily;
DROP VIEW IF EXISTS public.pm_metrics_hourly;
DROP VIEW IF EXISTS public.pm_adhoc_aggregation_results;
DROP TABLE IF EXISTS public.pm_aggregation_results;
DROP TABLE IF EXISTS public.pm_aggregation_windows;
DROP TABLE IF EXISTS public.pm_aggregation_outbox;

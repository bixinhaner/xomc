-- +goose Up
-- +goose StatementBegin
-- T-0164-P7 / G7 自定义聚合任务（oneshot + continuous）数据模型
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.7
-- 实施 plan：docs/project/plan-T-0164-P7-adhoc-aggregation.md
--
-- 变更：
--   1. 扩 pm_tasks（已有，per-module，不进 G8）加 7 列承载 adhoc_aggregation 模式
--   2. 新建 pm_adhoc_aggregation_results hypertable 落任务结果
--
-- 设计要点：
--   - pm_tasks 既有 device_sns(JSONB)/kpi_codes(JSONB)/granularity/time_range/status/progress
--     对老 extraction/report/threshold-check 任务无影响（新列 NULL 即可）
--   - G7 adhoc 任务复用 status/progress；新增 task_subtype 区分（extraction vs adhoc_aggregation）
--   - metric_paths / granularities 用 TEXT[]（PG 原生数组，支持 ANY/&&/contains 算子，比 JSONB 更紧凑）
--   - window_start/window_end 单独列（adhoc 必填，老任务可空 — 老任务的 time_range JSONB 仍用）
--   - cron_expr 仅 continuous 模式填
--   - pm_adhoc_aggregation_results 是 hypertable：retention 365d（G2 sys_configs 后续可改）
--     drop_chunks 整块清；task 删除不级联（panel ↔ task 独立生命周期，G6 引用）

ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS task_subtype  TEXT;
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS mode          TEXT;
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS cron_expr     TEXT;
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS metric_paths  TEXT[];
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS granularities TEXT[];
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS window_start  TIMESTAMPTZ;
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS window_end    TIMESTAMPTZ;

-- CHECK 约束（仅对新 adhoc 任务，老任务 task_subtype/mode IS NULL 不受约束）
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_mode;
ALTER TABLE pm_tasks ADD  CONSTRAINT chk_pm_tasks_mode
    CHECK (mode IS NULL OR mode IN ('oneshot', 'continuous'));

ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_continuous_cron;
ALTER TABLE pm_tasks ADD  CONSTRAINT chk_pm_tasks_continuous_cron
    CHECK (mode IS DISTINCT FROM 'continuous' OR cron_expr IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_pm_tasks_subtype_status
    ON pm_tasks (task_subtype, status)
    WHERE task_subtype IS NOT NULL;

-- ==================== pm_adhoc_aggregation_results：hypertable ====================
CREATE TABLE pm_adhoc_aggregation_results (
    id           UUID NOT NULL DEFAULT gen_random_uuid(),
    task_id      UUID NOT NULL,
    device_oui   TEXT NOT NULL,
    device_sn    TEXT NOT NULL,
    metric_path  TEXT NOT NULL,
    metric_type  TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value DOUBLE PRECISION NOT NULL,
    statis_type  TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity  TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time         TIMESTAMPTZ NOT NULL,
    start_time   TIMESTAMPTZ NOT NULL,
    end_time     TIMESTAMPTZ NOT NULL,
    ingest_time  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn   TEXT,
    extra        JSONB,
    PRIMARY KEY (id, time)
);

CREATE INDEX idx_pm_adhoc_results_task_time
    ON pm_adhoc_aggregation_results (task_id, time DESC);
CREATE INDEX idx_pm_adhoc_results_device
    ON pm_adhoc_aggregation_results (device_oui, device_sn, time DESC);

SELECT create_hypertable('pm_adhoc_aggregation_results', 'time', chunk_time_interval => INTERVAL '30 days');

ALTER TABLE pm_adhoc_aggregation_results SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'task_id,device_oui,device_sn,metric_type'
);

SELECT add_compression_policy('pm_adhoc_aggregation_results', INTERVAL '90 days');
-- G2 sys_configs 'pm.retention.adhoc_aggregation_days' 默认 365；retention policy 由 G2 reload 联动
SELECT add_retention_policy('pm_adhoc_aggregation_results', INTERVAL '365 days');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT remove_retention_policy('pm_adhoc_aggregation_results', if_exists => true);
SELECT remove_compression_policy('pm_adhoc_aggregation_results', if_exists => true);
DROP TABLE IF EXISTS pm_adhoc_aggregation_results CASCADE;

DROP INDEX IF EXISTS idx_pm_tasks_subtype_status;

ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_continuous_cron;
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_mode;

ALTER TABLE pm_tasks DROP COLUMN IF EXISTS window_end;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS window_start;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS granularities;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS metric_paths;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS cron_expr;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS mode;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS task_subtype;
-- +goose StatementEnd

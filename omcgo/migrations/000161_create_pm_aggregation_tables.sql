-- +goose Up
-- +goose StatementBegin
-- T-0164-P5 / G5 自然日历桶聚合表（设备维度 + 设备组维度）
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.5
-- 实施 plan：docs/project/plan-T-0164-P5-natural-bucket-aggregation.md
--
-- 变更：
--   1. 新增 4 张设备维度聚合表：
--        pm_metrics_hourly  — hypertable（chunk 7d，compress 14d，retention 180d）
--        pm_metrics_daily   — 普通表，G2 cron 按 sys_configs 清理
--        pm_metrics_weekly  — 普通表
--        pm_metrics_monthly — 普通表
--   2. 新增 4 张设备组维度聚合表（同粒度）：
--        pm_group_metrics_hourly / daily / weekly / monthly
--
-- 设计要点（与 pm_metrics 一致）：
--   - 设备唯一标识 (device_oui, device_sn) TR-069 双键（T-0164-P3 / G3 引入）
--   - statis_type / granularity / metric_type 等枚举约束与 pm_metrics 完全对齐
--   - hourly 仿 pm_metrics 走 hypertable + PRIMARY KEY (id, time) + UNIQUE 自然键，
--     支持 ON CONFLICT 幂等 UPSERT（cron 重跑 / 补算同窗口安全）
--   - daily/weekly/monthly 量小（年级别 ~1500 行/设备/KPI），普通表 PRIMARY KEY
--     直接落自然键，不再需要 id 列，节省一次 gen_random_uuid 调用
--   - device_group 维度同 device 维度但维度键换成 device_group_id（UUID）；
--     device_group_id 不建外键（device_groups 是普通表但避免删 group 级联连锁）
--   - extra JSONB 与 pm_metrics 一致，可装 carrier/technology/cluster 等扩展
--
-- 与 pm_metrics 的关系：
--   - 数据流：pm_metrics(15min) → pm_metrics_hourly → pm_metrics_daily → weekly → monthly
--   - cron runner（G5 §3 Task 3 + worker main G8 wiring）按整点对齐自然桶聚合
--   - QueryAggregated 路由：粒度 15min → pm_metrics，其余 → 对应粒度表（device 维度）
--                          或对应 pm_group_metrics_* 表（device_group 维度）

-- ==================== pm_metrics_hourly：hypertable ====================
CREATE TABLE pm_metrics_hourly (
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (id, time)
);

CREATE UNIQUE INDEX uq_pm_metrics_hourly_natural
    ON pm_metrics_hourly (device_oui, device_sn, metric_path, granularity, end_time, time);
CREATE INDEX idx_pm_metrics_hourly_path_time   ON pm_metrics_hourly (metric_path, time DESC);
CREATE INDEX idx_pm_metrics_hourly_device_time ON pm_metrics_hourly (device_oui, device_sn, time DESC);
CREATE INDEX idx_pm_metrics_hourly_object_ldn  ON pm_metrics_hourly (object_ldn) WHERE object_ldn IS NOT NULL;

SELECT create_hypertable('pm_metrics_hourly', 'time', chunk_time_interval => INTERVAL '7 days');

ALTER TABLE pm_metrics_hourly SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_oui,device_sn,metric_type,granularity'
);

SELECT add_compression_policy('pm_metrics_hourly', INTERVAL '14 days');
SELECT add_retention_policy('pm_metrics_hourly', INTERVAL '180 days');

-- ==================== pm_metrics_daily：普通表 ====================
CREATE TABLE pm_metrics_daily (
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_metrics_daily_path_time   ON pm_metrics_daily (metric_path, end_time DESC);
CREATE INDEX idx_pm_metrics_daily_device_time ON pm_metrics_daily (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_daily_object_ldn  ON pm_metrics_daily (object_ldn) WHERE object_ldn IS NOT NULL;

-- ==================== pm_metrics_weekly：普通表 ====================
CREATE TABLE pm_metrics_weekly (
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_metrics_weekly_path_time   ON pm_metrics_weekly (metric_path, end_time DESC);
CREATE INDEX idx_pm_metrics_weekly_device_time ON pm_metrics_weekly (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_weekly_object_ldn  ON pm_metrics_weekly (object_ldn) WHERE object_ldn IS NOT NULL;

-- ==================== pm_metrics_monthly：普通表 ====================
CREATE TABLE pm_metrics_monthly (
    device_oui    TEXT NOT NULL,
    device_sn     TEXT NOT NULL,
    metric_path   TEXT NOT NULL,
    metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value  DOUBLE PRECISION NOT NULL,
    statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity   TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time          TIMESTAMPTZ NOT NULL,
    start_time    TIMESTAMPTZ NOT NULL,
    end_time      TIMESTAMPTZ NOT NULL,
    ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    object_ldn    TEXT,
    extra         JSONB,
    PRIMARY KEY (device_oui, device_sn, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_metrics_monthly_path_time   ON pm_metrics_monthly (metric_path, end_time DESC);
CREATE INDEX idx_pm_metrics_monthly_device_time ON pm_metrics_monthly (device_oui, device_sn, end_time DESC);
CREATE INDEX idx_pm_metrics_monthly_object_ldn  ON pm_metrics_monthly (object_ldn) WHERE object_ldn IS NOT NULL;

-- ==================== pm_group_metrics_hourly：hypertable（设备组维度）====================
CREATE TABLE pm_group_metrics_hourly (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    device_group_id UUID NOT NULL,
    metric_path     TEXT NOT NULL,
    metric_type     TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value    DOUBLE PRECISION NOT NULL,
    statis_type     TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity     TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time            TIMESTAMPTZ NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ingest_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra           JSONB,
    PRIMARY KEY (id, time)
);

CREATE UNIQUE INDEX uq_pm_group_metrics_hourly_natural
    ON pm_group_metrics_hourly (device_group_id, metric_path, granularity, end_time, time);
CREATE INDEX idx_pm_group_metrics_hourly_path_time ON pm_group_metrics_hourly (metric_path, time DESC);
CREATE INDEX idx_pm_group_metrics_hourly_group_time ON pm_group_metrics_hourly (device_group_id, time DESC);

SELECT create_hypertable('pm_group_metrics_hourly', 'time', chunk_time_interval => INTERVAL '7 days');

ALTER TABLE pm_group_metrics_hourly SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_group_id,metric_type,granularity'
);

SELECT add_compression_policy('pm_group_metrics_hourly', INTERVAL '14 days');
SELECT add_retention_policy('pm_group_metrics_hourly', INTERVAL '180 days');

-- ==================== pm_group_metrics_daily：普通表 ====================
CREATE TABLE pm_group_metrics_daily (
    device_group_id UUID NOT NULL,
    metric_path     TEXT NOT NULL,
    metric_type     TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value    DOUBLE PRECISION NOT NULL,
    statis_type     TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity     TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time            TIMESTAMPTZ NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ingest_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra           JSONB,
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_group_metrics_daily_path_time ON pm_group_metrics_daily (metric_path, end_time DESC);
CREATE INDEX idx_pm_group_metrics_daily_group_time ON pm_group_metrics_daily (device_group_id, end_time DESC);

-- ==================== pm_group_metrics_weekly：普通表 ====================
CREATE TABLE pm_group_metrics_weekly (
    device_group_id UUID NOT NULL,
    metric_path     TEXT NOT NULL,
    metric_type     TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value    DOUBLE PRECISION NOT NULL,
    statis_type     TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity     TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time            TIMESTAMPTZ NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ingest_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra           JSONB,
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_group_metrics_weekly_path_time ON pm_group_metrics_weekly (metric_path, end_time DESC);
CREATE INDEX idx_pm_group_metrics_weekly_group_time ON pm_group_metrics_weekly (device_group_id, end_time DESC);

-- ==================== pm_group_metrics_monthly：普通表 ====================
CREATE TABLE pm_group_metrics_monthly (
    device_group_id UUID NOT NULL,
    metric_path     TEXT NOT NULL,
    metric_type     TEXT NOT NULL CHECK (metric_type IN ('counter', 'kpi')),
    metric_value    DOUBLE PRECISION NOT NULL,
    statis_type     TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum', 'avg', 'max', 'pct')),
    granularity     TEXT NOT NULL CHECK (granularity IN ('15min', 'hourly', 'daily', 'weekly', 'monthly')),
    time            TIMESTAMPTZ NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    ingest_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    extra           JSONB,
    PRIMARY KEY (device_group_id, metric_path, granularity, end_time)
);
CREATE INDEX idx_pm_group_metrics_monthly_path_time ON pm_group_metrics_monthly (metric_path, end_time DESC);
CREATE INDEX idx_pm_group_metrics_monthly_group_time ON pm_group_metrics_monthly (device_group_id, end_time DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT remove_retention_policy('pm_metrics_hourly', if_exists => true);
SELECT remove_compression_policy('pm_metrics_hourly', if_exists => true);
SELECT remove_retention_policy('pm_group_metrics_hourly', if_exists => true);
SELECT remove_compression_policy('pm_group_metrics_hourly', if_exists => true);

DROP TABLE IF EXISTS pm_group_metrics_monthly CASCADE;
DROP TABLE IF EXISTS pm_group_metrics_weekly CASCADE;
DROP TABLE IF EXISTS pm_group_metrics_daily CASCADE;
DROP TABLE IF EXISTS pm_group_metrics_hourly CASCADE;

DROP TABLE IF EXISTS pm_metrics_monthly CASCADE;
DROP TABLE IF EXISTS pm_metrics_weekly CASCADE;
DROP TABLE IF EXISTS pm_metrics_daily CASCADE;
DROP TABLE IF EXISTS pm_metrics_hourly CASCADE;
-- +goose StatementEnd

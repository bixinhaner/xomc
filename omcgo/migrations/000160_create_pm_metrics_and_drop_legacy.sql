-- +goose Up
-- +goose StatementBegin
-- T-0164-P3 / G3 合并 pm_counters + kpi_values 为单表 pm_metrics
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.3
--
-- 变更：
--   1. DROP MATERIALIZED VIEW pm_counters_hourly（TimescaleDB continuous aggregate，CASCADE 删底层 _materialized_hypertable_4）
--   2. DROP TABLE pm_counters / kpi_values（项目未上生产，0 行数据；schema 备份在 /tmp/pm-kpi-pre-G3-backup-20260523.sql）
--   3. CREATE TABLE pm_metrics（统一 counter + kpi 存储，metric_type 区分）
--      + hypertable on `time`，chunk_time_interval=1day
--      + compression（segmentby device_sn/metric_type/granularity）+ 7d policy
--      + retention 30d policy（G2 保留策略默认值，G5 配置化时 alter_job 联动）
--
-- pm_metrics 设计要点：
--   - metric_type CHECK ('counter','kpi')：单表存两类指标
--   - statis_type CHECK ('sum','avg','max','pct') 或 NULL：counter 必填（驱动 G5 自然桶聚合），kpi 为 NULL
--   - granularity CHECK ('15min','hourly','daily','weekly','monthly')：G3 仅 '15min'，G5 加 'hourly'+ 等
--   - 三时间字段 NOT NULL（T-0164-P4 / G4 已在 parser 层填充 FileBeginTime/FileEndTime/IngestTime）
--   - **设备唯一标识 (device_oui, device_sn)**：TR-069 标准双键
--       device_oui — TR-069 DeviceId.OUI（6 位十六进制大写厂商标识），如 '48BF74'
--       device_sn  — TR-069 DeviceId.SerialNumber（厂商内序列号），如 '1202000240194DP0026'
--       全系统设备唯一标识由这两列组合决定（详见 docs/project/plan-T-0165-system-wide-oui-sn-migration.md）
--   - 自然键唯一索引 (device_oui, device_sn, metric_path, granularity, end_time, time)：补传去重 + ON CONFLICT 幂等
--     （TimescaleDB 要求 UNIQUE 索引必须包含分区列 `time`；业务上 time = end_time，约束意义不变）
--   - PRIMARY KEY (id, time)：TS hypertable 要求 PK 含分区列
--   - object_ldn 替代旧 cell_id：更通用的 LDN 对象标识（cell / sector / 等）
--   - extra JSONB：扩展（carrier / technology / counter_group 等）
--
-- 现场聚合接口（pm/handler.go）暂走 pm_metrics granularity='15min'，G5 后路由到聚合表。

DROP MATERIALIZED VIEW IF EXISTS pm_counters_hourly CASCADE;
DROP TABLE IF EXISTS pm_counters CASCADE;
DROP TABLE IF EXISTS kpi_values CASCADE;

CREATE TABLE pm_metrics (
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

-- TimescaleDB 要求 UNIQUE 索引必须包含分区列 `time`；业务上 time = end_time，
-- 增加 time 列不改变自然键去重语义。
-- 设备唯一标识用 (device_oui, device_sn) 双键（TR-069 标准）
CREATE UNIQUE INDEX uq_pm_metrics_natural
    ON pm_metrics (device_oui, device_sn, metric_path, granularity, end_time, time);

CREATE INDEX idx_pm_metrics_path_time   ON pm_metrics (metric_path, time DESC);
CREATE INDEX idx_pm_metrics_device_time ON pm_metrics (device_oui, device_sn, time DESC);
CREATE INDEX idx_pm_metrics_ingest_time ON pm_metrics (ingest_time DESC);
CREATE INDEX idx_pm_metrics_object_ldn  ON pm_metrics (object_ldn) WHERE object_ldn IS NOT NULL;

SELECT create_hypertable('pm_metrics', 'time', chunk_time_interval => INTERVAL '1 day');

ALTER TABLE pm_metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_oui,device_sn,metric_type,granularity'
);

SELECT add_compression_policy('pm_metrics', INTERVAL '7 days');
SELECT add_retention_policy('pm_metrics', INTERVAL '30 days');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 兜底：删 pm_metrics + 还原空 schema（旧表 0 行数据，无需 restore 业务数据）
-- 完整旧 schema 见 /tmp/pm-kpi-pre-G3-backup-20260523.sql
SELECT remove_retention_policy('pm_metrics', if_exists => true);
SELECT remove_compression_policy('pm_metrics', if_exists => true);
DROP TABLE IF EXISTS pm_metrics CASCADE;

-- 旧 pm_counters 空 schema 兜底
CREATE TABLE IF NOT EXISTS pm_counters (
    "time"         TIMESTAMPTZ NOT NULL,
    device_id      UUID NOT NULL,
    cell_id        VARCHAR(32) NOT NULL DEFAULT '',
    counter_group  VARCHAR(32) NOT NULL,
    counter_name   VARCHAR(64) NOT NULL,
    counter_value  DOUBLE PRECISION NOT NULL,
    granularity    SMALLINT NOT NULL DEFAULT 15
);
CREATE INDEX IF NOT EXISTS idx_pm_counters_device_time ON pm_counters (device_id, "time" DESC);
CREATE INDEX IF NOT EXISTS idx_pm_counters_group_name  ON pm_counters (counter_group, counter_name, "time" DESC);
CREATE INDEX IF NOT EXISTS pm_counters_time_idx        ON pm_counters ("time" DESC);

-- 旧 kpi_values 空 schema 兜底
CREATE TABLE IF NOT EXISTS kpi_values (
    "time"      TIMESTAMPTZ NOT NULL,
    device_id   UUID NOT NULL,
    cell_id     VARCHAR(32) NOT NULL DEFAULT '',
    kpi_name    VARCHAR(64) NOT NULL,
    kpi_value   DOUBLE PRECISION NOT NULL,
    carrier     VARCHAR(4) NOT NULL,
    technology  VARCHAR(3) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_kpi_values_device_time ON kpi_values (device_id, "time" DESC);
CREATE INDEX IF NOT EXISTS idx_kpi_values_name        ON kpi_values (kpi_name, "time" DESC);
CREATE INDEX IF NOT EXISTS kpi_values_time_idx        ON kpi_values ("time" DESC);
-- +goose StatementEnd

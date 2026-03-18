-- PM counters table
CREATE TABLE IF NOT EXISTS pm_counters (
    time           TIMESTAMPTZ NOT NULL,
    device_id      UUID NOT NULL,
    cell_id        VARCHAR(32) NOT NULL DEFAULT '',
    counter_group  VARCHAR(32) NOT NULL,
    counter_name   VARCHAR(64) NOT NULL,
    counter_value  DOUBLE PRECISION NOT NULL,
    granularity    SMALLINT NOT NULL DEFAULT 15
);

-- TimescaleDB hypertable (skip if extension not available)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('pm_counters', 'time', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
        ALTER TABLE pm_counters SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'device_id, cell_id, counter_group, counter_name'
        );
        PERFORM add_compression_policy('pm_counters', INTERVAL '7 days', if_not_exists => TRUE);
        PERFORM add_retention_policy('pm_counters', INTERVAL '90 days', if_not_exists => TRUE);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_pm_counters_device_time ON pm_counters (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_pm_counters_group_name ON pm_counters (counter_group, counter_name, time DESC);

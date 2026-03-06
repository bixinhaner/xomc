-- KPI definitions table (regular PostgreSQL table)
CREATE TABLE IF NOT EXISTS kpi_definitions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(128) NOT NULL,
    formula      TEXT NOT NULL,
    unit         VARCHAR(16) NOT NULL,
    category     VARCHAR(32) NOT NULL,
    carrier      VARCHAR(4),
    technology   VARCHAR(3),
    counters     JSONB NOT NULL DEFAULT '[]',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- KPI values table
CREATE TABLE IF NOT EXISTS kpi_values (
    time       TIMESTAMPTZ NOT NULL,
    device_id  UUID NOT NULL,
    cell_id    VARCHAR(32) NOT NULL DEFAULT '',
    kpi_name   VARCHAR(64) NOT NULL,
    kpi_value  DOUBLE PRECISION NOT NULL,
    carrier    VARCHAR(4) NOT NULL,
    technology VARCHAR(3) NOT NULL
);

-- TimescaleDB hypertable + continuous aggregate (skip if extension not available)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('kpi_values', 'time', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_kpi_values_device_time ON kpi_values (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_kpi_values_name ON kpi_values (kpi_name, time DESC);

-- Continuous aggregate: hourly rollup (only with TimescaleDB)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        EXECUTE '
            CREATE MATERIALIZED VIEW IF NOT EXISTS pm_counters_hourly
            WITH (timescaledb.continuous) AS
            SELECT time_bucket(''1 hour'', time) AS bucket,
                   device_id, cell_id, counter_group, counter_name,
                   SUM(counter_value) AS sum_value,
                   AVG(counter_value) AS avg_value,
                   MIN(counter_value) AS min_value,
                   MAX(counter_value) AS max_value,
                   COUNT(*) AS sample_count
            FROM pm_counters
            GROUP BY bucket, device_id, cell_id, counter_group, counter_name
            WITH NO DATA';
    END IF;
END $$;

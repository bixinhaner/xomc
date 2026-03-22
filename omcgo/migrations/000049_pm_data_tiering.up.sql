-- PM Data Tiering: compression and retention policies for time-series hypertables
--
-- Problem: 90-day PM data ~3.5TB uncompressed; query performance degrades linearly.
-- Solution: TimescaleDB native compression (15-day delay) + retention (auto-drop old chunks).
--
-- Affected tables:
--   pm_counters    — update compression interval 7d→15d, add orderby time DESC
--   kpi_values     — add compression (15d) + retention (90d)
--   alarms_history — add compression (30d), keep existing 365d retention
--   mr_records     — update compression interval 7d→15d (retention 90d already set)

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        RAISE NOTICE 'TimescaleDB not available, skipping data tiering policies';
        RETURN;
    END IF;

    -- =========================================================================
    -- 1. pm_counters: update compression from 7d to 15d with explicit orderby
    -- =========================================================================

    -- Remove the old 7-day compression policy
    PERFORM remove_compression_policy('pm_counters', if_exists => TRUE);

    -- Re-enable compression with orderby (ALTER TABLE is idempotent for these settings)
    ALTER TABLE pm_counters SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'device_id, cell_id, counter_group, counter_name',
        timescaledb.compress_orderby = 'time DESC'
    );

    -- Add new 15-day compression policy
    PERFORM add_compression_policy('pm_counters', INTERVAL '15 days', if_not_exists => TRUE);

    -- Retention already set at 90 days in migration 000010; ensure it exists
    PERFORM add_retention_policy('pm_counters', INTERVAL '90 days', if_not_exists => TRUE);

    RAISE NOTICE 'pm_counters: compression=15d (segment by device_id/cell_id/counter_group/counter_name, order by time DESC), retention=90d';

    -- =========================================================================
    -- 2. kpi_values: add compression + retention (was hypertable without policies)
    -- =========================================================================

    ALTER TABLE kpi_values SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'device_id, kpi_name',
        timescaledb.compress_orderby = 'time DESC'
    );

    PERFORM add_compression_policy('kpi_values', INTERVAL '15 days', if_not_exists => TRUE);
    PERFORM add_retention_policy('kpi_values', INTERVAL '90 days', if_not_exists => TRUE);

    RAISE NOTICE 'kpi_values: compression=15d (segment by device_id/kpi_name, order by time DESC), retention=90d';

    -- =========================================================================
    -- 3. alarms_history: add compression (retention 365d already set in 000012)
    -- =========================================================================

    ALTER TABLE alarms_history SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'device_id, alarm_code',
        timescaledb.compress_orderby = 'time DESC'
    );

    PERFORM add_compression_policy('alarms_history', INTERVAL '30 days', if_not_exists => TRUE);

    -- Ensure retention exists
    PERFORM add_retention_policy('alarms_history', INTERVAL '365 days', if_not_exists => TRUE);

    RAISE NOTICE 'alarms_history: compression=30d (segment by device_id/alarm_code, order by time DESC), retention=365d';

    -- =========================================================================
    -- 4. mr_records: update compression from 7d to 15d with explicit orderby
    -- =========================================================================

    -- Remove the old 7-day compression policy
    PERFORM remove_compression_policy('mr_records', if_exists => TRUE);

    -- Re-enable compression with orderby
    ALTER TABLE mr_records SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'device_id, file_id, mr_type',
        timescaledb.compress_orderby = 'time DESC'
    );

    -- Add new 15-day compression policy
    PERFORM add_compression_policy('mr_records', INTERVAL '15 days', if_not_exists => TRUE);

    -- Retention already set at 90 days in migration 000013; ensure it exists
    PERFORM add_retention_policy('mr_records', INTERVAL '90 days', if_not_exists => TRUE);

    RAISE NOTICE 'mr_records: compression=15d (segment by device_id/file_id/mr_type, order by time DESC), retention=90d';

    -- =========================================================================
    -- 5. pm_counters_hourly: add continuous aggregate refresh policy
    -- =========================================================================

    -- Refresh policy: materialize data from 2 days ago up to 1 hour ago, every 1 hour
    -- This ensures the hourly rollup stays current without refreshing too frequently
    PERFORM add_continuous_aggregate_policy('pm_counters_hourly',
        start_offset    => INTERVAL '2 days',
        end_offset      => INTERVAL '1 hour',
        schedule_interval => INTERVAL '1 hour',
        if_not_exists   => TRUE
    );

    RAISE NOTICE 'pm_counters_hourly: continuous aggregate refresh every 1h (window: 2d ago to 1h ago)';

END $$;

-- Rollback PM Data Tiering: revert to original policies from migrations 000010-000013
--
-- This restores:
--   pm_counters    — compression back to 7d (original), retain 90d
--   kpi_values     — remove compression + retention (original had neither)
--   alarms_history — remove compression (original had only 365d retention)
--   mr_records     — compression back to 7d (original), retain 90d

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        RAISE NOTICE 'TimescaleDB not available, skipping rollback';
        RETURN;
    END IF;

    -- =========================================================================
    -- 1. pm_counters: revert compression to 7d, remove orderby override
    -- =========================================================================

    PERFORM remove_compression_policy('pm_counters', if_exists => TRUE);

    ALTER TABLE pm_counters SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'device_id, cell_id, counter_group, counter_name'
    );

    PERFORM add_compression_policy('pm_counters', INTERVAL '7 days', if_not_exists => TRUE);

    -- Retention 90d stays as it was in original migration 000010

    -- =========================================================================
    -- 2. kpi_values: remove compression + retention (original had none)
    -- =========================================================================

    PERFORM remove_compression_policy('kpi_values', if_exists => TRUE);
    PERFORM remove_retention_policy('kpi_values', if_exists => TRUE);

    -- Disable compression setting on the hypertable
    ALTER TABLE kpi_values SET (timescaledb.compress = false);

    -- =========================================================================
    -- 3. alarms_history: remove compression (original had only retention)
    -- =========================================================================

    PERFORM remove_compression_policy('alarms_history', if_exists => TRUE);

    ALTER TABLE alarms_history SET (timescaledb.compress = false);

    -- Retention 365d stays as it was in original migration 000012

    -- =========================================================================
    -- 4. mr_records: revert compression to 7d, remove orderby override
    -- =========================================================================

    PERFORM remove_compression_policy('mr_records', if_exists => TRUE);

    ALTER TABLE mr_records SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'device_id, file_id, mr_type'
    );

    PERFORM add_compression_policy('mr_records', INTERVAL '7 days', if_not_exists => TRUE);

    -- Retention 90d stays as it was in original migration 000013

    -- =========================================================================
    -- 5. pm_counters_hourly: remove continuous aggregate refresh policy
    -- =========================================================================

    PERFORM remove_continuous_aggregate_policy('pm_counters_hourly', if_not_exists => TRUE);

END $$;

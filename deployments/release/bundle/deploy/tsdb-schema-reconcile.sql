-- Pre-release baseline compatibility reconciliation.
--
-- Before versioned migrations are enabled, schema changes are folded into the
-- 000001 baseline. Goose correctly does not rerun an already-applied baseline,
-- so existing test/production databases need this small idempotent bridge.
-- Keep these statements in sync with migrations/tsdb/000001_tsdb_schema.sql.
BEGIN;

ALTER TABLE public.pm_files
    ADD COLUMN IF NOT EXISTS measurement_start timestamptz,
    ADD COLUMN IF NOT EXISTS measurement_end timestamptz;

CREATE INDEX IF NOT EXISTS idx_pm_files_measurement_slot
    ON public.pm_files (measurement_end, technology, carrier, device_id)
    WHERE measurement_end IS NOT NULL AND parsed = true;

CREATE TABLE IF NOT EXISTS public.pm_slot_health (
    slot_start timestamptz NOT NULL,
    slot_end timestamptz NOT NULL,
    technology varchar(16) NOT NULL,
    carrier varchar(16) NOT NULL,
    expected_devices bigint NOT NULL,
    received_devices bigint NOT NULL,
    coverage_ratio double precision NOT NULL,
    expected_snapshot_version text NOT NULL DEFAULT '',
    evaluated_at timestamptz NOT NULL,
    status varchar(32) NOT NULL,
    PRIMARY KEY (slot_end, technology, carrier),
    CONSTRAINT chk_pm_slot_health_window CHECK (slot_end > slot_start),
    CONSTRAINT chk_pm_slot_health_counts CHECK (
        expected_devices >= 0 AND received_devices >= 0
    ),
    CONSTRAINT chk_pm_slot_health_coverage CHECK (
        coverage_ratio >= 0 AND coverage_ratio <= 1
    ),
    CONSTRAINT chk_pm_slot_health_status CHECK (
        status IN ('complete', 'partial', 'missing', 'bootstrap_ignored')
    )
);

CREATE INDEX IF NOT EXISTS idx_pm_slot_health_latest
    ON public.pm_slot_health (technology, carrier, slot_end DESC);

-- Early observer versions marked every startup-crossing slot as ignored even
-- when the complete expected device set had already arrived. That state is
-- provably complete and safe to repair idempotently; partial startup slots stay
-- ignored because their missing segment cannot be reconstructed.
UPDATE public.pm_slot_health
SET status = 'complete'
WHERE status = 'bootstrap_ignored'
  AND coverage_ratio >= 0.98;

COMMIT;

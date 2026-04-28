-- +goose Up
-- T-0015 / R-103: License capacity & expiry enforcement
-- Adds grace period, capacity alert thresholds, and last alert tracking columns
-- to the existing licenses table; plus indexes for active+expiry queries.

ALTER TABLE licenses
    ADD COLUMN IF NOT EXISTS grace_period_days INT NOT NULL DEFAULT 0
        CHECK (grace_period_days >= 0 AND grace_period_days <= 365),
    ADD COLUMN IF NOT EXISTS capacity_alert_thresholds JSONB NOT NULL
        DEFAULT '[80, 90, 95]'::jsonb,
    ADD COLUMN IF NOT EXISTS last_capacity_alert_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_capacity_alert_threshold INT;

CREATE INDEX IF NOT EXISTS idx_licenses_active_status
    ON licenses(status) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_licenses_expiry_date
    ON licenses(expiry_date) WHERE expiry_date IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_licenses_expiry_date;
DROP INDEX IF EXISTS idx_licenses_active_status;
ALTER TABLE licenses
    DROP COLUMN IF EXISTS last_capacity_alert_threshold,
    DROP COLUMN IF EXISTS last_capacity_alert_at,
    DROP COLUMN IF EXISTS capacity_alert_thresholds,
    DROP COLUMN IF EXISTS grace_period_days;

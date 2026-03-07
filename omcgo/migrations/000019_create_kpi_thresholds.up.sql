-- KPI thresholds table for configurable threshold management
CREATE TABLE IF NOT EXISTS kpi_thresholds (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kpi_name            VARCHAR(128) NOT NULL,
    carrier             VARCHAR(16),
    technology          VARCHAR(16),
    warning_threshold   DOUBLE PRECISION,
    minor_threshold     DOUBLE PRECISION,
    major_threshold     DOUBLE PRECISION,
    critical_threshold  DOUBLE PRECISION,
    comparison          VARCHAR(16) NOT NULL DEFAULT 'gt',
    enabled             BOOLEAN NOT NULL DEFAULT true,
    description         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_kpi_thresholds_kpi_name ON kpi_thresholds (kpi_name);
CREATE INDEX IF NOT EXISTS idx_kpi_thresholds_carrier ON kpi_thresholds (carrier);
CREATE INDEX IF NOT EXISTS idx_kpi_thresholds_enabled ON kpi_thresholds (enabled);

CREATE TRIGGER trigger_kpi_thresholds_updated_at
    BEFORE UPDATE ON kpi_thresholds
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

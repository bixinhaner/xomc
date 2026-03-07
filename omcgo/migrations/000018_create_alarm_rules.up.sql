-- Alarm rules table for configurable alarm rule management
CREATE TABLE IF NOT EXISTS alarm_rules (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    alarm_code       VARCHAR(128),
    severity         INTEGER NOT NULL DEFAULT 4,
    condition_type   VARCHAR(64) NOT NULL,
    condition_config JSONB NOT NULL DEFAULT '{}',
    action_type      VARCHAR(64) NOT NULL,
    action_config    JSONB DEFAULT '{}',
    carrier          VARCHAR(16),
    technology       VARCHAR(16),
    enabled          BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alarm_rules_carrier ON alarm_rules (carrier);
CREATE INDEX IF NOT EXISTS idx_alarm_rules_enabled ON alarm_rules (enabled);
CREATE INDEX IF NOT EXISTS idx_alarm_rules_alarm_code ON alarm_rules (alarm_code);

CREATE TRIGGER trigger_alarm_rules_updated_at
    BEFORE UPDATE ON alarm_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

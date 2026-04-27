-- +goose Up
-- 迁移 000033: alarm_code → alarm_identifier 全量重命名（幂等版本）
-- 5 张告警相关表统一列名，alarms_active/history 扩容 VARCHAR(32) → VARCHAR(64)
-- 如果列已经是 alarm_identifier，则跳过 RENAME（兼容已在早期版本手动执行过的场景）。

-- 1. alarms_active
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alarms_active' AND column_name = 'alarm_code') THEN
        ALTER TABLE alarms_active RENAME COLUMN alarm_code TO alarm_identifier;
    END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE alarms_active ALTER COLUMN alarm_identifier TYPE VARCHAR(64);
DROP INDEX IF EXISTS idx_alarms_active_device_sn_code;
CREATE INDEX IF NOT EXISTS idx_alarms_active_device_identifier
    ON alarms_active(device_sn, alarm_identifier);

-- 2. alarms_history
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alarms_history' AND column_name = 'alarm_code') THEN
        ALTER TABLE alarms_history RENAME COLUMN alarm_code TO alarm_identifier;
    END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE alarms_history ALTER COLUMN alarm_identifier TYPE VARCHAR(64);

-- 3. alarm_libraries
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alarm_libraries' AND column_name = 'alarm_code') THEN
        ALTER TABLE alarm_libraries RENAME COLUMN alarm_code TO alarm_identifier;
    END IF;
END $$;
-- +goose StatementEnd
DROP INDEX IF EXISTS idx_alarm_libraries_alarm_code;
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_alarm_identifier
    ON alarm_libraries(alarm_identifier);

-- 4. alarm_rules
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alarm_rules' AND column_name = 'alarm_code') THEN
        ALTER TABLE alarm_rules RENAME COLUMN alarm_code TO alarm_identifier;
    END IF;
END $$;
-- +goose StatementEnd
DROP INDEX IF EXISTS idx_alarm_rules_alarm_code;
CREATE INDEX IF NOT EXISTS idx_alarm_rules_alarm_identifier
    ON alarm_rules(alarm_identifier);

-- 5. alarm_filters
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'alarm_filters' AND column_name = 'alarm_codes') THEN
        ALTER TABLE alarm_filters RENAME COLUMN alarm_codes TO alarm_identifiers;
    END IF;
END $$;
-- +goose StatementEnd
DROP INDEX IF EXISTS idx_alarm_filters_alarm_codes;
CREATE INDEX IF NOT EXISTS idx_alarm_filters_alarm_identifiers
    ON alarm_filters USING GIN(alarm_identifiers);

-- +goose Down
-- 迁移 000033 回滚: alarm_identifier → alarm_code

-- 5. alarm_filters
ALTER TABLE alarm_filters RENAME COLUMN alarm_identifiers TO alarm_codes;
DROP INDEX IF EXISTS idx_alarm_filters_alarm_identifiers;
CREATE INDEX IF NOT EXISTS idx_alarm_filters_alarm_codes
    ON alarm_filters USING GIN(alarm_codes);

-- 4. alarm_rules
ALTER TABLE alarm_rules RENAME COLUMN alarm_identifier TO alarm_code;
DROP INDEX IF EXISTS idx_alarm_rules_alarm_identifier;
CREATE INDEX IF NOT EXISTS idx_alarm_rules_alarm_code
    ON alarm_rules(alarm_code);

-- 3. alarm_libraries
ALTER TABLE alarm_libraries RENAME COLUMN alarm_identifier TO alarm_code;
DROP INDEX IF EXISTS idx_alarm_libraries_alarm_identifier;
CREATE INDEX IF NOT EXISTS idx_alarm_libraries_alarm_code
    ON alarm_libraries(alarm_code);

-- 2. alarms_history
ALTER TABLE alarms_history RENAME COLUMN alarm_identifier TO alarm_code;
ALTER TABLE alarms_history ALTER COLUMN alarm_code TYPE VARCHAR(32);

-- 1. alarms_active
ALTER TABLE alarms_active RENAME COLUMN alarm_identifier TO alarm_code;
ALTER TABLE alarms_active ALTER COLUMN alarm_code TYPE VARCHAR(32);
CREATE INDEX IF NOT EXISTS idx_alarms_active_device_sn_code
    ON alarms_active(device_sn, alarm_code);

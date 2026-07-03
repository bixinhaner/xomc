-- +goose Up
-- +goose StatementBegin
ALTER TABLE device_info
    ADD COLUMN IF NOT EXISTS highest_alarm_severity       smallint DEFAULT NULL;
ALTER TABLE device_info
    ADD COLUMN IF NOT EXISTS highest_severity_alarm_count smallint DEFAULT 0;
-- +goose StatementEnd

-- +goose StatementBegin
-- 回填存量：最高告警级别
WITH ranked AS (
    SELECT device_id, MIN(severity) AS min_sev
    FROM alarms_active GROUP BY device_id
)
UPDATE device_info di
SET highest_alarm_severity = r.min_sev
FROM ranked r WHERE di.device_id = r.device_id;
-- +goose StatementEnd

-- +goose StatementBegin
-- 回填存量：最高级别的告警数量
WITH counts AS (
    SELECT aa.device_id, COUNT(*) AS cnt
    FROM alarms_active aa
    JOIN device_info di ON di.device_id = aa.device_id
    WHERE aa.severity = di.highest_alarm_severity
    GROUP BY aa.device_id
)
UPDATE device_info di
SET highest_severity_alarm_count = c.cnt
FROM counts c WHERE di.device_id = c.device_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE device_info DROP COLUMN IF EXISTS highest_alarm_severity;
ALTER TABLE device_info DROP COLUMN IF EXISTS highest_severity_alarm_count;
-- +goose StatementEnd

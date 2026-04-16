-- +goose Up
-- alarms_history: add cleared_by and clear_note
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS cleared_by varchar(128);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS clear_note text DEFAULT ''::text;

-- +goose Down
ALTER TABLE alarms_history DROP COLUMN IF EXISTS clear_note;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS cleared_by;

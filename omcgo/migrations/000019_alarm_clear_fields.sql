-- +goose Up
-- alarms_active: add ack_note
ALTER TABLE alarms_active ADD COLUMN IF NOT EXISTS ack_note text DEFAULT ''::text;

-- alarms_history: add ack_note, cleared_by, clear_note
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS ack_note text DEFAULT ''::text;
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS cleared_by varchar(128);
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS clear_note text DEFAULT ''::text;

-- +goose Down
ALTER TABLE alarms_history DROP COLUMN IF EXISTS clear_note;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS cleared_by;
ALTER TABLE alarms_history DROP COLUMN IF EXISTS ack_note;
ALTER TABLE alarms_active DROP COLUMN IF EXISTS ack_note;

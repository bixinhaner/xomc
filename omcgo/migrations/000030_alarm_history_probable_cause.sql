-- +goose Up
-- alarms_history: add probable_cause column
ALTER TABLE alarms_history ADD COLUMN IF NOT EXISTS probable_cause TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE alarms_history DROP COLUMN IF EXISTS probable_cause;

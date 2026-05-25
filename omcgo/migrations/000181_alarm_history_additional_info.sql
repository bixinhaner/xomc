-- +goose Up
ALTER TABLE alarms_history
    ADD COLUMN IF NOT EXISTS additional_info JSONB DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE alarms_history
    DROP COLUMN IF EXISTS additional_info;
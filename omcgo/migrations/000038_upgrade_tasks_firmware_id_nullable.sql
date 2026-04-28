-- +goose Up
-- upgrade_tasks.firmware_id: rollback tasks do not reference a firmware file, so allow NULL.
ALTER TABLE upgrade_tasks ALTER COLUMN firmware_id DROP NOT NULL;

-- +goose Down
ALTER TABLE upgrade_tasks ALTER COLUMN firmware_id SET NOT NULL;

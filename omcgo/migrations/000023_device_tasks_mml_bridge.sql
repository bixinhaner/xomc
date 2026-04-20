-- +goose Up
-- Add columns to link device_tasks back to their parent MML task for batch fan-out

ALTER TABLE device_tasks ADD COLUMN IF NOT EXISTS parent_task_id UUID;
ALTER TABLE device_tasks ADD COLUMN IF NOT EXISTS command_index INTEGER DEFAULT 0;
ALTER TABLE device_tasks ADD COLUMN IF NOT EXISTS device_index INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_device_tasks_parent
    ON device_tasks (parent_task_id)
    WHERE parent_task_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_device_tasks_parent;
ALTER TABLE device_tasks DROP COLUMN IF EXISTS device_index;
ALTER TABLE device_tasks DROP COLUMN IF EXISTS command_index;
ALTER TABLE device_tasks DROP COLUMN IF EXISTS parent_task_id;

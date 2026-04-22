-- +goose Up
-- 将 device_tasks.parent_task_id 重命名为 source_id，语义从"同类型父任务 ID"
-- 扩展为"通用任务来源 ID"（例如 source='mml' 时为 mml_tasks.id，后续可支持其它来源）。
-- PostgreSQL 对分区表的 RENAME COLUMN 会自动级联到所有分区，这里直接在父表操作即可。

ALTER TABLE device_tasks RENAME COLUMN parent_task_id TO source_id;
ALTER INDEX idx_device_tasks_parent RENAME TO idx_device_tasks_source_id;

-- +goose Down
ALTER INDEX idx_device_tasks_source_id RENAME TO idx_device_tasks_parent;
ALTER TABLE device_tasks RENAME COLUMN source_id TO parent_task_id;

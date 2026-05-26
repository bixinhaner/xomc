-- +goose Up
-- 阶段 3: 自定义聚合"聚合到组"
-- 给 adhoc 任务（pm_tasks 表 task_subtype='adhoc_aggregation' 行）加 dimension 列，
-- 支持新维度 'aggregate_group'：N 个 SN 临时组聚合成一条结果。
--
-- 默认 'device' 与既有任务兼容（每设备保留一条结果）。
ALTER TABLE pm_tasks
    ADD COLUMN IF NOT EXISTS dimension TEXT NOT NULL DEFAULT 'device';

ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_dimension;
ALTER TABLE pm_tasks ADD CONSTRAINT chk_pm_tasks_dimension
    CHECK (dimension IN ('device', 'aggregate_group'));

-- +goose Down
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_dimension;
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS dimension;

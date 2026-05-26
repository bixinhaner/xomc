-- +goose Up
-- ============================================================
-- 000198_mr_customize_task_target_device_sns.sql
-- 给 mr_customize_task 加 target_device_sns 列保存用户在 UI 选中的设备 SN 列表。
--
-- 用户反馈（2026-05-26）：MR 任务需要像其他任务一样让用户选目标设备，
-- 不再"作用于所有 enabled cell"。
--
-- 设计：
--   - 用 TEXT[]（PG 原生数组），不引入新表
--   - scheduler.materializeTargets 在 task 开启时把这些 SN 展开为 cells
--     （JOIN mr_device_mappings WHERE device_sn = ANY(target_device_sns) AND enabled=true）
--   - 列允许 NULL 是为了历史数据兼容；新建任务时 Service 层强制非空
-- ============================================================

ALTER TABLE mr_customize_task
    ADD COLUMN IF NOT EXISTS target_device_sns TEXT[];

COMMENT ON COLUMN mr_customize_task.target_device_sns IS
    '用户选定的目标设备 SN 列表；scheduler 开启任务时 JOIN mr_device_mappings 展开为 cells';

-- +goose Down
ALTER TABLE mr_customize_task DROP COLUMN IF EXISTS target_device_sns;

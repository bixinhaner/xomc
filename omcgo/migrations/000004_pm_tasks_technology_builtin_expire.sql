-- +goose Up
-- T-0182 聚合任务模型与维度扩展（基础）
--
-- 1. 给 pm_tasks 加三列承载新设计的任务定义字段：
--    - technology：任务制式（lte/nr/gsm），建后不可改；用于范围制式过滤与内置任务标记。
--      可空，老任务无制式约束保持 NULL。
--    - is_builtin：内置任务标记（4 维度 × 3 制式 = 12 个内置任务在 T-0184 预置）。默认 false。
--    - expire_days：非持续型任务过期天数（默认 60，约束的是"任务定义"层面，
--      与结果数据按 PM 统一保留期清理两层口径分离）。
--
-- 2. 扩 dimension 的 CHECK 约束，加入 'product' / 'band' / 'network' 三个新维度。
--    product 维度在本任务实现（照搬 device_group 模式按 product_id 分组）；
--    band 维度仅入枚举/约束，聚合实现拆到 T-0183；
--    network 维度（全网汇总）由 seed/000010 预置内置任务即会写入，故必须在此一并放开，
--    否则存量库（已含 network 内置任务行）重跑本迁移会被旧约束拒绝（SQLSTATE 23514）。
--    000006 仍保留同名约束的幂等重建，对全新库与存量库都安全。
--    保留既有 'device' / 'aggregate_group'（adhoc 临时组）与 'device_group'（设备组）。
ALTER TABLE pm_tasks
    ADD COLUMN IF NOT EXISTS technology  TEXT,
    ADD COLUMN IF NOT EXISTS is_builtin  BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS expire_days INTEGER NOT NULL DEFAULT 60;

ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_dimension;
ALTER TABLE pm_tasks ADD CONSTRAINT chk_pm_tasks_dimension
    CHECK (dimension IN ('device', 'aggregate_group', 'device_group', 'product', 'band', 'network'));

ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_technology;
ALTER TABLE pm_tasks ADD CONSTRAINT chk_pm_tasks_technology
    CHECK (technology IS NULL OR technology IN ('lte', 'nr', 'gsm'));

-- +goose Down
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_technology;
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_dimension;
ALTER TABLE pm_tasks ADD CONSTRAINT chk_pm_tasks_dimension
    CHECK (dimension IN ('device', 'aggregate_group'));

ALTER TABLE pm_tasks
    DROP COLUMN IF EXISTS expire_days,
    DROP COLUMN IF EXISTS is_builtin,
    DROP COLUMN IF EXISTS technology;

-- +goose Up
-- T-0184 内置任务预置 + 调度：新增 'network'（全网）聚合维度。
--
-- 扩 dimension 的 CHECK 约束，加入 'network'（全网汇总成一条总线，仅制式过滤、无实体键）。
-- 'device_group' 已在 000220 的白名单内，无需重复添加。
-- 保留既有 'device' / 'aggregate_group' / 'device_group' / 'product' / 'band'。
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_dimension;
ALTER TABLE pm_tasks ADD CONSTRAINT chk_pm_tasks_dimension
    CHECK (dimension IN ('device', 'aggregate_group', 'device_group', 'product', 'band', 'network'));

-- +goose Down
-- 回滚到 000220 的白名单（不含 'network'）。
ALTER TABLE pm_tasks DROP CONSTRAINT IF EXISTS chk_pm_tasks_dimension;
ALTER TABLE pm_tasks ADD CONSTRAINT chk_pm_tasks_dimension
    CHECK (dimension IN ('device', 'aggregate_group', 'device_group', 'product', 'band'));

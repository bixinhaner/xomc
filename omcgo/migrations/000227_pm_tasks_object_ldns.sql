-- +goose Up
-- T-0193 自选设备下钻到小区/PLMN —— 任务小区/PLMN 白名单字段。
--
-- 给 pm_tasks 加一个可空数组列 object_ldns，承载"小区/PLMN 白名单"：
--   - NULL / 空数组 = 不过滤 = 全小区（向后兼容旧任务，旧任务无此列照旧全量）。
--   - 非空 = 只看选中的 object_ldn（完整字符串，形如 'Cellid=111172245,PLMN=46068'）。
--
-- 性质：纯查看级过滤（设计 §3"全存底层、查看收口"），底层聚合/落库不因白名单改变。
-- 加法迁移，不加 CHECK、不回填。
ALTER TABLE pm_tasks
    ADD COLUMN IF NOT EXISTS object_ldns TEXT[];

-- +goose Down
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS object_ldns;

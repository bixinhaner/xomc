-- +goose Up
-- ============================================================
-- 000114_device_tasks_path_translation_miss.sql
-- device_tasks 加路径翻译 miss 元数据（Stage 1 — fanout 入队前翻译）
--
-- 设计：用户决策 2026-05-16 — MML fanout 阶段把 standardPath 翻译为 device
-- 对应的 privatePath；翻译失败（未命中 ParamModel mapping）的 path 用
-- standardPath 兜底下发，并在 device_task 上打 miss 标记，前端任务详情
-- 页据此显示"路径翻译警告"标签。
--
-- 两列：
--   has_path_translation_miss BOOLEAN — 本 device_task 是否有任何 path
--     翻译未命中（true=有 fallback）；按 device 粒度聚合，前端筛选 / 标签用
--   path_translation_miss_count INT  — 具体 miss 的 path 数量，运维排查时
--     可对照 task.params.names 找出哪些是 standardPath 直透
-- ============================================================

ALTER TABLE device_tasks
    ADD COLUMN IF NOT EXISTS has_path_translation_miss   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS path_translation_miss_count INT     NOT NULL DEFAULT 0;

COMMENT ON COLUMN device_tasks.has_path_translation_miss IS
    'MML fanout 翻译标记：true=本任务中至少一个 standardPath 未找到 device 对应的 '
    'privatePath，已 fallback 用 standardPath 下发；前端任务详情页据此显示警告。';

COMMENT ON COLUMN device_tasks.path_translation_miss_count IS
    'MML fanout 翻译未命中的 path 数量；与 task.params.names 长度对照可定位具体 '
    'standardPath 直透条目。0 = 全部命中或本任务非 MML 来源（默认值）。';

-- 部分索引仅索引 miss 的少量 row，扫表查"有问题的任务"很快。
CREATE INDEX IF NOT EXISTS idx_device_tasks_has_path_translation_miss
    ON device_tasks (created_at DESC)
    WHERE has_path_translation_miss = true;


-- +goose Down
DROP INDEX IF EXISTS idx_device_tasks_has_path_translation_miss;
ALTER TABLE device_tasks
    DROP COLUMN IF EXISTS path_translation_miss_count,
    DROP COLUMN IF EXISTS has_path_translation_miss;

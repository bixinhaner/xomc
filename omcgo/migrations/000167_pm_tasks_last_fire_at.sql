-- +goose Up
-- G7-Gap-9: continuous adhoc 任务 lossless 补跑
--
-- 增 last_fire_at 列追踪"上一次 cron 触发时刻"（由 ContinuousScheduler 写入），
-- 与 updated_at（worker 状态切换时刷新）解耦。
-- 调度器停机后启动：从 last_fire_at 开始一次一次推 Next(cron)，每个 worker 完成后
-- 下一个 sweep 继续推进一个 cron 窗口，直至追平 NOW()。
--
-- 列可为 NULL（老行无值），调度器对 NULL 走 fallback：使用 created_at 作起点。
ALTER TABLE pm_tasks ADD COLUMN IF NOT EXISTS last_fire_at TIMESTAMPTZ;

COMMENT ON COLUMN pm_tasks.last_fire_at IS 'G7 continuous 任务上次 cron 触发时刻；ContinuousScheduler 写入，worker 不动。';

-- +goose Down
ALTER TABLE pm_tasks DROP COLUMN IF EXISTS last_fire_at;

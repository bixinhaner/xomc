-- +goose Up
-- +goose StatementBegin
-- T-0164 收尾 G8-Gap-1：cron 触发持久化 + 启动补跑
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.8（G8 cron 错过补跑）
-- 实施 plan：docs/project/plan-T-0164-followup-gaps.md §G8-Gap-1
--
-- 用途：worker 重启 / 停机后，cron 调度器需要知道"上次成功触发到哪了"以补跑漏桶。
-- 没有这张表 = 停机期间所有 cron 触发完全丢失（robfig/cron 是内存调度器，无持久化）。
--
-- 设计要点：
--   - PK = job_type（每种 cron job 一行；多 worker 用 UPSERT 单 SQL CAS 防双触发）
--   - last_triggered_at = 上次成功 enqueue async_jobs 的时刻
--   - cron_expr = 当前生效的 cron expr（变更时也记录便于排查）
--   - updated_at 自动更新触发器

CREATE TABLE async_jobs_cron_state (
    job_type           TEXT PRIMARY KEY,
    cron_expr          TEXT NOT NULL,
    last_triggered_at  TIMESTAMPTZ NOT NULL,
    last_bucket_end    TIMESTAMPTZ,   -- 上次触发的 bucket 时间窗右界（便于按时间窗对齐查询）
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trigger_async_jobs_cron_state_updated_at
    BEFORE UPDATE ON async_jobs_cron_state
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS async_jobs_cron_state CASCADE;
-- +goose StatementEnd

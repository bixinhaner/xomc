-- +goose Up
-- parameter_sync_runs 建表以来只有主键 (id) 索引，但 findActiveRun /
-- GetActiveRunByDevice（pg_repository.go）的热点查询按 device_id + status 过滤、
-- 按 started_at 排序，一直在做全表顺序扫描 + 排序。线上巡检（78 服务器）实测：
-- postgres 长期 CPU 400%/400（4 核打满）时，一条等价查询在仅 1.4 万行的表上
-- 跑了 45+ 秒未完成。该表随同步历史持续增长，问题只会恶化，需补索引。
CREATE INDEX IF NOT EXISTS idx_parameter_sync_runs_device_status
    ON public.parameter_sync_runs USING btree (device_id, status, started_at DESC);

-- +goose Down
DROP INDEX IF EXISTS public.idx_parameter_sync_runs_device_status;

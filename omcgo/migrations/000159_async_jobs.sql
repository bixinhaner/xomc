-- +goose Up
-- +goose StatementBegin
-- T-0164-P8 / G8 通用任务框架：async_jobs 表
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.8
--
-- 用途：进程级通用异步任务总线，承载 G5 cron 实例（小时/日/周/月聚合）+ 未来批量计算任务。
-- 状态机：pending → running → succeeded / failed / zombie / canceled
--   - 心跳上报：running 中每 30s 更新 heartbeat_at
--   - 僵尸检测：sweeper 每 60s 扫 heartbeat_at < now() - 5min 的 running 任务，重置 pending
--   - 重启续跑：worker 崩溃后 conn 归还触发 PG session 结束，advisory lock 自动释放；
--     僵尸任务下一 sweeper 回合自动 pending 让其他 worker 抢
--   - cron 触发：scheduler 内 cron tick（@由 LeaderElector 单实例选主，避免多 worker 重复入队）
--
-- 与 G7 自定义聚合任务的分工（设计 §4.7/§4.8 锁定）：
--   - G7 走 PM 模块 per-module 表 pm_tasks（独立 worker 池）
--   - G8 仅承载"系统层"任务：G5 cron + 未来批量计算

CREATE TABLE async_jobs (
    id              UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    job_type        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','running','succeeded','failed','zombie','canceled')),
    schedule_expr   TEXT,                       -- cron expression（仅 cron 触发的任务有值）
    scheduled_at    TIMESTAMPTZ NOT NULL,       -- 计划执行时刻（cron 解析后或手动设定）
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    heartbeat_at    TIMESTAMPTZ,                -- running 期间每 30s 更新
    lock_owner      TEXT,                       -- worker 进程 ID (hostname-pid)
    attempt         INT NOT NULL DEFAULT 1,
    max_attempts    INT NOT NULL DEFAULT 3,
    payload         JSONB,                      -- job 参数（如聚合的时间窗）
    result          JSONB,                      -- 成功时的结果（如聚合行数）
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 关键查询：LockNextPending(job_type) 用 (status, scheduled_at, job_type) 三列定位
CREATE INDEX idx_async_jobs_pending_pickup
    ON async_jobs (job_type, status, scheduled_at)
    WHERE status = 'pending';

-- 关键查询：ListZombies 扫 running 且 heartbeat_at 过期
CREATE INDEX idx_async_jobs_zombie_check
    ON async_jobs (status, heartbeat_at)
    WHERE status = 'running';

-- 历史排查：按 job_type + created_at 倒序看最近运行历史
CREATE INDEX idx_async_jobs_type_created
    ON async_jobs (job_type, created_at DESC);

-- updated_at 触发器（与项目其他表一致）— 复用 000001 已创建的 update_updated_at_column()
CREATE TRIGGER trg_async_jobs_updated_at
    BEFORE UPDATE ON async_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE  async_jobs IS 'T-0164-P8 / G8 通用异步任务总线（承载 G5 cron + 未来批量计算）';
COMMENT ON COLUMN async_jobs.status IS 'pending/running/succeeded/failed/zombie/canceled';
COMMENT ON COLUMN async_jobs.heartbeat_at IS 'running 期间每 30s 更新；sweeper 用 heartbeat_at < now()-5min 识别僵尸';
COMMENT ON COLUMN async_jobs.lock_owner IS 'worker 进程标识 hostname-pid，便于排查"哪个 worker 抢到任务"';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_async_jobs_updated_at ON async_jobs;
DROP INDEX IF EXISTS idx_async_jobs_type_created;
DROP INDEX IF EXISTS idx_async_jobs_zombie_check;
DROP INDEX IF EXISTS idx_async_jobs_pending_pickup;
DROP TABLE IF EXISTS async_jobs CASCADE;
-- +goose StatementEnd

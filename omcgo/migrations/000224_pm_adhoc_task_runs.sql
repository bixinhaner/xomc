-- +goose Up
-- T-0186: 自定义聚合任务运行历史表。
-- adhoc 任务（pm_tasks task_subtype='adhoc_aggregation' 行）刻意不进 async_jobs，
-- 且 pm_tasks 只在 status 循环、无逐次运行记录。本表为每次 worker 执行落一行运行快照，
-- 供任务详情运行历史视图（编号/粒度/维度/时间窗/状态/入队·完成时间/失败原因）查询。
-- task_id 逻辑引用 pm_tasks(id)，不建外键（pm_tasks 共享于 partition 场景，参见 5.5.3）。
CREATE TABLE IF NOT EXISTS pm_adhoc_task_runs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID        NOT NULL,                 -- 逻辑引用 pm_tasks.id
    run_seq       INT         NOT NULL,                 -- 按任务自增编号（该任务第 N 次执行）
    granularity   TEXT        NOT NULL DEFAULT '',      -- 运行时粒度快照（单粒度，多粒度逗号拼）
    dimension     TEXT        NOT NULL DEFAULT '',      -- 运行时维度快照
    window_start  TIMESTAMPTZ,                          -- 源数据时窗起（continuous 开窗为 NULL）
    window_end    TIMESTAMPTZ,                          -- 源数据时窗止
    status        TEXT        NOT NULL,                 -- running / succeeded / failed
    queued_at     TIMESTAMPTZ,                          -- 进入执行队列时刻（best-effort）
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),   -- 开始执行时刻
    finished_at   TIMESTAMPTZ,                          -- 完成时刻（running 行为 NULL）
    error         TEXT        NOT NULL DEFAULT '',      -- 失败原因（failed 行非空）
    rows_total    INT         NOT NULL DEFAULT 0,       -- 写入结果行数
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 运行历史按 started_at 倒序分页查询
CREATE INDEX IF NOT EXISTS idx_pm_adhoc_task_runs_task_started
    ON pm_adhoc_task_runs (task_id, started_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_pm_adhoc_task_runs_task_started;
DROP TABLE IF EXISTS pm_adhoc_task_runs;

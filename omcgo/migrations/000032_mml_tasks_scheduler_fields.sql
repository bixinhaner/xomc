-- +goose Up
-- ============================================================
-- 000032_mml_tasks_scheduler_fields.sql
-- P2/P3：MML 任务调度器（docs/design/mml-task-flow-design-20260424.md §4.2/§4.3）
--
--   · next_trigger_at —— 下一次 Scheduler 应拉起 fanout 的时刻。
--     scheduled 任务 = scheduled_at（一次性）；
--     periodic 任务 = 今日/明日 period_time 的 UTC 时刻（每次触发后滚动）。
--     Scheduler 以此列加 FOR UPDATE SKIP LOCKED 做幂等取任务。
--
--   · parent_task_id —— periodic 模板任务生成的子实例回指模板行，
--     历史执行列表 API（P4 §C11）按 script_id 或 parent_task_id 反查实例。
-- ============================================================

ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS next_trigger_at TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS parent_task_id  UUID;

-- Scheduler 扫描索引：仅索引 next_trigger_at 非空且 status='pending' 的行，
-- 大部分历史记录（completed/failed）不进索引，保持索引体积可控。
CREATE INDEX IF NOT EXISTS idx_mml_tasks_next_trigger
    ON mml_tasks (next_trigger_at)
    WHERE next_trigger_at IS NOT NULL AND status = 'pending';

-- 历史执行列表查询索引（P4 配套）
CREATE INDEX IF NOT EXISTS idx_mml_tasks_parent_task
    ON mml_tasks (parent_task_id)
    WHERE parent_task_id IS NOT NULL;

COMMENT ON COLUMN mml_tasks.next_trigger_at IS
    'MML 任务下一次触发时刻：scheduled 为一次性值；periodic 为模板行的下次 period_time 命中时刻（由 Scheduler 滚动更新）';
COMMENT ON COLUMN mml_tasks.parent_task_id IS
    'periodic 模板行生成的子实例指向模板 mml_task.id；非 periodic 子实例或模板本身为 NULL';

-- +goose Down
DROP INDEX IF EXISTS idx_mml_tasks_parent_task;
DROP INDEX IF EXISTS idx_mml_tasks_next_trigger;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS parent_task_id;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS next_trigger_at;

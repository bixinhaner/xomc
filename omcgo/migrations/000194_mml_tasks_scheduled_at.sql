-- +goose Up
-- ============================================================
-- 000194_mml_tasks_scheduled_at.sql
--
-- 补回 mml_tasks.scheduled_at —— 该列在 internal/mml/pg_repository.go
-- taskColumns（line 74）/ INSERT（line 673）/ UPDATE（line 746）/
-- SELECT（line 1195）四处直接引用，但 main/ 任何 migration 都没有
-- 真正 ADD COLUMN 它。
--
-- 历史：
--   · 原由 seed/000004_mml_enhance.sql 在 idempotent DO $$ 块里 ALTER ADD
--     COLUMN 加入；
--   · 000004 在 2026-05-22 整体降级为 noop（注释声称 execute_type /
--     scheduled_at 等已由 main migrations 接管，但实际只接管了 execute_type，
--     scheduled_at 没人接走）；
--   · 本地老环境因当年跑过 000004 而带这一列；fresh deploy（如 172.19.1.73）
--     没有它 → /mml/console/execute-statements-structured 失败：
--       create mml_task: ERROR: column "scheduled_at" of relation
--       "mml_tasks" does not exist (SQLSTATE 42703)
--
-- 类型对齐 Go 模型 MMLTask.ScheduledAt（*time.Time）→ 可空 TIMESTAMPTZ。
-- 默认 NULL = "立即执行 / 无定时计划"，与 execute_type='immediate' 语义一致。
-- IF NOT EXISTS 保证已经手工补过的老环境二次执行无副作用。
-- ============================================================

ALTER TABLE mml_tasks
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;

COMMENT ON COLUMN mml_tasks.scheduled_at IS
    'MML 任务计划执行时刻：execute_type=scheduled 时为一次性执行时间；'
    'execute_type=immediate / periodic / 立即派发场景为 NULL。'
    'Scheduler 与 next_trigger_at 配合使用（next_trigger_at 是滚动触发时间，'
    'scheduled_at 是原始计划时间，便于审计 / 列表展示）。';


-- +goose Down
ALTER TABLE mml_tasks
    DROP COLUMN IF EXISTS scheduled_at;

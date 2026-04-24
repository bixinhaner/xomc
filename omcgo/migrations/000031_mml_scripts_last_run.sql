-- +goose Up
-- ============================================================
-- 000031_mml_scripts_last_run.sql
-- P1：MML 任务全链路处理方案（docs/design/mml-task-flow-design-20260424.md）
--   · last_run_status —— 脚本最近一次执行的终态（completed/failed/partial）
--   · last_run_at     —— 脚本最近一次执行的完成时刻
--   每次执行的详情由独立的 mml_tasks 行持有，脚本层只留"最近一次"指针，
--   前端脚本详情页用于快速显示最近运行结果，同时为 §4.4 历史执行列表 API
--   （GET /api/v1/mml/scripts/:id/runs）提供主索引基础。
-- ============================================================

ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS last_run_status VARCHAR(20);
ALTER TABLE mml_scripts ADD COLUMN IF NOT EXISTS last_run_at     TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_mml_scripts_last_run_at
    ON mml_scripts(last_run_at DESC) WHERE last_run_at IS NOT NULL;

COMMENT ON COLUMN mml_scripts.last_run_status IS
    'MML 脚本最近一次关联 mml_task 的终态：completed/failed/partial。每次执行详情查 mml_tasks。';
COMMENT ON COLUMN mml_scripts.last_run_at IS
    'MML 脚本最近一次执行完成时刻（与 last_run_status 对应）。';

-- +goose Down
DROP INDEX IF EXISTS idx_mml_scripts_last_run_at;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS last_run_at;
ALTER TABLE mml_scripts DROP COLUMN IF EXISTS last_run_status;

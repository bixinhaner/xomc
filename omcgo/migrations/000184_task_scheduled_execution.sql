-- +goose Up
-- ============================================================
-- 000184_task_scheduled_execution.sql
-- 文件传输任务新增"定时执行"模式
--
-- 背景：transfer / software 模块现有任务创建仅支持
--   · 立即执行（status=in_progress 直接派发）
--   · 挂起（status=pending + create_status=active 等手动 Resume）
-- 缺少"指定时间自动启动"能力。本迁移加上 scheduled_at 列 + 索引，
-- 配合 create_status='timing'（000033 已加入 CHECK 约束）完成三态模型：
--   · 立即:  status=in_progress, scheduled_at IS NULL
--   · 挂起:  status=pending, create_status=active, scheduled_at IS NULL
--   · 定时:  status=pending, create_status=timing, scheduled_at = 计划时间
--
-- 5 张表 schema 来自 000144 (CREATE TABLE LIKE upgrade_tasks)；ALTER 时
-- 旧表（upgrade_tasks）先加，4 张派生表逐一加（LIKE 不会跟随后续修改）。
-- 索引选用 partial index 仅覆盖待触发行，规模可控（单表 < 1k 行）。
-- ============================================================

ALTER TABLE upgrade_tasks
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;
ALTER TABLE config_backup_tasks
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;
ALTER TABLE config_restore_tasks
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;
ALTER TABLE runtime_log_collect_tasks
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;
ALTER TABLE fault_log_collect_tasks
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;

-- Partial index：调度器每 30s 扫描"已到期且待触发"的任务。
-- 索引按 scheduled_at 升序，最早到期排在前面，便于 LIMIT N 切批处理。
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_due_scheduled
    ON upgrade_tasks (scheduled_at)
    WHERE create_status = 'timing' AND status = 'pending';
CREATE INDEX IF NOT EXISTS idx_config_backup_tasks_due_scheduled
    ON config_backup_tasks (scheduled_at)
    WHERE create_status = 'timing' AND status = 'pending';
CREATE INDEX IF NOT EXISTS idx_config_restore_tasks_due_scheduled
    ON config_restore_tasks (scheduled_at)
    WHERE create_status = 'timing' AND status = 'pending';
CREATE INDEX IF NOT EXISTS idx_runtime_log_collect_tasks_due_scheduled
    ON runtime_log_collect_tasks (scheduled_at)
    WHERE create_status = 'timing' AND status = 'pending';
CREATE INDEX IF NOT EXISTS idx_fault_log_collect_tasks_due_scheduled
    ON fault_log_collect_tasks (scheduled_at)
    WHERE create_status = 'timing' AND status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_upgrade_tasks_due_scheduled;
DROP INDEX IF EXISTS idx_config_backup_tasks_due_scheduled;
DROP INDEX IF EXISTS idx_config_restore_tasks_due_scheduled;
DROP INDEX IF EXISTS idx_runtime_log_collect_tasks_due_scheduled;
DROP INDEX IF EXISTS idx_fault_log_collect_tasks_due_scheduled;

ALTER TABLE upgrade_tasks            DROP COLUMN IF EXISTS scheduled_at;
ALTER TABLE config_backup_tasks      DROP COLUMN IF EXISTS scheduled_at;
ALTER TABLE config_restore_tasks     DROP COLUMN IF EXISTS scheduled_at;
ALTER TABLE runtime_log_collect_tasks DROP COLUMN IF EXISTS scheduled_at;
ALTER TABLE fault_log_collect_tasks  DROP COLUMN IF EXISTS scheduled_at;

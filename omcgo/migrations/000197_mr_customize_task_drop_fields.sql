-- +goose Up
-- ============================================================
-- 000197_mr_customize_task_drop_fields.sql
-- F05 MR 任务模型简化：删除 note / operator_code 列
--
-- 用户反馈（2026-05-25）：MR 任务不应让用户填 note / operator_code，
-- 也不需要 multi-tenant 隔离。creator 列保留，由后端从登录态自动填，
-- 不再走入参。
--
-- 同步删依赖的 operator_code 索引；start_time / end_time / status 索引保留。
-- ============================================================

DROP INDEX IF EXISTS idx_mr_task_operator;

ALTER TABLE mr_customize_task DROP COLUMN IF EXISTS note;
ALTER TABLE mr_customize_task DROP COLUMN IF EXISTS operator_code;

-- +goose Down
-- 回滚：补回两列（值无法恢复，新数据为 NULL；operator_code 给个空字符串占位）
ALTER TABLE mr_customize_task ADD COLUMN IF NOT EXISTS note TEXT;
ALTER TABLE mr_customize_task ADD COLUMN IF NOT EXISTS operator_code VARCHAR(16) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_mr_task_operator
    ON mr_customize_task (operator_code, created_at DESC);

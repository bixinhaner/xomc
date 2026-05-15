-- +goose Up
-- ============================================================
-- 000099_upgrade_tasks_download_file_type.sql
-- 软件升级任务增加 Download FileType 覆盖字段
-- ============================================================

ALTER TABLE upgrade_tasks
    ADD COLUMN IF NOT EXISTS download_file_type TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE upgrade_tasks
    DROP COLUMN IF EXISTS download_file_type;

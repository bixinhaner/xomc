-- +goose Up
-- backup_restore_file 加 task_id 字段隔离不同任务的同名文件：
-- 之前唯一键 (serial_number, file_name) 会让同 SN 下不同任务上传同名文件
-- 互相覆盖（典型：设备厂商每次都用同样的 "mib-home-fap.nv" 名上传）。
-- task_id 来源于 upgrade_tasks.id（UFTE 任务的主任务 UUID）。
ALTER TABLE backup_restore_file
    ADD COLUMN IF NOT EXISTS task_id UUID;

-- 旧唯一约束 (serial_number, file_name) 改成 (serial_number, task_id, file_name)。
-- 旧数据 task_id 为 NULL — PostgreSQL UNIQUE 默认允许多个 NULL，所以历史孤儿
-- 行不会阻塞新约束。需要时运维可通过 backfill 把旧行的 task_id 补齐。
ALTER TABLE backup_restore_file
    DROP CONSTRAINT IF EXISTS backup_restore_file_sn_file_uk;

ALTER TABLE backup_restore_file
    ADD CONSTRAINT backup_restore_file_sn_task_file_uk
        UNIQUE (serial_number, task_id, file_name);

CREATE INDEX IF NOT EXISTS idx_backup_restore_file_task_id
    ON backup_restore_file (task_id) WHERE task_id IS NOT NULL;


-- +goose Down
DROP INDEX IF EXISTS idx_backup_restore_file_task_id;

ALTER TABLE backup_restore_file
    DROP CONSTRAINT IF EXISTS backup_restore_file_sn_task_file_uk;

ALTER TABLE backup_restore_file
    ADD CONSTRAINT backup_restore_file_sn_file_uk
        UNIQUE (serial_number, file_name);

ALTER TABLE backup_restore_file
    DROP COLUMN IF EXISTS task_id;

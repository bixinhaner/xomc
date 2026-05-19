-- M1 of backup-restore-alignment-plan-20260518.md.
--
-- Goal: bring `backup_tasks` / `restore_tasks` / `backup_schedules` closer to
-- the operator-spec schema (`backup_restore_task` / `backup_restore_file` /
-- `backup_period_task`) WITHOUT breaking any existing column or external API.
--
-- Strategy (decision D-1 / D-2 in the plan): keep UUID PKs and the split
-- backup_tasks / restore_tasks tables, but expose spec-compatible columns
-- alongside:
--   * `task_seq BIGSERIAL UNIQUE` — int task_id the spec demands
--   * `task_name`                 — operator-supplied display name
--   * `task_result SMALLINT`      — 1=成功 / 2=失败 (derived at completion)
--   * `operator_code VARCHAR(8)`  — 运营商代码 (映射 devices.carrier)
--   * `create_user`               — 创建人 (业务侧已存于 restore_tasks.created_by)
-- All new columns are NULLABLE so historical rows keep working unchanged.
--
-- Also introduces `backup_restore_file`, the operator-spec表 to track each
-- successfully landed backup (SN/file_name/md5/size/operator_code/update_time)
-- so M3 周期备份与运维查询有数据源。
--
-- `backup_schedules` gets `is_enable SMALLINT GENERATED` so规范字段直接可查
-- (0/1) 而不必读 boolean。

-- +goose Up

-- =====================================================================
-- 1. backup_tasks alignment columns
-- =====================================================================
ALTER TABLE backup_tasks
    ADD COLUMN IF NOT EXISTS task_seq      BIGSERIAL,
    ADD COLUMN IF NOT EXISTS task_name     VARCHAR(200),
    ADD COLUMN IF NOT EXISTS task_result   SMALLINT
        CHECK (task_result IS NULL OR task_result IN (1, 2)),
    ADD COLUMN IF NOT EXISTS operator_code VARCHAR(8),
    ADD COLUMN IF NOT EXISTS create_user   VARCHAR(64);

-- BIGSERIAL on an existing table won't auto-add UNIQUE; do it explicitly so
-- task_seq behaves like the spec's int PK (mono-increasing, unique).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename = 'backup_tasks'
          AND indexname = 'backup_tasks_task_seq_key'
    ) THEN
        ALTER TABLE backup_tasks ADD CONSTRAINT backup_tasks_task_seq_key UNIQUE (task_seq);
    END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_backup_tasks_operator_code ON backup_tasks(operator_code);
CREATE INDEX IF NOT EXISTS idx_backup_tasks_create_user   ON backup_tasks(create_user);

-- =====================================================================
-- 2. restore_tasks alignment columns
-- =====================================================================
ALTER TABLE restore_tasks
    ADD COLUMN IF NOT EXISTS task_seq      BIGSERIAL,
    ADD COLUMN IF NOT EXISTS task_name     VARCHAR(200),
    ADD COLUMN IF NOT EXISTS task_result   SMALLINT
        CHECK (task_result IS NULL OR task_result IN (1, 2)),
    ADD COLUMN IF NOT EXISTS operator_code VARCHAR(8),
    ADD COLUMN IF NOT EXISTS create_user   VARCHAR(64);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename = 'restore_tasks'
          AND indexname = 'restore_tasks_task_seq_key'
    ) THEN
        ALTER TABLE restore_tasks ADD CONSTRAINT restore_tasks_task_seq_key UNIQUE (task_seq);
    END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_restore_tasks_operator_code ON restore_tasks(operator_code);
CREATE INDEX IF NOT EXISTS idx_restore_tasks_create_user   ON restore_tasks(create_user);

-- =====================================================================
-- 3. backup_restore_file — spec-mandated metadata table
-- =====================================================================
-- One row per successfully landed backup config file. Upserted by
-- backup.FilePathRecorder when SubjectBackupFileReceived fires. MD5 sourced
-- from MinIO ETag (single-part PutObject; backup files are small XML).
CREATE TABLE IF NOT EXISTS backup_restore_file (
    id              BIGSERIAL   PRIMARY KEY,
    serial_number   VARCHAR(64) NOT NULL,
    file_name       TEXT        NOT NULL,
    object_path     TEXT        NOT NULL,
    md5             VARCHAR(64),
    file_size       BIGINT      NOT NULL DEFAULT 0,
    operator_code   VARCHAR(8),
    update_time     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Spec uses (serial_number, file_name) as the natural key — re-upload of
    -- the same logical file replaces the previous metadata row. Object path
    -- can drift across compressions/encryption suffix changes; tracking it
    -- as data, not key.
    CONSTRAINT backup_restore_file_sn_file_uk UNIQUE (serial_number, file_name)
);

CREATE INDEX IF NOT EXISTS idx_backup_restore_file_sn            ON backup_restore_file(serial_number);
CREATE INDEX IF NOT EXISTS idx_backup_restore_file_operator      ON backup_restore_file(operator_code);
CREATE INDEX IF NOT EXISTS idx_backup_restore_file_update_time   ON backup_restore_file(update_time DESC);

-- =====================================================================
-- 4. backup_schedules — expose is_enable 0/1 alongside boolean enabled
-- =====================================================================
-- Generated column keeps backend code untouched (still writes `enabled`)
-- while spec-style queries can `SELECT is_enable FROM ...`.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'backup_schedules'
          AND column_name = 'is_enable'
    ) THEN
        ALTER TABLE backup_schedules
            ADD COLUMN is_enable SMALLINT
            GENERATED ALWAYS AS (CASE WHEN enabled THEN 1 ELSE 0 END) STORED;
    END IF;
END$$;

-- +goose Down

DROP INDEX IF EXISTS idx_backup_restore_file_update_time;
DROP INDEX IF EXISTS idx_backup_restore_file_operator;
DROP INDEX IF EXISTS idx_backup_restore_file_sn;
DROP TABLE IF EXISTS backup_restore_file;

ALTER TABLE backup_schedules DROP COLUMN IF EXISTS is_enable;

DROP INDEX IF EXISTS idx_restore_tasks_create_user;
DROP INDEX IF EXISTS idx_restore_tasks_operator_code;
ALTER TABLE restore_tasks DROP CONSTRAINT IF EXISTS restore_tasks_task_seq_key;
ALTER TABLE restore_tasks
    DROP COLUMN IF EXISTS create_user,
    DROP COLUMN IF EXISTS operator_code,
    DROP COLUMN IF EXISTS task_result,
    DROP COLUMN IF EXISTS task_name,
    DROP COLUMN IF EXISTS task_seq;

DROP INDEX IF EXISTS idx_backup_tasks_create_user;
DROP INDEX IF EXISTS idx_backup_tasks_operator_code;
ALTER TABLE backup_tasks DROP CONSTRAINT IF EXISTS backup_tasks_task_seq_key;
ALTER TABLE backup_tasks
    DROP COLUMN IF EXISTS create_user,
    DROP COLUMN IF EXISTS operator_code,
    DROP COLUMN IF EXISTS task_result,
    DROP COLUMN IF EXISTS task_name,
    DROP COLUMN IF EXISTS task_seq;

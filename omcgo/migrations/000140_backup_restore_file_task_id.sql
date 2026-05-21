-- +goose Up

-- ═══════════════════════════════════════════════════════════════════════════
-- 自愈段：幂等回放 000130 的 Up 全部内容
-- ═══════════════════════════════════════════════════════════════════════════
-- 历史问题：早期 c834c9e4 新建过一个 000130_param_mappings_mirror.sql，部分
-- 实例的 goose_db_version 已把"版本 130"标为 applied。后来上游 4542c7e9 把
-- 000128_backup_restore_alignment.sql 重命名为 000130_backup_restore_alignment.sql
-- 撞了号，修复提交 2fa45d00 又把 param_mappings_mirror 重命名到 000133。
-- 对那些先 apply 过旧 000130 的实例来说，新内容的 000130 (backup_restore_alignment)
-- 永远不会被 goose 重跑 —— backup_restore_file 表从未创建，本迁移当年直接 ALTER
-- 报 `relation "backup_restore_file" does not exist`。
--
-- 修复策略：在做自己的 ALTER 之前，先把 000130 的 Up 段以 IF NOT EXISTS / DO 块
-- 形式重放一遍。干净部署（000130 已正确 applied）下每条都是 no-op；漂移部署下
-- 完成补建。
--
-- 详见 omcgo/CLAUDE.md §5.5.10 迁移规范、2fa45d00 重命名提交。

-- 1. backup_tasks 对齐列
ALTER TABLE backup_tasks
    ADD COLUMN IF NOT EXISTS task_seq      BIGSERIAL,
    ADD COLUMN IF NOT EXISTS task_name     VARCHAR(200),
    ADD COLUMN IF NOT EXISTS task_result   SMALLINT
        CHECK (task_result IS NULL OR task_result IN (1, 2)),
    ADD COLUMN IF NOT EXISTS operator_code VARCHAR(8),
    ADD COLUMN IF NOT EXISTS create_user   VARCHAR(64);

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_backup_tasks_operator_code ON backup_tasks(operator_code);
CREATE INDEX IF NOT EXISTS idx_backup_tasks_create_user   ON backup_tasks(create_user);

-- 2. restore_tasks 对齐列
ALTER TABLE restore_tasks
    ADD COLUMN IF NOT EXISTS task_seq      BIGSERIAL,
    ADD COLUMN IF NOT EXISTS task_name     VARCHAR(200),
    ADD COLUMN IF NOT EXISTS task_result   SMALLINT
        CHECK (task_result IS NULL OR task_result IN (1, 2)),
    ADD COLUMN IF NOT EXISTS operator_code VARCHAR(8),
    ADD COLUMN IF NOT EXISTS create_user   VARCHAR(64);

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_restore_tasks_operator_code ON restore_tasks(operator_code);
CREATE INDEX IF NOT EXISTS idx_restore_tasks_create_user   ON restore_tasks(create_user);

-- 3. backup_restore_file — 规范化备份文件元数据表（本迁移真正依赖它存在）
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
    CONSTRAINT backup_restore_file_sn_file_uk UNIQUE (serial_number, file_name)
);

CREATE INDEX IF NOT EXISTS idx_backup_restore_file_sn          ON backup_restore_file(serial_number);
CREATE INDEX IF NOT EXISTS idx_backup_restore_file_operator    ON backup_restore_file(operator_code);
CREATE INDEX IF NOT EXISTS idx_backup_restore_file_update_time ON backup_restore_file(update_time DESC);

-- 4. backup_schedules.is_enable GENERATED 列
-- +goose StatementBegin
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
-- +goose StatementEnd

-- 5. upgrade_tasks CHECK 约束 (10 = TaskTypeLogCollect)
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_task_type;
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_task_type
    CHECK (task_type = ANY (ARRAY[1, 2, 4, 6, 8, 10]));


-- ═══════════════════════════════════════════════════════════════════════════
-- 本迁移真正要做的事：backup_restore_file 加 task_id 隔离不同任务的同名文件
-- ═══════════════════════════════════════════════════════════════════════════
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
-- 只 revert 本迁移"自己的"那部分；000130 的自愈段不在这里反转 —— 那是 000130
-- 自己 Down 段的责任，rollback 链路一致。
DROP INDEX IF EXISTS idx_backup_restore_file_task_id;

ALTER TABLE backup_restore_file
    DROP CONSTRAINT IF EXISTS backup_restore_file_sn_task_file_uk;

ALTER TABLE backup_restore_file
    ADD CONSTRAINT backup_restore_file_sn_file_uk
        UNIQUE (serial_number, file_name);

ALTER TABLE backup_restore_file
    DROP COLUMN IF EXISTS task_id;

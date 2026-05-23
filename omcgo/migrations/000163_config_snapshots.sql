-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- T-0164 配置快照表：每台设备一行最新备份/导入快照
--   设计：docs/project/config-snapshot-table-plan-20260522.md
--   约束：历史数据不回填（用户确认 2026-05-22）；新表从上线起累积
--   独立 bucket：config-snapshots（与 backup_tasks 的 config_backup 隔离）
--   命名规范：<serial_number>_CFG.{xml,nv}
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS config_snapshots (
    serial_number   VARCHAR(64)  PRIMARY KEY,
    enb_name        VARCHAR(128),
    product_type    VARCHAR(64),
    file_name       TEXT         NOT NULL,
    file_ext        VARCHAR(8)   NOT NULL CHECK (file_ext IN ('xml', 'nv')),
    object_bucket   VARCHAR(64)  NOT NULL,
    object_path     TEXT         NOT NULL,
    md5             VARCHAR(64),
    file_size       BIGINT       NOT NULL DEFAULT 0,
    source          VARCHAR(16)  NOT NULL CHECK (source IN ('backup', 'manual_upload')),
    source_task_id  UUID,
    update_by       VARCHAR(64),
    update_time     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_config_snapshots_update_time
    ON config_snapshots(update_time DESC);
CREATE INDEX IF NOT EXISTS idx_config_snapshots_product_type
    ON config_snapshots(product_type);
CREATE INDEX IF NOT EXISTS idx_config_snapshots_enb_name
    ON config_snapshots(enb_name);
CREATE INDEX IF NOT EXISTS idx_config_snapshots_source
    ON config_snapshots(source);

-- +goose Down
DROP INDEX IF EXISTS idx_config_snapshots_source;
DROP INDEX IF EXISTS idx_config_snapshots_enb_name;
DROP INDEX IF EXISTS idx_config_snapshots_product_type;
DROP INDEX IF EXISTS idx_config_snapshots_update_time;
DROP TABLE IF EXISTS config_snapshots;

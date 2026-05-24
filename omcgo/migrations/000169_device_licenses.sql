-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- T-0165 设备 License 库：每台设备一行最新 license 文件（手动导入）
--   独立 bucket：device-licenses（与 config_snapshots / config_backup 隔离）
--   命名规范：<serial_number>_LIC.<ext>，ext ∈ {lic, bin, dat}
--   场景：license 升级任务（LICENSE_UPGRADE）按 SN 取最新一份 license，
--        通过 TR-069 Download RPC（FileType="License File"）下发到设备
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS device_licenses (
    serial_number   VARCHAR(64)  PRIMARY KEY,
    enb_name        VARCHAR(128),
    product_type    VARCHAR(64),
    file_name       TEXT         NOT NULL,
    file_ext        VARCHAR(8)   NOT NULL CHECK (file_ext IN ('lic', 'bin', 'dat')),
    object_bucket   VARCHAR(64)  NOT NULL,
    object_path     TEXT         NOT NULL,
    md5             VARCHAR(64),
    file_size       BIGINT       NOT NULL DEFAULT 0,
    -- license 当前只有手动导入一种来源；预留 source 字段方便将来扩展
    source          VARCHAR(16)  NOT NULL DEFAULT 'manual_upload' CHECK (source IN ('manual_upload')),
    description     TEXT,
    update_by       VARCHAR(64),
    update_time     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_licenses_update_time
    ON device_licenses(update_time DESC);
CREATE INDEX IF NOT EXISTS idx_device_licenses_product_type
    ON device_licenses(product_type);
CREATE INDEX IF NOT EXISTS idx_device_licenses_enb_name
    ON device_licenses(enb_name);

-- +goose Down
DROP INDEX IF EXISTS idx_device_licenses_enb_name;
DROP INDEX IF EXISTS idx_device_licenses_product_type;
DROP INDEX IF EXISTS idx_device_licenses_update_time;
DROP TABLE IF EXISTS device_licenses;

-- +goose Up
-- ============================================================
-- 000033_software_remove_carrier.up.sql
-- 移除固件升级模块中不需要的 carrier / operator_code 字段
-- ============================================================

-- firmware_versions: 删除 carrier 列及相关索引
DROP INDEX IF EXISTS idx_firmware_carrier;
DROP INDEX IF EXISTS idx_firmware_carrier_product;
DROP INDEX IF EXISTS idx_firmware_unique_version;
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS carrier;
-- 添加 file_type 列（如果不存在）用于唯一约束
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS file_type VARCHAR(32) NOT NULL DEFAULT '';
-- 重建唯一约束（仅按 product_class + version + file_type）
CREATE UNIQUE INDEX IF NOT EXISTS idx_firmware_unique_version
    ON firmware_versions (COALESCE(product_class, ''), version, file_type);

-- upgrade_tasks: 删除 operator_code 列及相关索引
DROP INDEX IF EXISTS idx_upgrade_tasks_task_operator;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS operator_code;

-- +goose Down
-- ============================================================
-- 回滚：恢复 carrier / operator_code 列
-- ============================================================

-- upgrade_tasks: 恢复 operator_code
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS operator_code VARCHAR(8) NOT NULL DEFAULT 'n/a';
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_operator ON upgrade_tasks (operator_code);

-- firmware_versions: 恢复 carrier，删除 file_type
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS carrier VARCHAR(4) NOT NULL DEFAULT 'n/a';
DROP INDEX IF EXISTS idx_firmware_unique_version;
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS file_type;
CREATE UNIQUE INDEX IF NOT EXISTS idx_firmware_unique_version
    ON firmware_versions (carrier, product_class, version);
CREATE INDEX IF NOT EXISTS idx_firmware_carrier ON firmware_versions (carrier);
CREATE INDEX IF NOT EXISTS idx_firmware_carrier_product ON firmware_versions (carrier, product_class);

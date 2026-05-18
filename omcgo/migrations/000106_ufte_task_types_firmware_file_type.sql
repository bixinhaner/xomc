-- +goose Up
-- ============================================================
-- 000106_ufte_task_types_firmware_file_type.sql
-- UFTE 模板新增软件库文件分类，用于精确关联升级文件库 tab
-- ============================================================

ALTER TABLE ufte_task_types
    ADD COLUMN IF NOT EXISTS firmware_file_type INT;

CREATE INDEX IF NOT EXISTS idx_ufte_task_types_firmware_file_type
    ON ufte_task_types(firmware_file_type);

UPDATE ufte_task_types
SET firmware_file_type = 0
WHERE type_code IN ('ENB_IMG_UPGRADE', 'GNB_IMG_UPGRADE')
  AND firmware_file_type IS NULL;

UPDATE ufte_task_types
SET firmware_file_type = 1
WHERE type_code = 'ENB_PATCH_UPGRADE'
  AND firmware_file_type IS NULL;

UPDATE ufte_task_types
SET firmware_file_type = 6
WHERE type_code = 'ENB_FPGA_UPGRADE'
  AND firmware_file_type IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_ufte_task_types_firmware_file_type;

ALTER TABLE ufte_task_types
    DROP COLUMN IF EXISTS firmware_file_type;
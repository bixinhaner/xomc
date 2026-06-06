-- +goose Up
-- MML 任务结果 CSV 导出（存 MinIO reports/mml-results/ 目录，地址记录到本表）。
--   export_object          —— 全设备汇总 CSV 的 MinIO object key（单文件）
--   device_export_objects  —— 每设备单文件 CSV 的 {device_sn: object_key} 映射
--   export_generated_at    —— 最近一次导出生成时间
ALTER TABLE mml_tasks
    ADD COLUMN IF NOT EXISTS export_object TEXT,
    ADD COLUMN IF NOT EXISTS device_export_objects JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS export_generated_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE mml_tasks
    DROP COLUMN IF EXISTS export_object,
    DROP COLUMN IF EXISTS device_export_objects,
    DROP COLUMN IF EXISTS export_generated_at;

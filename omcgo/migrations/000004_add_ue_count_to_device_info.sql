-- +goose Up
-- UE 数：由 InfoSyncer 从 device_parameters 投影，用于 GIS 地图「UE=0 基站」统计与过滤。
ALTER TABLE device_info ADD COLUMN IF NOT EXISTS ue_count integer DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_device_info_ue_count ON device_info (ue_count);

-- +goose Down
DROP INDEX IF EXISTS idx_device_info_ue_count;
ALTER TABLE device_info DROP COLUMN IF EXISTS ue_count;

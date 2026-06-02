-- +goose Up
-- 告警库新增「描述」字段（用户可编辑的备注，XML 不含此字段）。
-- 注意：Loader 的 ON CONFLICT DO UPDATE 不触及 description，故 XML 重载/导入
-- 不会覆盖用户填写的描述。
ALTER TABLE alarm_definitions ADD COLUMN IF NOT EXISTS description text;

-- +goose Down
ALTER TABLE alarm_definitions DROP COLUMN IF EXISTS description;

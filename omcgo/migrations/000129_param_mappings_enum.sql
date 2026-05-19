-- +goose Up
-- T-0158 (out-of-band of T-0157): 给 param_mappings 加 enum_values + enum_labels 列，
-- 让 dictloader 能从 paramModel XML 的 enumValues / enumLabels 属性解析枚举元数据，
-- 透出到前端校验 + UI 渲染（Select 而非 Input）。
--
-- 字段语义：
--   enum_values:  下发设备的实际值列表（CSV，如 "25,50,75,100"）
--   enum_labels:  UI 展示给用户的标签列表（CSV，与 enum_values 一一对应；
--                 留空时 UI 直接用 enum_values 作 label）
--
-- discovered_param_mappings 同步加列，保证 default + discovered 双源合并时字段对齐。
ALTER TABLE param_mappings           ADD COLUMN IF NOT EXISTS enum_values TEXT;
ALTER TABLE param_mappings           ADD COLUMN IF NOT EXISTS enum_labels TEXT;
ALTER TABLE discovered_param_mappings ADD COLUMN IF NOT EXISTS enum_values TEXT;
ALTER TABLE discovered_param_mappings ADD COLUMN IF NOT EXISTS enum_labels TEXT;

-- +goose Down
ALTER TABLE param_mappings           DROP COLUMN IF EXISTS enum_labels;
ALTER TABLE param_mappings           DROP COLUMN IF EXISTS enum_values;
ALTER TABLE discovered_param_mappings DROP COLUMN IF EXISTS enum_labels;
ALTER TABLE discovered_param_mappings DROP COLUMN IF EXISTS enum_values;

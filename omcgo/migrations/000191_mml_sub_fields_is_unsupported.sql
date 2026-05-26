-- +goose Up
-- ============================================================
-- 000191_mml_sub_fields_is_supported.sql
-- T-0174 Phase 1：MML 命令 sub_field 增加 is_supported 列,
-- 表示该 path 是否被 CPE 支持(默认 true 全部支持)。
--
-- 文件名保留 *_is_unsupported.sql(版本号被占用历史命名),实际列名
-- 与语义都是 `is_supported`(positive boolean,与全仓约定一致)。
--
-- 背景:catalog/字典登记的 path 与 CPE 实际能力可能不一致。BLQ 设备
-- 上的 LST DEVICE_INFO 17 path,有 5 个 CPE 实际不实现导致 GPV
-- atomic fault 9005。删除 sub_field 会失去字典完整性;改用列标记,
-- MML executor 在构造 GPV 时只取 is_supported=true 的 path,字典
-- 仍完整可见(管理 UI 可显示"该 path 被 BLQ CPE 不支持")。
--
-- 默认 true:所有 path 默认视为 supported。运行时测试 / 管理员
-- 手工 / 后续 ACS 自动学习把不支持的 path 标为 false。
--
-- 注:本列在 mml_command_sub_fields 上,语义为"全局是否支持"。
-- 不同 paramModel/product_class 的能力差异**不**在此表达,见
-- backlog T-0175 — 设备级能力学习应迁移到 discovered_param_mappings
-- (product_id, sw_version) 设备级表,本列仅作为单 paramModel 部署
-- 的简化方案 + 全局降级开关。
-- ============================================================

ALTER TABLE mml_command_sub_fields
    ADD COLUMN IF NOT EXISTS is_supported BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN mml_command_sub_fields.is_supported IS
    'CPE 是否支持该 path。true (默认) = supported;false = 测试验证不支持(GPV fault 9005)。'
    'MML executor 构造 GPV 时按此列过滤,catalog loader 的 ON CONFLICT UPDATE 不动此列,保留学习状态。';

-- 部分索引:加速 executor 的 "supported only" 查询
CREATE INDEX IF NOT EXISTS idx_mml_sub_fields_supported
    ON mml_command_sub_fields (command_id, sort_order)
    WHERE is_supported = true;

-- +goose Down
DROP INDEX IF EXISTS idx_mml_sub_fields_supported;
ALTER TABLE mml_command_sub_fields DROP COLUMN IF EXISTS is_supported;

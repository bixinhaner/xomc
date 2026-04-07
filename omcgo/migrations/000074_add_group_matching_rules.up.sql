-- ============================================================
-- 000074_add_group_matching_rules.up.sql
-- 设备分组匹配规则：支持设备名称、LAC、TAC 匹配
-- ============================================================

-- 1. 添加匹配规则字段到 device_groups 表（仅 L2 分组使用）
ALTER TABLE device_groups ADD COLUMN matching_mode VARCHAR(16);
ALTER TABLE device_groups ADD COLUMN name_rule_list JSONB;
ALTER TABLE device_groups ADD COLUMN lac_list INTEGER[];
ALTER TABLE device_groups ADD COLUMN tac_list INTEGER[];

-- 2. 添加约束：matching_mode 只能是有效值
ALTER TABLE device_groups ADD CONSTRAINT chk_dg_matching_mode
    CHECK (matching_mode IS NULL OR matching_mode IN ('deviceName', 'lac', 'tac'));

-- 3. 注释
COMMENT ON COLUMN device_groups.matching_mode IS '匹配模式: deviceName(设备名称), lac(位置区码), tac(跟踪区码)';
COMMENT ON COLUMN device_groups.name_rule_list IS '设备名称匹配规则列表，JSONB格式: [{"condition":"contain","value":"BJ-","andOr":"and"}]';
COMMENT ON COLUMN device_groups.lac_list IS 'LAC位置区码列表，整数数组';
COMMENT ON COLUMN device_groups.tac_list IS 'TAC跟踪区码列表，整数数组';

-- 4. 索引：用于快速查询有匹配规则的分组
CREATE INDEX idx_dg_matching_mode ON device_groups (matching_mode) WHERE matching_mode IS NOT NULL;

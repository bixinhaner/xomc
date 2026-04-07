-- ============================================================
-- 000074_add_group_matching_rules.down.sql
-- 回滚设备分组匹配规则
-- ============================================================

-- 1. 删除索引
DROP INDEX IF EXISTS idx_dg_matching_mode;

-- 2. 删除约束
ALTER TABLE device_groups DROP CONSTRAINT IF EXISTS chk_dg_matching_mode;

-- 3. 删除字段
ALTER TABLE device_groups DROP COLUMN IF EXISTS tac_list;
ALTER TABLE device_groups DROP COLUMN IF EXISTS lac_list;
ALTER TABLE device_groups DROP COLUMN IF EXISTS name_rule_list;
ALTER TABLE device_groups DROP COLUMN IF EXISTS matching_mode;

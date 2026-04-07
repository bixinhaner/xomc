-- 回滚设备规则表

-- 先删除 device_groups 表的外键引用
ALTER TABLE device_groups DROP COLUMN IF EXISTS bound_rule_id;

-- 删除触发器
DROP TRIGGER IF EXISTS trigger_device_rules_updated_at ON device_rules;

-- 删除索引
DROP INDEX IF EXISTS idx_device_rules_priority;
DROP INDEX IF EXISTS idx_device_rules_enabled;
DROP INDEX IF EXISTS idx_device_rules_target_group;
DROP INDEX IF EXISTS idx_device_rules_priority_unique;

-- 删除表
DROP TABLE IF EXISTS device_rules;

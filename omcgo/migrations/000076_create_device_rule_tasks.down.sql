-- 回滚设备规则任务表
DROP INDEX IF EXISTS idx_device_rule_tasks_created;
DROP INDEX IF EXISTS idx_device_rule_tasks_status;
DROP INDEX IF EXISTS idx_device_rule_tasks_rule;
DROP TABLE IF EXISTS device_rule_tasks;

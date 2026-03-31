-- 000065_enhance_device_groups.down.sql

-- 移除默认组成员关系
DELETE FROM device_group_members
WHERE group_id IN (
    '00000000-0000-0000-0000-000000000001'::uuid,
    '00000000-0000-0000-0000-000000000002'::uuid
);

-- 移除默认组
DELETE FROM device_groups WHERE is_default = TRUE;

-- 移除约束和索引
ALTER TABLE device_group_members DROP CONSTRAINT IF EXISTS uq_dgm_device;
DROP INDEX IF EXISTS idx_dg_name_parent;
ALTER TABLE device_groups DROP CONSTRAINT IF EXISTS chk_dg_level_parent;
ALTER TABLE device_groups DROP CONSTRAINT IF EXISTS chk_dg_level;

-- 移除字段
ALTER TABLE device_groups DROP COLUMN IF EXISTS updated_by;
ALTER TABLE device_groups DROP COLUMN IF EXISTS created_by;
ALTER TABLE device_groups DROP COLUMN IF EXISTS level;
ALTER TABLE device_groups DROP COLUMN IF EXISTS is_default;
ALTER TABLE device_groups DROP COLUMN IF EXISTS remark;
ALTER TABLE device_groups DROP COLUMN IF EXISTS status;

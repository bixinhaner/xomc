-- ============================================================
-- 000065_enhance_device_groups.up.sql
-- 设备组管理增强：两级树约束 + 默认组 + 审计字段
-- ============================================================

-- 1. 追加字段
ALTER TABLE device_groups ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active';
ALTER TABLE device_groups ADD COLUMN remark TEXT;
ALTER TABLE device_groups ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE device_groups ADD COLUMN level SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE device_groups ADD COLUMN created_by VARCHAR(64);
ALTER TABLE device_groups ADD COLUMN updated_by VARCHAR(64);

-- 2. 回填现有数据：根据 parent_id 推导 level
UPDATE device_groups SET level = 1 WHERE parent_id IS NULL;
UPDATE device_groups SET level = 2 WHERE parent_id IS NOT NULL;

-- 3. 层级约束：只允许 1 和 2
ALTER TABLE device_groups
    ADD CONSTRAINT chk_dg_level CHECK (level IN (1, 2));

-- 4. 层级与父子关系一致性：一级无父，二级必有父
ALTER TABLE device_groups
    ADD CONSTRAINT chk_dg_level_parent
    CHECK ((level = 1 AND parent_id IS NULL) OR (level = 2 AND parent_id IS NOT NULL));

-- 5. 同一父节点下名称唯一（一级组用 COALESCE 处理 NULL parent_id）
CREATE UNIQUE INDEX idx_dg_name_parent
    ON device_groups (name, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- 6. 设备只能归属一个组（新增唯一约束）
ALTER TABLE device_group_members
    ADD CONSTRAINT uq_dgm_device UNIQUE (device_id);

-- 7. 种子数据：默认设备组（固定 UUID，代码常量引用）
INSERT INTO device_groups (id, name, parent_id, level, is_default, status, remark, created_by)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '默认设备组',
    NULL,
    1,
    TRUE,
    'active',
    '系统默认一级设备组，不可修改删除',
    'system'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO device_groups (id, name, parent_id, level, is_default, status, remark, created_by)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '未分组设备',
    '00000000-0000-0000-0000-000000000001',
    2,
    TRUE,
    'active',
    '系统默认二级设备组，删除组后设备自动归入此组',
    'system'
) ON CONFLICT (id) DO NOTHING;

-- 8. 将现有未归组的设备自动加入默认二级组
INSERT INTO device_group_members (group_id, device_id, added_at)
SELECT '00000000-0000-0000-0000-000000000002'::uuid, d.id, NOW()
FROM devices d
WHERE NOT EXISTS (
    SELECT 1 FROM device_group_members dgm WHERE dgm.device_id = d.id
)
ON CONFLICT (device_id) DO NOTHING;

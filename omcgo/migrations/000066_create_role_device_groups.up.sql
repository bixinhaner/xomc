-- ============================================================
-- 000066_create_role_device_groups.up.sql
-- 角色-设备组数据权限关联表
-- ============================================================

CREATE TABLE role_device_groups (
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    group_id   UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, group_id)
);

CREATE INDEX idx_rdg_role ON role_device_groups (role_id);
CREATE INDEX idx_rdg_group ON role_device_groups (group_id);

COMMENT ON TABLE role_device_groups IS
    '角色数据权限：角色可见的设备组。关联一级组=可见其下所有二级组的设备';

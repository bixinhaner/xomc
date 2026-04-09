-- ============================================================
-- 000083_add_user_roles_default.up.sql
-- 用户角色表增加 is_default 列，支持多角色绑定与默认角色
-- ============================================================

ALTER TABLE user_roles ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT FALSE;

-- 每个用户只能有一个默认角色（唯一部分索引）
CREATE UNIQUE INDEX uniq_user_default_role ON user_roles(user_id) WHERE is_default = TRUE;

COMMENT ON COLUMN user_roles.is_default IS '是否为默认角色：登录后默认激活的角色';

-- 迁移现有数据：每个用户的第一个角色设为默认角色
UPDATE user_roles ur
SET is_default = TRUE
WHERE id = (
    SELECT MIN(id) FROM user_roles ur2 WHERE ur2.user_id = ur.user_id
);

-- ============================================================
-- 000082_add_role_inheritance.up.sql
-- 角色继承表：支持 Casbin RBAC 角色继承链
-- ============================================================

CREATE TABLE role_inheritance (
    parent_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    child_role_id  UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    domain         VARCHAR(16) NOT NULL DEFAULT 'system',
    PRIMARY KEY (parent_role_id, child_role_id, domain)
);

COMMENT ON TABLE role_inheritance IS '角色继承：admin 继承 operator，operator 继承 viewer';

-- 种子数据：admin 继承 operator，operator 继承 viewer
INSERT INTO role_inheritance (parent_role_id, child_role_id, domain)
SELECT p.id, c.id, 'system'
FROM roles p, roles c
WHERE p.name = 'admin' AND c.name = 'operator';

INSERT INTO role_inheritance (parent_role_id, child_role_id, domain)
SELECT p.id, c.id, 'system'
FROM roles p, roles c
WHERE p.name = 'operator' AND c.name = 'viewer';

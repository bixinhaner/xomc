-- ============================================================
-- 菜单管理表设计
-- 包含创建人、创建时间、更新人、更新时间等审计字段
-- 不使用触发器，由应用层维护
-- ============================================================

-- ============================================================
-- 1. 菜单表 (menus)
-- ============================================================
CREATE TABLE menus (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(64) NOT NULL,           -- 菜单名称
    type            VARCHAR(16) NOT NULL,           -- 类型: directory(目录)/menu(菜单)/button(按钮)
    permission_key  VARCHAR(128) NOT NULL,          -- 权限标识，如 device:list:query
    parent_id       UUID REFERENCES menus(id) ON DELETE CASCADE,  -- 父菜单ID
    sort_order      INT NOT NULL DEFAULT 0,         -- 排序号

    -- 路由相关
    route_path      VARCHAR(256),                   -- 前端路由路径，如 /device/list
    route_params    TEXT,                           -- 路由参数 (JSON)

    -- 显示相关
    icon            VARCHAR(64),                    -- 图标名称
    show_status     VARCHAR(16) NOT NULL DEFAULT 'show',  -- 显示状态: show/hide
    is_external     VARCHAR(8) NOT NULL DEFAULT 'no',     -- 是否外链: yes/no
    component_path VARCHAR(256),                   -- 组件路径

    -- API 权限
    api_permission  VARCHAR(16) NOT NULL DEFAULT 'none',   -- API权限: required/none

    -- 状态
    status          VARCHAR(16) NOT NULL DEFAULT 'normal',  -- 状态: normal/disabled

    -- 审计字段（不使用触发器，应用层维护）
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_menu_type CHECK (type IN ('directory', 'menu', 'button')),
    CONSTRAINT chk_show_status CHECK (show_status IN ('show', 'hide')),
    CONSTRAINT chk_is_external CHECK (is_external IN ('yes', 'no')),
    CONSTRAINT chk_api_permission CHECK (api_permission IN ('required', 'none')),
    CONSTRAINT chk_status CHECK (status IN ('normal', 'disabled'))
);

-- 索引
CREATE INDEX idx_menus_parent ON menus(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_menus_type ON menus(type);
CREATE INDEX idx_menus_permission_key ON menus(permission_key);
CREATE INDEX idx_menus_status ON menus(status);
CREATE INDEX idx_menus_sort ON menus(parent_id NULLS FIRST, sort_order);

-- 唯一约束：同级菜单下名称唯一
CREATE UNIQUE INDEX uniq_menu_name_per_parent ON menus(parent_id NULLS FIRST, name)
    WHERE status = 'normal';

-- 注释
COMMENT ON TABLE menus IS '菜单表：支持三级菜单结构（目录-菜单-按钮）';
COMMENT ON COLUMN menus.type IS '菜单类型: directory=一级目录, menu=二级菜单, button=三级按钮';
COMMENT ON COLUMN menus.permission_key IS '权限标识，格式: {module}:{page}:{action}，如 device:list:query';
COMMENT ON COLUMN menus.show_status IS '显示状态: show=显示, hide=隐藏';
COMMENT ON COLUMN menus.api_permission IS 'API权限: required=需要接口鉴权, none=无需鉴权';

-- ============================================================
-- 2. 角色-菜单关联表 (role_menus)
-- ============================================================
CREATE TABLE role_menus (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    menu_id         UUID NOT NULL REFERENCES menus(id) ON DELETE CASCADE,

    -- 审计字段
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uniq_role_menu UNIQUE(role_id, menu_id)
);

-- 索引
CREATE INDEX idx_role_menus_role ON role_menus(role_id);
CREATE INDEX idx_role_menus_menu ON role_menus(menu_id);

COMMENT ON TABLE role_menus IS '角色-菜单关联表：定义角色可访问的菜单';

-- ============================================================
-- 3. 菜单操作权限模板 (menu_operation_templates)
--    用于快速生成按钮级菜单的标准操作
-- ============================================================
CREATE TABLE menu_operation_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(32) NOT NULL UNIQUE,     -- 操作代码，如 query, add, edit
    name            VARCHAR(32) NOT NULL,            -- 操作名称，如 查询, 新增, 修改
    sort_order      INT NOT NULL DEFAULT 0,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,

    -- 审计字段
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE menu_operation_templates IS '菜单操作模板：定义标准操作按钮（查询、新增、修改等）';

-- 插入默认操作模板
INSERT INTO menu_operation_templates (code, name, sort_order) VALUES
('query', '查询', 1),
('add', '添加', 2),
('edit', '修改', 3),
('delete', '删除', 4),
('export', '导出', 5),
('import', '导入', 6),
('batch', '批量操作', 7),
('detail', '详情', 8),
('approve', '审核', 9),
('execute', '执行', 10);

-- ============================================================
-- 4. 初始化菜单数据示例
-- ============================================================

-- 一级：设备管理目录
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111101', '设备管理', 'directory', 'device', NULL, 1, 'DeviceOutlined', 'normal', NOW());

-- 二级：设备管理下的菜单
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111102', '设备列表', 'menu', 'device:list', '11111111-1111-1111-1111-111111111101', 1, '/device/list', 'UnorderedListOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111103', '设备分组', 'menu', 'device:group', '11111111-1111-1111-1111-111111111101', 2, '/device/group', 'ApartmentOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111104', '设备注册', 'menu', 'device:register', '11111111-1111-1111-1111-111111111101', 3, '/device/register', 'PlusOutlined', 'normal', NOW());

-- 三级：设备列表的操作按钮
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status, created_at) VALUES
('11111111-1111-1111-1111-111111111201', '查询', 'button', 'device:list:query', '11111111-1111-1111-1111-111111111102', 1, 'normal', NOW()),
('11111111-1111-1111-1111-111111111202', '添加', 'button', 'device:list:add', '11111111-1111-1111-1111-111111111102', 2, 'normal', NOW()),
('11111111-1111-1111-1111-111111111203', '修改', 'button', 'device:list:edit', '11111111-1111-1111-1111-111111111102', 3, 'normal', NOW()),
('11111111-1111-1111-1111-111111111204', '删除', 'button', 'device:list:delete', '11111111-1111-1111-1111-111111111102', 4, 'normal', NOW()),
('11111111-1111-1111-1111-111111111205', '导出', 'button', 'device:list:export', '11111111-1111-1111-1111-111111111102', 5, 'normal', NOW()),
('11111111-1111-1111-1111-111111111206', '批量操作', 'button', 'device:list:batch', '11111111-1111-1111-1111-111111111102', 6, 'normal', NOW());

-- 一级：告警管理目录
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111105', '告警管理', 'directory', 'alarm', NULL, 2, 'AlertOutlined', 'normal', NOW());

-- 二级：告警管理下的菜单
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111106', '当前告警', 'menu', 'alarm:current', '11111111-1111-1111-1111-111111111105', 1, '/alarm/current', 'BellOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111107', '历史告警', 'menu', 'alarm:history', '11111111-1111-1111-1111-111111111105', 2, '/alarm/history', 'HistoryOutlined', 'normal', NOW());

-- 一级：系统管理目录
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111108', '系统管理', 'directory', 'system', NULL, 99, 'SettingOutlined', 'normal', NOW());

-- 二级：系统管理下的菜单
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111109', '用户管理', 'menu', 'system:user', '11111111-1111-1111-1111-111111111108', 1, '/system/user', 'UserOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111110', '角色管理', 'menu', 'system:role', '11111111-1111-1111-1111-111111111108', 2, '/system/role', 'TeamOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111111', '菜单管理', 'menu', 'system:menu', '11111111-1111-1111-1111-111111111108', 3, '/system/menu', 'MenuOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111112', '操作日志', 'menu', 'system:log', '11111111-1111-1111-1111-111111111108', 4, '/system/log', 'FileTextOutlined', 'normal', NOW());

-- ============================================================
-- 5. 初始化管理员角色菜单关联（示例：admin 拥有所有菜单权限）
-- ============================================================

-- 假设 admin 用户的角色 ID 为 '22222222-2222-2222-2222-222222222201'
-- 这里需要根据实际 roles 表的 admin 角色 ID 进行调整

-- 取消注释并替换实际的角色 ID 后执行
/*
INSERT INTO role_menus (role_id, menu_id, created_at)
SELECT
    (SELECT id FROM roles WHERE name = 'admin' LIMIT 1),
    m.id,
    NOW()
FROM menus m;
*/

-- ============================================================
-- 6. 有用的查询视图
-- ============================================================

-- 菜单树视图（带层级信息）
CREATE OR REPLACE VIEW v_menu_tree AS
WITH RECURSIVE menu_hierarchy AS (
    -- 根节点（一级目录）
    SELECT
        id, name, type, permission_key, parent_id,
        sort_order, route_path, icon, show_status,
        status, created_by, created_at, updated_by, updated_at,
        ARRAY[id] AS path,
        1 AS level,
        name AS full_name
    FROM menus
    WHERE parent_id IS NULL

    UNION ALL

    -- 子节点
    SELECT
        m.id, m.name, m.type, m.permission_key, m.parent_id,
        m.sort_order, m.route_path, m.icon, m.show_status,
        m.status, m.created_by, m.created_at, m.updated_by, m.updated_at,
        mh.path || m.id,
        mh.level + 1,
        mh.full_name || ' > ' || m.name
    FROM menus m
    INNER JOIN menu_hierarchy mh ON m.parent_id = mh.id
)
SELECT * FROM menu_hierarchy ORDER BY path, sort_order;

COMMENT ON VIEW v_menu_tree IS '菜单树视图：包含完整的层级路径和层级信息';

-- 角色菜单视图
CREATE OR REPLACE VIEW v_role_menus AS
SELECT
    r.id AS role_id,
    r.name AS role_name,
    m.id AS menu_id,
    m.name AS menu_name,
    m.type AS menu_type,
    m.permission_key,
    m.route_path,
    m.parent_id,
    mh.full_name AS menu_path,
    m.sort_order
FROM roles r
INNER JOIN role_menus rm ON r.id = rm.role_id
INNER JOIN v_menu_tree mh ON mh.id = rm.menu_id
ORDER BY mh.path, mh.sort_order;

COMMENT ON VIEW v_menu_tree IS '角色菜单视图：角色与菜单的关联关系';

-- ============================================================
-- 7. 辅助函数（替代触发器）
-- ============================================================

-- 更新 updated_at 字段的函数（应用层调用）
CREATE OR REPLACE FUNCTION update_menu_timestamp(
    p_menu_id UUID,
    p_updated_by UUID
) RETURNS UUID AS $$
DECLARE
    v_menu_id UUID;
BEGIN
    UPDATE menus
    SET updated_at = NOW(),
        updated_by = p_updated_by
    WHERE id = p_menu_id
    RETURNING id INTO v_menu_id;

    RETURN v_menu_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION update_menu_timestamp IS '更新菜单时间戳（应用层调用，不使用触发器）';

-- 获取角色的所有菜单（含权限标识）
CREATE OR REPLACE FUNCTION get_role_menus(p_role_id UUID)
RETURNS TABLE (
    menu_id UUID,
    menu_name VARCHAR,
    menu_type VARCHAR,
    permission_key VARCHAR,
    route_path VARCHAR,
    parent_id UUID,
    level INT,
    full_name VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        mh.id,
        mh.name,
        mh.type,
        mh.permission_key,
        mh.route_path,
        mh.parent_id,
        mh.level,
        mh.full_name
    FROM v_menu_tree mh
    INNER JOIN role_menus rm ON mh.id = rm.menu_id
    WHERE rm.role_id = p_role_id
        AND mh.status = 'normal'
    ORDER BY mh.path, mh.sort_order;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_role_menus IS '获取角色的所有菜单（含权限标识）';

-- ============================================================
-- 8. 应用层使用示例
-- ============================================================

/*
-- 创建菜单（应用层代码示例）
INSERT INTO menus (
    name, type, permission_key, parent_id, sort_order,
    route_path, icon, status,
    created_by, created_at, updated_by, updated_at
) VALUES (
    '新菜单', 'menu', 'new:menu', <parent_uuid>, 1,
    '/new/menu', 'IconOutlined', 'normal',
    <user_uuid>, NOW(), <user_uuid>, NOW()
);

-- 更新菜单（应用层代码示例）
UPDATE menus
SET
    name = '更新后的菜单名',
    route_path = '/updated/path',
    updated_by = <user_uuid>,
    updated_at = NOW()
WHERE id = <menu_uuid>;

-- 或使用辅助函数
SELECT update_menu_timestamp(<menu_uuid>, <user_uuid>);

-- 为角色分配菜单
INSERT INTO role_menus (role_id, menu_id, created_by, created_at)
VALUES (<role_uuid>, <menu_uuid>, <user_uuid>, NOW());

-- 查询用户的菜单树（通过角色）
SELECT DISTINCT mh.*
FROM v_menu_tree mh
INNER JOIN role_menus rm ON mh.id = rm.menu_id
INNER JOIN user_roles ur ON rm.role_id = ur.role_id
WHERE ur.user_id = <user_uuid>
    AND mh.status = 'normal'
ORDER BY mh.path, mh.sort_order;
*/

-- ============================================================
-- 000072_create_menu_tables.up.sql
-- 菜单管理表：通过菜单统一管理权限
-- 删除了冗余的 permissions 和 menu_operation_templates 表
-- ============================================================

-- ============================================================
-- 1. 菜单表 (menus) - 菜单即权限
-- ============================================================
CREATE TABLE menus (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(64) NOT NULL,
    type            VARCHAR(16) NOT NULL,           -- directory/menu/button
    permission_key  VARCHAR(128) NOT NULL,          -- 权限标识：device:list:query
    parent_id       UUID REFERENCES menus(id) ON DELETE CASCADE,
    sort_order      INT NOT NULL DEFAULT 0,

    -- 路由相关
    route_path      VARCHAR(256),
    component_path VARCHAR(256),

    -- 显示相关
    icon            VARCHAR(64),
    show_status     VARCHAR(16) NOT NULL DEFAULT 'show',  -- show/hide

    -- 状态
    status          VARCHAR(16) NOT NULL DEFAULT 'normal',  -- normal/disabled

    -- 审计字段（应用层维护，不使用触发器）
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_menu_type CHECK (type IN ('directory', 'menu', 'button')),
    CONSTRAINT chk_show_status CHECK (show_status IN ('show', 'hide')),
    CONSTRAINT chk_status CHECK (status IN ('normal', 'disabled'))
);

-- 索引
CREATE INDEX idx_menus_parent ON menus(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_menus_type ON menus(type);
CREATE INDEX idx_menus_permission_key ON menus(permission_key);
CREATE INDEX idx_menus_status ON menus(status);
CREATE INDEX idx_menus_sort ON menus(parent_id NULLS FIRST, sort_order);
CREATE UNIQUE INDEX uniq_menu_name_per_parent ON menus(parent_id NULLS FIRST, name) WHERE status = 'normal';

COMMENT ON TABLE menus IS '菜单表：通过菜单统一管理权限，菜单即权限';
COMMENT ON COLUMN menus.type IS '菜单类型: directory=一级目录, menu=二级菜单, button=三级按钮';
COMMENT ON COLUMN menus.permission_key IS '权限标识，格式: {module}:{page}[:{action}]，如 device:list:query';

-- ============================================================
-- 2. 角色-菜单关联表 (role_menus)
-- ============================================================
CREATE TABLE role_menus (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    menu_id     UUID NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uniq_role_menu UNIQUE(role_id, menu_id)
);

CREATE INDEX idx_role_menus_role ON role_menus(role_id);
CREATE INDEX idx_role_menus_menu ON role_menus(menu_id);

COMMENT ON TABLE role_menus IS '角色-菜单关联：角色拥有哪些菜单，就拥有哪些权限';

-- ============================================================
-- 3. 初始化菜单数据
-- ============================================================

-- 一级：设备管理
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

-- 一级：告警管理
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111105', '告警管理', 'directory', 'alarm', NULL, 2, 'AlertOutlined', 'normal', NOW());

-- 二级：告警管理下的菜单
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111106', '当前告警', 'menu', 'alarm:current', '11111111-1111-1111-1111-111111111105', 1, '/alarm/current', 'BellOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111107', '历史告警', 'menu', 'alarm:history', '11111111-1111-1111-1111-111111111105', 2, '/alarm/history', 'HistoryOutlined', 'normal', NOW());

-- 一级：系统管理
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111108', '系统管理', 'directory', 'system', NULL, 99, 'SettingOutlined', 'normal', NOW());

-- 二级：系统管理下的菜单
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111109', '用户管理', 'menu', 'system:user', '11111111-1111-1111-1111-111111111108', 1, '/system/user', 'UserOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111110', '角色管理', 'menu', 'system:role', '11111111-1111-1111-1111-111111111108', 2, '/system/role', 'TeamOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111111', '菜单管理', 'menu', 'system:menu', '11111111-1111-1111-1111-111111111108', 3, '/system/menu', 'MenuOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111112', '操作日志', 'menu', 'system:log', '11111111-1111-1111-1111-111111111108', 4, '/system/log', 'FileTextOutlined', 'normal', NOW());

-- 系统管理菜单的操作按钮
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status, created_at) VALUES
('11111111-1111-1111-1111-111111211101', '查询', 'button', 'system:user:query', '11111111-1111-1111-1111-111111111109', 1, 'normal', NOW()),
('11111111-1111-1111-1111-111111211102', '添加', 'button', 'system:user:add', '11111111-1111-1111-1111-111111111109', 2, 'normal', NOW()),
('11111111-1111-1111-1111-111111211103', '修改', 'button', 'system:user:edit', '11111111-1111-1111-1111-111111111109', 3, 'normal', NOW()),
('11111111-1111-1111-1111-111111211104', '删除', 'button', 'system:user:delete', '11111111-1111-1111-1111-111111111109', 4, 'normal', NOW()),
('11111111-1111-1111-1111-111111211105', '重置密码', 'button', 'system:user:resetpwd', '11111111-1111-1111-1111-111111111109', 5, 'normal', NOW());

-- +goose Up
-- ============================================================
-- 000009_sys_admin.up.sql
-- Menus, role_menus, sys_dictionaries, sys_dictionary_details,
-- sys_configs, sys_login_logs, sys_oper_logs, sys_task_logs
-- ============================================================

-- ============================================================
-- 1. menus — menu tree (menu = permission)
-- ============================================================
CREATE TABLE menus (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(64) NOT NULL,
    type            VARCHAR(16) NOT NULL,           -- directory/menu/button
    permission_key  VARCHAR(128) NOT NULL,          -- e.g. device:list:query
    parent_id       UUID REFERENCES menus(id) ON DELETE CASCADE,
    sort_order      INT NOT NULL DEFAULT 0,

    -- Route
    route_path      VARCHAR(256),
    component_path VARCHAR(256),

    -- Display
    icon            VARCHAR(64),
    show_status     VARCHAR(16) NOT NULL DEFAULT 'show',  -- show/hide

    -- Status
    status          VARCHAR(16) NOT NULL DEFAULT 'normal',  -- normal/disabled

    -- Audit fields
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_menu_type CHECK (type IN ('directory', 'menu', 'button')),
    CONSTRAINT chk_show_status CHECK (show_status IN ('show', 'hide')),
    CONSTRAINT chk_status CHECK (status IN ('normal', 'disabled'))
);

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
-- 2. role_menus — role-menu association
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
-- 3. sys_dictionaries — dictionary main table
-- ============================================================
CREATE TABLE sys_dictionaries (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(255) NOT NULL,
    status      BOOLEAN NOT NULL DEFAULT TRUE,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX uniq_dict_type_active ON sys_dictionaries(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_sys_dict_deleted_at ON sys_dictionaries(deleted_at) WHERE deleted_at IS NOT NULL;

COMMENT ON TABLE sys_dictionaries IS '字典主表：统一管理枚举值分类';
COMMENT ON COLUMN sys_dictionaries.type IS '字典类型（英文标识），全局唯一，如 gender、status';

-- ============================================================
-- 4. sys_dictionary_details — dictionary detail entries
-- ============================================================
CREATE TABLE sys_dictionary_details (
    id                 BIGSERIAL PRIMARY KEY,
    label              VARCHAR(255) NOT NULL,
    value              VARCHAR(255) NOT NULL,
    extend             VARCHAR(255) NOT NULL DEFAULT '',
    status             BOOLEAN NOT NULL DEFAULT TRUE,
    sort               INT NOT NULL DEFAULT 0,
    sys_dictionary_id  BIGINT NOT NULL REFERENCES sys_dictionaries(id) ON DELETE CASCADE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ
);

CREATE INDEX idx_sys_dict_detail_dict_id ON sys_dictionary_details(sys_dictionary_id);
CREATE INDEX idx_sys_dict_detail_deleted_at ON sys_dictionary_details(deleted_at) WHERE deleted_at IS NOT NULL;

COMMENT ON TABLE sys_dictionary_details IS '字典详情表：存储字典的具体选项值';

-- ============================================================
-- 5. sys_configs — system configuration parameters
-- ============================================================
CREATE TABLE sys_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category    VARCHAR(64) NOT NULL,
    key         VARCHAR(128) NOT NULL,
    value       TEXT NOT NULL DEFAULT '',
    value_type  VARCHAR(16) NOT NULL DEFAULT 'string',
    description VARCHAR(255) NOT NULL DEFAULT '',
    is_public   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uniq_config_key UNIQUE(category, key),
    CONSTRAINT chk_value_type CHECK (value_type IN ('string', 'int', 'float', 'bool', 'json'))
);

CREATE INDEX idx_sys_configs_category ON sys_configs(category);
CREATE INDEX idx_sys_configs_public ON sys_configs(is_public) WHERE is_public = TRUE;

COMMENT ON TABLE sys_configs IS '系统配置参数表';

-- ============================================================
-- 6. sys_login_logs — login/logout event logs
-- ============================================================
CREATE TABLE sys_login_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL,
    ip_address  VARCHAR(64) NOT NULL DEFAULT '',
    location    VARCHAR(128) NOT NULL DEFAULT '',
    browser     VARCHAR(128) NOT NULL DEFAULT '',
    os          VARCHAR(128) NOT NULL DEFAULT '',
    status      BOOLEAN NOT NULL DEFAULT TRUE,
    message     VARCHAR(255) NOT NULL DEFAULT '',
    login_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_logs_user_id ON sys_login_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_login_logs_login_at ON sys_login_logs(login_at);

COMMENT ON TABLE sys_login_logs IS '登录日志：记录用户登录/登出事件';

-- ============================================================
-- 7. sys_oper_logs — operation audit logs
-- ============================================================
CREATE TABLE sys_oper_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL DEFAULT '',
    action      VARCHAR(64) NOT NULL,
    module      VARCHAR(64) NOT NULL DEFAULT '',
    target      VARCHAR(255) NOT NULL DEFAULT '',
    detail      TEXT NOT NULL DEFAULT '',
    ip_address  VARCHAR(64) NOT NULL DEFAULT '',
    user_agent  VARCHAR(512) NOT NULL DEFAULT '',
    status      BOOLEAN NOT NULL DEFAULT TRUE,
    error_msg   VARCHAR(512) NOT NULL DEFAULT '',
    cost_ms     INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oper_logs_user_id ON sys_oper_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_oper_logs_action ON sys_oper_logs(action);
CREATE INDEX idx_oper_logs_created_at ON sys_oper_logs(created_at);
CREATE INDEX idx_oper_logs_module ON sys_oper_logs(module);

COMMENT ON TABLE sys_oper_logs IS '操作日志：记录用户关键操作';

-- ============================================================
-- 8. sys_task_logs — background task execution logs
-- ============================================================
CREATE TABLE sys_task_logs (
    id          BIGSERIAL PRIMARY KEY,
    task_type   VARCHAR(64) NOT NULL,
    task_id     VARCHAR(128) NOT NULL DEFAULT '',
    status      VARCHAR(16) NOT NULL DEFAULT 'running',
    operator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    operator    VARCHAR(64) NOT NULL DEFAULT '',
    target      VARCHAR(255) NOT NULL DEFAULT '',
    detail      TEXT NOT NULL DEFAULT '',
    error_msg   TEXT NOT NULL DEFAULT '',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    cost_ms     INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_task_logs_task_type ON sys_task_logs(task_type);
CREATE INDEX idx_task_logs_operator_id ON sys_task_logs(operator_id) WHERE operator_id IS NOT NULL;
CREATE INDEX idx_task_logs_status ON sys_task_logs(status);
CREATE INDEX idx_task_logs_started_at ON sys_task_logs(started_at);

COMMENT ON TABLE sys_task_logs IS '任务日志：记录后台任务执行结果';

-- ============================================================
-- Seed data: Menus
-- ============================================================

-- Level 1: Device management
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111101', '设备管理', 'directory', 'device', NULL, 1, 'DeviceOutlined', 'normal', NOW());

-- Level 2: Device management menus
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111102', '设备列表', 'menu', 'device:list', '11111111-1111-1111-1111-111111111101', 1, '/device/list', 'UnorderedListOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111103', '设备分组', 'menu', 'device:group', '11111111-1111-1111-1111-111111111101', 2, '/device/group', 'ApartmentOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111104', '设备注册', 'menu', 'device:register', '11111111-1111-1111-1111-111111111101', 3, '/device/register', 'PlusOutlined', 'normal', NOW());

-- Level 3: Device list buttons
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status, created_at) VALUES
('11111111-1111-1111-1111-111111111201', '查询', 'button', 'device:list:query', '11111111-1111-1111-1111-111111111102', 1, 'normal', NOW()),
('11111111-1111-1111-1111-111111111202', '添加', 'button', 'device:list:add', '11111111-1111-1111-1111-111111111102', 2, 'normal', NOW()),
('11111111-1111-1111-1111-111111111203', '修改', 'button', 'device:list:edit', '11111111-1111-1111-1111-111111111102', 3, 'normal', NOW()),
('11111111-1111-1111-1111-111111111204', '删除', 'button', 'device:list:delete', '11111111-1111-1111-1111-111111111102', 4, 'normal', NOW()),
('11111111-1111-1111-1111-111111111205', '导出', 'button', 'device:list:export', '11111111-1111-1111-1111-111111111102', 5, 'normal', NOW()),
('11111111-1111-1111-1111-111111111206', '批量操作', 'button', 'device:list:batch', '11111111-1111-1111-1111-111111111102', 6, 'normal', NOW());

-- Level 1: Alarm management
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111105', '告警管理', 'directory', 'alarm', NULL, 2, 'AlertOutlined', 'normal', NOW());

-- Level 2: Alarm management menus
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111106', '当前告警', 'menu', 'alarm:current', '11111111-1111-1111-1111-111111111105', 1, '/alarm/current', 'BellOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111107', '历史告警', 'menu', 'alarm:history', '11111111-1111-1111-1111-111111111105', 2, '/alarm/history', 'HistoryOutlined', 'normal', NOW());

-- Level 1: System management
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111108', '系统管理', 'directory', 'system', NULL, 99, 'SettingOutlined', 'normal', NOW());

-- Level 2: System management menus
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111109', '用户管理', 'menu', 'system:user', '11111111-1111-1111-1111-111111111108', 1, '/system/user', 'UserOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111110', '角色管理', 'menu', 'system:role', '11111111-1111-1111-1111-111111111108', 2, '/system/role', 'TeamOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111111', '菜单管理', 'menu', 'system:menu', '11111111-1111-1111-1111-111111111108', 3, '/system/menu', 'MenuOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111112', '操作日志', 'menu', 'system:log', '11111111-1111-1111-1111-111111111108', 4, '/system/log', 'FileTextOutlined', 'normal', NOW());

-- Level 3: System management buttons
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status, created_at) VALUES
('11111111-1111-1111-1111-111111211101', '查询', 'button', 'system:user:query', '11111111-1111-1111-1111-111111111109', 1, 'normal', NOW()),
('11111111-1111-1111-1111-111111211102', '添加', 'button', 'system:user:add', '11111111-1111-1111-1111-111111111109', 2, 'normal', NOW()),
('11111111-1111-1111-1111-111111211103', '修改', 'button', 'system:user:edit', '11111111-1111-1111-1111-111111111109', 3, 'normal', NOW()),
('11111111-1111-1111-1111-111111211104', '删除', 'button', 'system:user:delete', '11111111-1111-1111-1111-111111111109', 4, 'normal', NOW()),
('11111111-1111-1111-1111-111111211105', '重置密码', 'button', 'system:user:resetpwd', '11111111-1111-1111-1111-111111111109', 5, 'normal', NOW());

-- ============================================================
-- Seed data: Dictionaries
-- ============================================================
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
('性别', 'gender', TRUE, '用户性别'),
('数据库int类型', 'int', TRUE, '整型映射'),
('时间日期类型', 'time.Time', TRUE, '时间类型映射'),
('浮点型', 'float64', TRUE, '浮点类型映射'),
('字符串', 'string', TRUE, '字符串类型映射'),
('布尔类型', 'bool', TRUE, '布尔类型映射');

-- Gender dictionary items
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('男', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'gender')),
('女', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'gender'));

-- Int type dictionary items
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('int', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int8', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int16', '3', 3, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int32', '4', 4, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int64', '5', 5, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('rune', '6', 6, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint', '7', 7, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint8', '8', 8, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint16', '9', 9, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint32', '10', 10, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint64', '11', 11, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uintptr', '12', 12, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('byte', '13', 13, (SELECT id FROM sys_dictionaries WHERE type = 'int'));

-- time.Time type dictionary items
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('time.Time', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'time.Time'));

-- float64 type dictionary items
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('float32', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'float64')),
('float64', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'float64'));

-- string type dictionary items
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('string', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'string'));

-- bool type dictionary items
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('bool', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'bool'));

-- ============================================================
-- Seed data: System configs
-- ============================================================
INSERT INTO sys_configs (category, key, value, value_type, description, is_public) VALUES
('system', 'system_name', 'OMC 网管系统', 'string', '系统名称', TRUE),
('system', 'system_version', '1.0.0', 'string', '系统版本', TRUE),
('system', 'session_timeout', '30', 'int', '会话超时时间（分钟）', FALSE),
('system', 'max_login_attempts', '5', 'int', '最大登录尝试次数', FALSE),
('system', 'lockout_duration', '30', 'int', '锁定时长（分钟）', FALSE),
('system', 'password_min_length', '6', 'int', '密码最小长度', FALSE);

-- +goose Down
DROP TABLE IF EXISTS sys_task_logs CASCADE;
DROP TABLE IF EXISTS sys_oper_logs CASCADE;
DROP TABLE IF EXISTS sys_login_logs CASCADE;
DROP TABLE IF EXISTS sys_configs CASCADE;
DROP TABLE IF EXISTS sys_dictionary_details CASCADE;
DROP TABLE IF EXISTS sys_dictionaries CASCADE;
DROP TABLE IF EXISTS role_menus CASCADE;
DROP TABLE IF EXISTS menus CASCADE;

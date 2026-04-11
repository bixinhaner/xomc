-- ============================================================
-- 000002_users_roles.up.sql
-- 用户、角色、权限、菜单管理
-- ============================================================

-- 1. 用户表
CREATE TABLE users (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username              VARCHAR(64) NOT NULL UNIQUE,
    password_hash         VARCHAR(256) NOT NULL,
    display_name          VARCHAR(128),
    email                 VARCHAR(256),
    carrier               VARCHAR(4),
    status                VARCHAR(16) NOT NULL DEFAULT 'active',
    last_login_at         TIMESTAMPTZ,
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until          TIMESTAMPTZ,
    last_failed_login_at  TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_username ON users (username);
CREATE INDEX idx_users_carrier ON users (carrier) WHERE carrier IS NOT NULL;
CREATE INDEX idx_users_status ON users (status);
CREATE TRIGGER trigger_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 2. 角色表
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT,
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trigger_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 3. 用户-角色映射
CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);
CREATE UNIQUE INDEX uniq_user_default_role ON user_roles(user_id) WHERE is_default = TRUE;

-- 4. 权限表
CREATE TABLE permissions (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id  UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource VARCHAR(64) NOT NULL,
    action   VARCHAR(16) NOT NULL,
    UNIQUE(role_id, resource, action)
);
CREATE INDEX idx_permissions_role ON permissions (role_id);

-- 5. API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(8) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- 6. 角色继承
CREATE TABLE role_inheritance (
    parent_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    child_role_id  UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    domain         VARCHAR(16) NOT NULL DEFAULT 'system',
    PRIMARY KEY (parent_role_id, child_role_id, domain)
);

-- 7. 用户自定义列配置
CREATE TABLE user_column_configs (
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    page_key     VARCHAR(64) NOT NULL,
    columns      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, page_key)
);
CREATE TRIGGER trigger_ucc_updated_at BEFORE UPDATE ON user_column_configs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 8. 仪表盘布局
CREATE TABLE dashboard_widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    layout JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);
CREATE TRIGGER trigger_dashboard_widgets_updated_at BEFORE UPDATE ON dashboard_widgets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

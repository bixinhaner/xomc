-- +goose Up

-- 角色-API 端点权限关联表
CREATE TABLE IF NOT EXISTS role_api_permissions (
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES api_endpoints(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, endpoint_id)
);

CREATE INDEX idx_role_api_permissions_role     ON role_api_permissions(role_id);
CREATE INDEX idx_role_api_permissions_endpoint ON role_api_permissions(endpoint_id);

COMMENT ON TABLE  role_api_permissions            IS '角色-API端点权限关联表';
COMMENT ON COLUMN role_api_permissions.role_id     IS '角色 ID';
COMMENT ON COLUMN role_api_permissions.endpoint_id IS 'API 端点 ID';

-- +goose Down

DROP TABLE IF EXISTS role_api_permissions;

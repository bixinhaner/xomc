-- +goose Up

-- API 端点管理表
CREATE TABLE IF NOT EXISTS api_endpoints (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    path        VARCHAR(256)    NOT NULL,
    method      VARCHAR(16)     NOT NULL,
    name        VARCHAR(128)    DEFAULT '',
    description TEXT            DEFAULT '',
    api_group   VARCHAR(64)     DEFAULT '',
    is_auto     BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE(path, method)
);

CREATE INDEX idx_api_endpoints_group ON api_endpoints(api_group);
CREATE INDEX idx_api_endpoints_method ON api_endpoints(method);

COMMENT ON TABLE api_endpoints IS 'API 端点注册表';
COMMENT ON COLUMN api_endpoints.path IS 'API 路径';
COMMENT ON COLUMN api_endpoints.method IS 'HTTP 方法 (GET/POST/PUT/DELETE)';
COMMENT ON COLUMN api_endpoints.name IS 'API 名称/简介';
COMMENT ON COLUMN api_endpoints.description IS 'API 详细描述';
COMMENT ON COLUMN api_endpoints.api_group IS 'API 分组名称';
COMMENT ON COLUMN api_endpoints.is_auto IS '是否自动扫描生成';

-- 数据权限增强：role_device_groups 增加网络类型字段
ALTER TABLE role_device_groups 
ADD COLUMN IF NOT EXISTS network_types TEXT[] NOT NULL DEFAULT '{}';

COMMENT ON COLUMN role_device_groups.network_types IS '角色可见的网络类型数组，如 {lte,nr,gsm}，空数组表示不限制';

-- +goose Down

ALTER TABLE role_device_groups DROP COLUMN IF EXISTS network_types;
DROP TABLE IF EXISTS api_endpoints;

-- ============================================================
-- 000080_create_sys_configs.up.sql
-- 系统配置参数表
-- ============================================================

CREATE TABLE sys_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category    VARCHAR(64) NOT NULL,                  -- 配置分类：system/email/backup/pm 等
    key         VARCHAR(128) NOT NULL,                 -- 配置键
    value       TEXT NOT NULL DEFAULT '',              -- 配置值
    value_type  VARCHAR(16) NOT NULL DEFAULT 'string', -- 值类型：string/int/float/bool/json
    description VARCHAR(255) NOT NULL DEFAULT '',      -- 说明
    is_public   BOOLEAN NOT NULL DEFAULT FALSE,        -- 是否公开（前端可见）
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uniq_config_key UNIQUE(category, key),
    CONSTRAINT chk_value_type CHECK (value_type IN ('string', 'int', 'float', 'bool', 'json'))
);

CREATE INDEX idx_sys_configs_category ON sys_configs(category);
CREATE INDEX idx_sys_configs_public ON sys_configs(is_public) WHERE is_public = TRUE;

COMMENT ON TABLE sys_configs IS '系统配置参数表';
COMMENT ON COLUMN sys_configs.category IS '配置分类：system/email/backup/pm 等';
COMMENT ON COLUMN sys_configs.is_public IS '是否公开（前端可见，如系统名称等）';

-- 初始化：系统基础配置
INSERT INTO sys_configs (category, key, value, value_type, description, is_public) VALUES
('system', 'system_name', 'OMC 网管系统', 'string', '系统名称', TRUE),
('system', 'system_version', '1.0.0', 'string', '系统版本', TRUE),
('system', 'session_timeout', '30', 'int', '会话超时时间（分钟）', FALSE),
('system', 'max_login_attempts', '5', 'int', '最大登录尝试次数', FALSE),
('system', 'lockout_duration', '30', 'int', '锁定时长（分钟）', FALSE),
('system', 'password_min_length', '6', 'int', '密码最小长度', FALSE);

-- ============================================================
-- 000068_create_user_column_configs.up.sql
-- 用户自定义列配置
-- ============================================================

CREATE TABLE user_column_configs (
    user_id      UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    page_key     VARCHAR(64)  NOT NULL,     -- 页面标识：device_list, alarm_list, ...
    columns      JSONB        NOT NULL,     -- 列配置 JSON 数组
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, page_key)
);

CREATE TRIGGER trigger_ucc_updated_at
    BEFORE UPDATE ON user_column_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

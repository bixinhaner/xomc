-- +goose Up
-- API 端点注册表加「用户手工修改」标记：
-- 当用户在 UI 编辑了某端点的 name/api_group 后置 true，Sync 扫描 gin 路由时
-- 用 CASE WHEN api_endpoints.is_user_modified THEN 保留原值 ELSE 用扫描值，
-- 避免自动同步覆盖用户已保存的修改。
-- 新增列带 DEFAULT false 且 NOT NULL，既有行/既有 seed 自动满足（§4.6 铁律）。
ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS is_user_modified boolean NOT NULL DEFAULT false;
COMMENT ON COLUMN api_endpoints.is_user_modified IS 'true=name/api_group 被用户在 UI 手工改过；Sync 扫描不再用自动推断值覆盖。Update 改 name/api_group 时置 true。';

-- +goose Down
ALTER TABLE api_endpoints DROP COLUMN IF EXISTS is_user_modified;

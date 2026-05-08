-- +goose Up
-- ============================================================
-- 000064_seed_role_api_permissions_viewer.sql
-- 为 viewer 内置角色补 role_api_permissions：仅授予全部 GET 类 API 端点
-- （只读权限，符合 viewer 角色语义）。
--
-- 背景（B3-Phase1，参 docs/prd/system/menu-dynamic-loading.md §4.2.4）：
--   现网 admin/operator 角色的 role_api_permissions 各 445 行（=api_endpoints
--   全集），但 viewer 0 行。Casbin LoadPolicy 已切到双源（permissions 表 +
--   role_api_permissions JOIN api_endpoints）；Phase2 替换中间件为 端点级
--   RequireAPIPermission 后，没有 role_api_permissions 数据的角色将所有 API
--   返回 403。本 seed 给 viewer 兜底。
--
-- 幂等：role_api_permissions 有 UNIQUE(role_id, endpoint_id)，ON CONFLICT 兜底。
-- ============================================================

INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000003'::uuid, ae.id
FROM api_endpoints ae
WHERE ae.method = 'GET'
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down
-- 仅清理本迁移注入的 viewer GET 类绑定，admin/operator/test 不动
DELETE FROM role_api_permissions
WHERE role_id = '10000000-0000-0000-0000-000000000003'::uuid
  AND endpoint_id IN (SELECT id FROM api_endpoints WHERE method = 'GET');

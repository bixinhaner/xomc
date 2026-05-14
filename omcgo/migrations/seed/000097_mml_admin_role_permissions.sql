-- +goose Up
-- ============================================================
-- 000097_mml_admin_role_permissions.sql
-- T-0123-P0 §M.4.2 — 给 admin 角色批量授权 13 个 catalog 管理 API 端点
--
-- 端点路径：/api/v1/mml/admin/* （api_group='mml_admin'）
-- 端点方法：POST / PATCH / DELETE / GET
-- 自注册：13 端点由 admin/api_endpoint_scanner 启动期扫描 Gin 路由表自动 UPSERT
--         到 api_endpoints 表（is_auto=TRUE）。本 seed 仅做 role × endpoint 绑定。
--
-- 部署 ordering 注意：
--   1. migrate-up 应用 schema 000095（mml_command_sub_fields 表 + 元数据列）
--   2. migrate-seed 应用 000096（standard params 2001 行）+ 000097（本文件）
--   3. app 首次启动 → api_endpoint_scanner 把 13 mml_admin 端点写入 api_endpoints
--   4. 第二次 migrate-seed（或下一次重新部署）→ 本 INSERT 真正生效
--
-- 若需首次部署即生效：执行 `omcctl admin reload-api-endpoints`（待 P3 admin UI 提供）
-- 或重启 app 后再跑一次 migrate-seed。
--
-- 幂等：role_api_permissions PK = (role_id, endpoint_id)，ON CONFLICT 兜底。
-- 复刻：mirror 000067 viewer GET 绑定模式（B3-Phase1 已验证）。
-- ============================================================

INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT r.id, ae.id
FROM roles r
JOIN api_endpoints ae ON ae.api_group = 'mml_admin'
WHERE r.code = 'admin'
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down
-- ============================================================
-- 反向：删除 admin × mml_admin 绑定（不动 api_endpoints 自注册结果）
-- ============================================================

DELETE FROM role_api_permissions
WHERE role_id IN (SELECT id FROM roles WHERE code = 'admin')
  AND endpoint_id IN (SELECT id FROM api_endpoints WHERE api_group = 'mml_admin');

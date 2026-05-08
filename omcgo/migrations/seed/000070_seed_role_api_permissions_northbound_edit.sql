-- 北向主备服务器编辑权限初始化（与 PUT /api/v1/northbound/servers/:role 配套）
--
-- 业务背景：system/config 北向设置页支持编辑主备 host/port/description。该写
-- 操作权限独立管控，不与"普通 northbound 数据导出 / 推送"混用，避免误授权。
--
-- 默认授权角色：
--   admin (10000000-...001)    — 系统管理员，必须有
--   operator (10000000-...002) — 运维管理员，业务上需要修改 OSS 对接配置
--   viewer (10000000-...003)   — 只读，不给（默认）
--   自定义角色                  — 由 RolePermission 页面手动勾选
--
-- 注意：app 启动期 syncApiEndpoints 自动注册 PUT /api/v1/northbound/servers/:role
-- 到 api_endpoints 表（router.go:507）。但 seed 跑在 app 启动前，那时 endpoint
-- 还没注册。本 seed 先用 INSERT ON CONFLICT 显式 upsert 注入 api_endpoints
-- （含 name / api_group），再 sub-select 拿 endpoint_id grant 给两个角色。
-- syncApiEndpoints 后续看 (path, method) 已存在 → upsert 走 update，本 seed
-- 设置的 name / api_group 与启动期 inferRouteName / inferApiGroup 推断一致，
-- 不会被覆盖回奇怪值。
--
-- 幂等：api_endpoints UNIQUE (path, method) + role_api_permissions UNIQUE
-- (role_id, endpoint_id)，ON CONFLICT DO NOTHING 兜底。

-- +goose Up
-- 1. 显式注册编辑端点到 api_endpoints（启动期 syncApiEndpoints 看 path/method 已
--    存在会跳过创建，避免 race）。
INSERT INTO api_endpoints (id, path, method, name, description, api_group, is_auto)
VALUES (
    '50000000-0000-0000-0000-000000000001'::uuid,
    '/api/v1/northbound/servers/:role',
    'PUT',
    'PUT /api/v1/northbound/servers/:role',
    '北向 OSS 主备服务器编辑（独立编辑权限）',
    'northbound',
    FALSE
)
ON CONFLICT (path, method) DO NOTHING;

-- 2. 给 admin 和 operator 角色 grant 此 endpoint。
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT
    role_id,
    ae.id
FROM (VALUES
    ('10000000-0000-0000-0000-000000000001'::uuid),  -- admin
    ('10000000-0000-0000-0000-000000000002'::uuid)   -- operator
) AS r(role_id)
CROSS JOIN api_endpoints ae
WHERE ae.path = '/api/v1/northbound/servers/:role' AND ae.method = 'PUT'
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down
-- 仅清理本 seed 注入的 admin/operator grant；endpoint 行不删（启动期会再 sync 进来）。
DELETE FROM role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM api_endpoints
    WHERE path = '/api/v1/northbound/servers/:role' AND method = 'PUT'
)
AND role_id IN (
    '10000000-0000-0000-0000-000000000001'::uuid,
    '10000000-0000-0000-0000-000000000002'::uuid
);

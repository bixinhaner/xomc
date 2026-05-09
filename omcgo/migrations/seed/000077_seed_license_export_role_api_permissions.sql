-- T-0100-P4-A: License 导出端点权限注入
--
-- 来源 PRD: docs/project/prd/F06-license.md §5.3.4 §6.5
--
-- 业务背景：T-0100-P4-A 新增 2 个导出端点：
--   GET /api/v1/licenses/:id/export?format=pdf|json   单条 PDF/JSON
--   GET /api/v1/licenses/export?format=csv             全量 active CSV
--
-- 与 P3（seed/000075）一致，采用"前端按钮门控（system:license:operate）+ 后端
-- 端点级 RBAC 兜底"双层防护：本 seed 显式注册端点到 api_endpoints +
-- grant 给 admin。其他角色按需在 /system/roles 自建。
--
-- 幂等：api_endpoints UNIQUE (path, method) + role_api_permissions UNIQUE
-- (role_id, endpoint_id)，ON CONFLICT DO NOTHING 兜底。

-- +goose Up
-- 1. 显式注册导出端点
INSERT INTO api_endpoints (id, path, method, name, description, api_group, is_auto)
VALUES
    ('50000000-0001-0000-0000-000000000004'::uuid, '/api/v1/licenses/:id/export', 'GET', 'GET /api/v1/licenses/:id/export', '许可证 - 单条导出（PDF/JSON）',  'license', FALSE),
    ('50000000-0001-0000-0000-000000000005'::uuid, '/api/v1/licenses/export',     'GET', 'GET /api/v1/licenses/export',     '许可证 - 全量导出（CSV）',       'license', FALSE)
ON CONFLICT (path, method) DO NOTHING;

-- 2. grant 给 admin
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, ae.id
FROM api_endpoints ae
WHERE (ae.path, ae.method) IN (
    ('/api/v1/licenses/:id/export', 'GET'),
    ('/api/v1/licenses/export',     'GET')
)
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down
DELETE FROM role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM api_endpoints
    WHERE (path, method) IN (
        ('/api/v1/licenses/:id/export', 'GET'),
        ('/api/v1/licenses/export',     'GET')
    )
)
AND role_id = '10000000-0000-0000-0000-000000000001'::uuid;

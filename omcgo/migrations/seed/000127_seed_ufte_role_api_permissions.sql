-- +goose Up
-- 为 admin 和 operator 角色分配 UFTE（统一文件传输）所有 API 端点权限
-- 这些端点在 000003_role_api_permissions.sql 中已注册到 api_endpoints，
-- 但当时缺少 role_api_permissions 绑定，导致 UFTE 页面 403。

-- +goose StatementBegin
DO $$
DECLARE
    _admin_role_id  UUID := '10000000-0000-0000-0000-000000000001';
    _operator_role_id UUID := '10000000-0000-0000-0000-000000000002';
BEGIN
    -- admin
    INSERT INTO role_api_permissions (role_id, endpoint_id)
    SELECT _admin_role_id, id
    FROM api_endpoints
    WHERE path LIKE '/api/v1/ufte%'
    ON CONFLICT DO NOTHING;

    -- operator
    INSERT INTO role_api_permissions (role_id, endpoint_id)
    SELECT _operator_role_id, id
    FROM api_endpoints
    WHERE path LIKE '/api/v1/ufte%'
    ON CONFLICT DO NOTHING;
END $$;
-- +goose StatementEnd

-- +goose Down
DELETE FROM role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM api_endpoints WHERE path LIKE '/api/v1/ufte%'
)
AND role_id IN (
    '10000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000002'
);

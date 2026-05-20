-- +goose Up
-- 为 admin 和 operator 角色分配 UFTE（统一文件传输）所有 API 端点权限。
--
-- 背景：api_endpoints 由 app 启动时 SyncApiEndpoints 自动注册，
-- 但 seed 迁移在 app 启动之前执行，此时 UFTE 端点尚不存在，
-- 导致 000127 原版 SELECT ... FROM api_endpoints 返回 0 行。
-- 修复：先预 INSERT UFTE 端点（ON CONFLICT DO NOTHING），再赋权。
-- 参考 000075_seed_license_operate_button_menu.sql 的同名模式。

-- 1. 预注册 UFTE 端点（app 启动后 SyncApiEndpoints 会 upsert 覆盖）
INSERT INTO api_endpoints (id, path, method, name, description, api_group, is_auto)
VALUES
    ('50000000-0002-0000-0000-000000000001'::uuid, '/api/v1/ufte/device-candidates',    'GET',    'GET /api/v1/ufte/device-candidates',    'UFTE - 候选设备列表',     'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000002'::uuid, '/api/v1/ufte/devices',               'GET',    'GET /api/v1/ufte/devices',               'UFTE - 设备执行列表',     'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000003'::uuid, '/api/v1/ufte/overview',               'GET',    'GET /api/v1/ufte/overview',               'UFTE - 总览面板',         'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000004'::uuid, '/api/v1/ufte/task-types',             'GET',    'GET /api/v1/ufte/task-types',             'UFTE - 任务类型列表',     'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000005'::uuid, '/api/v1/ufte/task-types',             'POST',   'POST /api/v1/ufte/task-types',            'UFTE - 创建任务类型',     'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000006'::uuid, '/api/v1/ufte/task-types/:typeCode',   'PUT',    'PUT /api/v1/ufte/task-types/:typeCode',   'UFTE - 更新任务类型',     'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000007'::uuid, '/api/v1/ufte/task-types/:typeCode',   'DELETE', 'DELETE /api/v1/ufte/task-types/:typeCode','UFTE - 删除任务类型',     'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000008'::uuid, '/api/v1/ufte/tasks',                  'GET',    'GET /api/v1/ufte/tasks',                  'UFTE - 任务列表',         'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000009'::uuid, '/api/v1/ufte/tasks',                  'POST',   'POST /api/v1/ufte/tasks',                 'UFTE - 创建任务',         'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000010'::uuid, '/api/v1/ufte/tasks/:id',              'DELETE', 'DELETE /api/v1/ufte/tasks/:id',            'UFTE - 删除任务',         'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000011'::uuid, '/api/v1/ufte/tasks/:id/retry',        'POST',   'POST /api/v1/ufte/tasks/:id/retry',       'UFTE - 重试任务',         'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000012'::uuid, '/api/v1/ufte/tasks/:id/start',        'PUT',    'PUT /api/v1/ufte/tasks/:id/start',        'UFTE - 启动任务',         'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000013'::uuid, '/api/v1/ufte/tasks/:id/suspend',      'PUT',    'PUT /api/v1/ufte/tasks/:id/suspend',      'UFTE - 暂停任务',         'ufte', FALSE),
    ('50000000-0002-0000-0000-000000000014'::uuid, '/api/v1/ufte/tasks/:id/terminate',    'PUT',    'PUT /api/v1/ufte/tasks/:id/terminate',    'UFTE - 终止任务',         'ufte', FALSE)
ON CONFLICT (path, method) DO NOTHING;

-- 2. admin — 全部 UFTE 端点
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, id
FROM api_endpoints
WHERE path LIKE '/api/v1/ufte%'
ON CONFLICT DO NOTHING;

-- 3. operator — 全部 UFTE 端点
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000002'::uuid, id
FROM api_endpoints
WHERE path LIKE '/api/v1/ufte%'
ON CONFLICT DO NOTHING;

-- +goose Down
-- 清理本 seed 注入的 admin/operator 权限绑定；endpoint 行不删
-- （app 启动期 SyncApiEndpoints 会再 sync 进来）。
DELETE FROM role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM api_endpoints WHERE path LIKE '/api/v1/ufte%'
)
AND role_id IN (
    '10000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000002'
);

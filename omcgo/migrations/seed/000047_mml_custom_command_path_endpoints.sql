-- +goose Up
-- issue #115 调整3（A1）：注册 4 个「自定义命令 PATH 管理」端点并授权内置角色。
--
-- 端点在 app 启动期会被 SyncApiEndpoints 自动 upsert（按 path+method）；这里显式 INSERT
-- 是因为 seed 在 migrate 期执行（早于启动自动同步），须先有确定的 api_endpoints 行才能
-- 授权 role_api_permissions。casbin 无 admin/super_admin 旁路，授权全靠 role_api_permissions。
--
-- 授权策略：admin/operator 拥有全部 4 端点；viewer 仅 GET（只读全集，与现有约定一致）。
INSERT INTO public.api_endpoints (id, path, method, name, api_group, is_auto)
VALUES
    (gen_random_uuid(), '/api/v1/mml/templates/:id/paths',         'GET',    'GET /api/v1/mml/templates/:id/paths',          'mml', true),
    (gen_random_uuid(), '/api/v1/mml/templates/:id/paths/batch',   'POST',   'POST /api/v1/mml/templates/:id/paths/batch',   'mml', true),
    (gen_random_uuid(), '/api/v1/mml/templates/:id/paths/:pathId', 'PATCH',  'PATCH /api/v1/mml/templates/:id/paths/:pathId','mml', true),
    (gen_random_uuid(), '/api/v1/mml/templates/:id/paths/:pathId', 'DELETE', 'DELETE /api/v1/mml/templates/:id/paths/:pathId','mml', true)
ON CONFLICT (path, method) DO NOTHING;

-- admin + operator：全部 4 端点
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT r.id, ae.id
FROM public.roles r
JOIN public.api_endpoints ae
    ON ae.path IN (
        '/api/v1/mml/templates/:id/paths',
        '/api/v1/mml/templates/:id/paths/batch',
        '/api/v1/mml/templates/:id/paths/:pathId'
    )
WHERE r.name IN ('admin', 'operator')
ON CONFLICT DO NOTHING;

-- viewer：仅 GET
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT r.id, ae.id
FROM public.roles r
JOIN public.api_endpoints ae
    ON ae.path = '/api/v1/mml/templates/:id/paths' AND ae.method = 'GET'
WHERE r.name = 'viewer'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM public.role_api_permissions rap
USING public.api_endpoints ae
WHERE rap.endpoint_id = ae.id
  AND ae.path LIKE '/api/v1/mml/templates/:id/paths%';

DELETE FROM public.api_endpoints
WHERE path LIKE '/api/v1/mml/templates/:id/paths%';

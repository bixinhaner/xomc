-- +goose Up

-- admin：沿用内置角色兼容基线，补齐全部已登记 API。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, ae.id
FROM public.api_endpoints AS ae
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- operator：沿用内置角色兼容基线，补齐全部已登记 API。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000002'::uuid, ae.id
FROM public.api_endpoints AS ae
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- viewer：只补齐只读 API，不授予写操作。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000003'::uuid, ae.id
FROM public.api_endpoints AS ae
WHERE ae.method = 'GET'
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down
-- 无法区分本迁移新增权限与历史合法授权，禁止破坏性回滚。
SELECT 1;

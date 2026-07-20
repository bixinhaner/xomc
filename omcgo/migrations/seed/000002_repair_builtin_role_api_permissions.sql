-- +goose Up

-- admin：兼容历史行为，授权全部已登记 API。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, ae.id
FROM public.api_endpoints AS ae
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- operator：兼容历史行为，授权全部已登记 API。
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
-- 数据修复无法安全区分既有授权与本 migration 新增授权，禁止破坏性回滚。
SELECT 1;

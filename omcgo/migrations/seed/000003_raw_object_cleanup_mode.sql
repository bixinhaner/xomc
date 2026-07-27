-- +goose Up
INSERT INTO public.sys_configs (
    id, category, key, value, value_type, description, is_public,
    created_at, updated_at, description_i18n
) VALUES (
    'ef07a965-4e50-4a27-af18-e14dca393d7e',
    'minio.retention',
    'cleanup_mode',
    'shadow',
    'string',
    '原始对象精确清理阶段：shadow/fallback/exclusive',
    false,
    now(),
    now(),
    '{}'::jsonb
)
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM public.sys_configs
 WHERE category = 'minio.retention'
   AND key = 'cleanup_mode';

-- +goose Up
UPDATE public.menus
SET sort_order = 1,
    show_status = 'show',
    updated_at = NOW()
WHERE permission_key = 'mml:console';

UPDATE public.menus
SET name = '脚本管理',
    name_i18n = COALESCE(name_i18n, '{}'::jsonb) || '{"zh-CN": "脚本管理", "en-US": "Script Management"}'::jsonb,
    sort_order = 2,
    show_status = 'show',
    updated_at = NOW()
WHERE permission_key = 'mml:script';

-- +goose Down
UPDATE public.menus
SET name = '脚本任务',
    name_i18n = COALESCE(name_i18n, '{}'::jsonb) || '{"zh-CN": "脚本任务", "en-US": "Script Tasks"}'::jsonb,
    sort_order = 2,
    show_status = 'show',
    updated_at = NOW()
WHERE permission_key = 'mml:script';

UPDATE public.menus
SET sort_order = 2,
    show_status = 'show',
    updated_at = NOW()
WHERE permission_key = 'mml:console';

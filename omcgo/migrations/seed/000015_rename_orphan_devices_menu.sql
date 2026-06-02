-- +goose Up
-- 菜单「孤儿设备」改名为「未知设备」（产品中心 > product/orphan-devices）。
-- 同步更新 legacy name 列与 name_i18n（zh-CN / en-US），按菜单 id 定位。
UPDATE public.menus
   SET name = '未知设备',
       name_i18n = jsonb_set(
                     jsonb_set(COALESCE(name_i18n, '{}'::jsonb), '{zh-CN}', '"未知设备"'),
                     '{en-US}', '"Unknown Devices"'),
       updated_at = now()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000005';

-- +goose Down
UPDATE public.menus
   SET name = '孤儿设备',
       name_i18n = jsonb_set(
                     jsonb_set(COALESCE(name_i18n, '{}'::jsonb), '{zh-CN}', '"孤儿设备"'),
                     '{en-US}', '"Orphan Devices"'),
       updated_at = now()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000005';

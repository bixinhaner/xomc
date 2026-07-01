-- +goose Up
-- 将系统内置二级默认设备组（ID ...0002）的展示名改为「默认设备组」，
-- 与当前前端页面命名保持一致；该节点语义仍是「未绑定任何分组的设备视图」。

UPDATE public.device_groups
SET name = '默认设备组',
    name_i18n = '{"en-US": "Default Group", "zh-CN": "默认设备组"}'::jsonb,
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000002';

-- +goose Down
UPDATE public.device_groups
SET name = '未分组设备',
    name_i18n = '{"en-US": "Ungrouped Devices", "zh-CN": "未分组设备"}'::jsonb,
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000002';
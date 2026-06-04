-- +goose Up
-- 菜单「参数模型」改名为「参数模型库」(2026-06-04 用户决策)。
-- 菜单数据在 menus 表(seed/000001 初始化);按 id 精确 UPDATE name + name_i18n。
-- 不改已 applied 的 000001(避免 goose checksum 漂移),新增本迁移做改名。
-- UPDATE 幂等:重跑改成同值无副作用。
UPDATE menus
SET name = '参数模型库',
    name_i18n = '{"zh-CN": "参数模型库", "en-US": "Param Model Library"}'::jsonb,
    updated_at = now()
WHERE id = 'aaaa0098-1000-0000-0000-000000000002';

-- +goose Down
UPDATE menus
SET name = '参数模型',
    name_i18n = '{"zh-CN": "参数模型", "en-US": "Param Models"}'::jsonb,
    updated_at = now()
WHERE id = 'aaaa0098-1000-0000-0000-000000000002';

-- 文件传输菜单拆分修正
--
-- 目标：将“文件传输中心”拆分为两个独立菜单入口：
-- 1. 任务创建：复用 /transfer/center 任务发起页
-- 2. 模板配置：复用 /transfer/template-management 模板治理页
--
-- 说明：
-- - 不改历史 seed，新增增量修正，兼容已落库环境。
-- - 模板配置默认给 admin 角色可见；其他角色若已有手工授权，不在此删除。

-- +goose Up

UPDATE menus
SET name = '任务创建',
    name_i18n = '{"zh-CN":"任务创建","en-US":"Task Creation"}'::jsonb,
    updated_at = NOW()
WHERE id = 'aaaa000b-1000-0000-0000-000000000001'::uuid;

UPDATE menus
SET name = '模板配置',
    name_i18n = '{"zh-CN":"模板配置","en-US":"Template Configuration"}'::jsonb,
    updated_at = NOW()
WHERE id = 'aaaa000b-1000-0000-0000-000000000002'::uuid;

INSERT INTO role_menus (role_id, menu_id)
VALUES (
    '10000000-0000-0000-0000-000000000001'::uuid,
    'aaaa000b-1000-0000-0000-000000000002'::uuid
)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down

DELETE FROM role_menus
WHERE role_id = '10000000-0000-0000-0000-000000000001'::uuid
  AND menu_id = 'aaaa000b-1000-0000-0000-000000000002'::uuid;

UPDATE menus
SET name = '文件传输中心',
    name_i18n = '{"zh-CN":"文件传输中心","en-US":"File Transfer Center"}'::jsonb,
    updated_at = NOW()
WHERE id = 'aaaa000b-1000-0000-0000-000000000001'::uuid;

UPDATE menus
SET name = '传输模板管理',
    name_i18n = '{"zh-CN":"传输模板管理","en-US":"Transfer Template Management"}'::jsonb,
    updated_at = NOW()
WHERE id = 'aaaa000b-1000-0000-0000-000000000002'::uuid;
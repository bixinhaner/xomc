-- 文件传输中心（UFTE 预览入口）菜单 seed
--
-- 目标：在不影响现有升级/日志菜单的前提下，新增一个独立入口给领导和项目组先看
-- UI 形态。后续统一引擎落地时，新任务只从该入口创建，旧入口逐步转发。
--
-- 幂等：menus PK + role_menus UNIQUE(role_id, menu_id) + ON CONFLICT 兜底。

-- +goose Up

-- 1. 一级目录：文件传输
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, icon, status, show_status
)
VALUES (
    'aaaa000b-0000-0000-0000-000000000001'::uuid,
    '文件传输',
    '{"zh-CN":"文件传输","en-US":"File Transfer"}'::jsonb,
    'directory',
    'transfer',
    NULL,
    12,
    '',
    'CloudServerOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 2. 二级页面：文件传输中心
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa000b-1000-0000-0000-000000000001'::uuid,
    '文件传输中心',
    '{"zh-CN":"文件传输中心","en-US":"File Transfer Center"}'::jsonb,
    'menu',
    'transfer:center',
    'aaaa000b-0000-0000-0000-000000000001'::uuid,
    1,
    '/transfer/center',
    'transfer/FileTransferCenter',
    'CloudServerOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 3. 角色默认绑定：admin / operator / viewer 全部可见
INSERT INTO role_menus (role_id, menu_id)
SELECT role_id, menu_id
FROM (
    VALUES
        ('10000000-0000-0000-0000-000000000001'::uuid, 'aaaa000b-0000-0000-0000-000000000001'::uuid),
        ('10000000-0000-0000-0000-000000000001'::uuid, 'aaaa000b-1000-0000-0000-000000000001'::uuid),
        ('10000000-0000-0000-0000-000000000002'::uuid, 'aaaa000b-0000-0000-0000-000000000001'::uuid),
        ('10000000-0000-0000-0000-000000000002'::uuid, 'aaaa000b-1000-0000-0000-000000000001'::uuid),
        ('10000000-0000-0000-0000-000000000003'::uuid, 'aaaa000b-0000-0000-0000-000000000001'::uuid),
        ('10000000-0000-0000-0000-000000000003'::uuid, 'aaaa000b-1000-0000-0000-000000000001'::uuid)
) AS bindings(role_id, menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
DELETE FROM role_menus
WHERE menu_id IN (
    'aaaa000b-0000-0000-0000-000000000001'::uuid,
    'aaaa000b-1000-0000-0000-000000000001'::uuid
);

DELETE FROM menus
WHERE id IN (
    'aaaa000b-1000-0000-0000-000000000001'::uuid,
    'aaaa000b-0000-0000-0000-000000000001'::uuid
);
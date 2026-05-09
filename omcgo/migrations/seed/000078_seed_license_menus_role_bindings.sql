-- T-0100-P6: License 模块菜单补全 + role_menus 默认绑定
--
-- 来源 PRD: docs/project/prd/F06-license.md §6.5
--
-- 关闭 P3 阶段已知遗留：seed/000075 仅 seed 了 system:license:operate button
-- 节点（parent_id=NULL orphan），明确注释「/license/* 一级菜单暂未 seed，
-- T-0100-P4 完整菜单时再补 parent_id」—— 但 P4-A/B/C 三子项都没做这步，
-- 导致 /system/roles 编辑角色时菜单权限树看不到 license 模块，无法授权。
--
-- 本迁移补全：
--   1. menus 一级目录 `/license` (directory, sort_order=10) — 排在 system 后
--   2. menus 二级 page menus (3): /license/list, /license/operations, /license/logs
--   3. menus 三级 button menus (3): system:license:view / operate / audit
--      - operate button 是更新（UPDATE parent_id），把已存在的孤儿节点重挂
--      - view / audit 是新增
--   4. role_menus 默认绑定（PRD §6.5）:
--      - admin (super_admin)：directory + 3 page + 3 button (全部 7 节点)
--      - operator：directory + list page + view button (3 节点；只读，不操作不审计)
--      - viewer：directory + list page + view button (3 节点；同 operator 默认)
--
-- 设计取舍：
--   - operator/viewer 默认相同（仅看）—— PRD §6.5 把 operate 限定 super_admin、
--     audit 限定 super_admin（有可选 license-admin/auditor 角色但 Q3=C 决议不
--     新增内置角色）；企业按需在 /system/roles 自助加权
--   - 后端 RBAC：viewer 已通过 seed/000067 默认 grant 全部 GET 端点；admin/
--     operator 默认 grant 全部端点（包括 POST license operate）。本 seed 仅控
--     菜单可见性 + UI 按钮可点性，**不**改 role_api_permissions（避免影响既有
--     行为；如需收紧 operator 的 POST 权限属另一个 task）
--
-- 命名约定：
--   - UUID 沿用 P3 的 aaaa0009 namespace（aaaa0009-0000=dir / aaaa0009-1000-N=
--     page / aaaa0009-1100-N=button），与 system/config 的 aaaa0008 namespace 区分
--   - permission_key 三段式：directory='license' / menu='license:list' /
--     button='system:license:operate' 等
--
-- 幂等：menus PK + role_menus UNIQUE (role_id, menu_id) + ON CONFLICT 兜底，
-- 可重复执行；operate button 走 ON CONFLICT (id) DO UPDATE 重挂 parent_id。

-- +goose Up
-- ============================================================
-- 1. 一级目录 `/license`
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES (
    'aaaa0009-0000-0000-0000-000000000001'::uuid,
    '许可证管理',
    'directory',
    'license',
    NULL,
    10,                          -- 排在 system (sort=9) 之后
    '',
    'SafetyCertificateOutlined',  -- AntD 图标，与"证书/许可"语义贴合
    'normal',
    'show'
)
ON CONFLICT (id) DO UPDATE SET
    name=EXCLUDED.name, type=EXCLUDED.type, permission_key=EXCLUDED.permission_key,
    parent_id=EXCLUDED.parent_id, sort_order=EXCLUDED.sort_order, icon=EXCLUDED.icon,
    status=EXCLUDED.status, show_status=EXCLUDED.show_status, updated_at=NOW();

-- ============================================================
-- 2. 二级 page menus (3) — list / operations / logs
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES
    ('aaaa0009-1000-0000-0000-000000000001'::uuid, '许可证列表', 'menu', 'license:list',       'aaaa0009-0000-0000-0000-000000000001'::uuid, 1, '/license/list',       'UnorderedListOutlined', 'normal', 'show'),
    ('aaaa0009-1000-0000-0000-000000000002'::uuid, '许可证操作', 'menu', 'license:operations', 'aaaa0009-0000-0000-0000-000000000001'::uuid, 2, '/license/operations', 'ToolOutlined',          'normal', 'show'),
    ('aaaa0009-1000-0000-0000-000000000003'::uuid, '审计日志',   'menu', 'license:logs',       'aaaa0009-0000-0000-0000-000000000001'::uuid, 3, '/license/logs',       'FileSearchOutlined',    'normal', 'show')
ON CONFLICT (id) DO UPDATE SET
    name=EXCLUDED.name, permission_key=EXCLUDED.permission_key, parent_id=EXCLUDED.parent_id,
    sort_order=EXCLUDED.sort_order, route_path=EXCLUDED.route_path, icon=EXCLUDED.icon,
    status=EXCLUDED.status, show_status=EXCLUDED.show_status, updated_at=NOW();

-- ============================================================
-- 3. 三级 button menus (3) — view / operate (重挂 parent) / audit
-- ============================================================
-- 3a. system:license:view (新增) — 挂在 /license/list 下
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, status, show_status)
VALUES (
    'aaaa0009-1100-0000-0000-000000000002'::uuid,
    '许可证查看',
    'button',
    'system:license:view',
    'aaaa0009-1000-0000-0000-000000000001'::uuid,  -- /license/list page
    1,
    '',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 3b. system:license:operate (P3 已存在 orphan) — UPDATE 重挂到 /license/operations
UPDATE menus
SET parent_id = 'aaaa0009-1000-0000-0000-000000000002'::uuid,  -- /license/operations page
    sort_order = 1,
    updated_at = NOW()
WHERE id = 'aaaa0009-1100-0000-0000-000000000001'::uuid
  AND parent_id IS NULL;  -- 仅当还是 orphan 时才更新（重复执行幂等）

-- 3c. system:license:audit (新增) — 挂在 /license/logs 下
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, status, show_status)
VALUES (
    'aaaa0009-1100-0000-0000-000000000003'::uuid,
    '审计日志查看',
    'button',
    'system:license:audit',
    'aaaa0009-1000-0000-0000-000000000003'::uuid,  -- /license/logs page
    1,
    '',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 4. role_menus 默认绑定
-- ============================================================
-- 4a. admin (10000000-...001): 全部 7 节点（directory + 3 page + 3 button）
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, menu_id
FROM (VALUES
    ('aaaa0009-0000-0000-0000-000000000001'::uuid),  -- directory
    ('aaaa0009-1000-0000-0000-000000000001'::uuid),  -- list page
    ('aaaa0009-1000-0000-0000-000000000002'::uuid),  -- operations page
    ('aaaa0009-1000-0000-0000-000000000003'::uuid),  -- logs page
    ('aaaa0009-1100-0000-0000-000000000002'::uuid),  -- view button
    ('aaaa0009-1100-0000-0000-000000000001'::uuid),  -- operate button (P3 复用)
    ('aaaa0009-1100-0000-0000-000000000003'::uuid)   -- audit button
) AS m(menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 4b. operator (10000000-...002): directory + list page + view button (3 节点)
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000002'::uuid, menu_id
FROM (VALUES
    ('aaaa0009-0000-0000-0000-000000000001'::uuid),  -- directory
    ('aaaa0009-1000-0000-0000-000000000001'::uuid),  -- list page
    ('aaaa0009-1100-0000-0000-000000000002'::uuid)   -- view button
) AS m(menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 4c. viewer (10000000-...003): directory + list page + view button (3 节点)
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000003'::uuid, menu_id
FROM (VALUES
    ('aaaa0009-0000-0000-0000-000000000001'::uuid),  -- directory
    ('aaaa0009-1000-0000-0000-000000000001'::uuid),  -- list page
    ('aaaa0009-1100-0000-0000-000000000002'::uuid)   -- view button
) AS m(menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
-- 清理 role_menus 默认绑定（admin/operator/viewer 7 + 3 + 3 = 13 行；ON CASCADE 也会
-- 跟 menus DROP 一起带走，但显式删避免 down 中间状态有残留）
DELETE FROM role_menus
WHERE menu_id IN (
    'aaaa0009-0000-0000-0000-000000000001'::uuid,
    'aaaa0009-1000-0000-0000-000000000001'::uuid,
    'aaaa0009-1000-0000-0000-000000000002'::uuid,
    'aaaa0009-1000-0000-0000-000000000003'::uuid,
    'aaaa0009-1100-0000-0000-000000000002'::uuid,
    'aaaa0009-1100-0000-0000-000000000003'::uuid
);

-- 把 operate button 重新还原成 orphan（保留 P3 既有节点行不删，避免影响其他迁移）
UPDATE menus
SET parent_id = NULL, sort_order = 100, updated_at = NOW()
WHERE id = 'aaaa0009-1100-0000-0000-000000000001'::uuid;

-- 删除本迁移新增的 view + audit button + 3 page + 1 directory（共 6 个新节点）
DELETE FROM menus
WHERE id IN (
    'aaaa0009-1100-0000-0000-000000000002'::uuid,  -- view button
    'aaaa0009-1100-0000-0000-000000000003'::uuid,  -- audit button
    'aaaa0009-1000-0000-0000-000000000001'::uuid,  -- list page
    'aaaa0009-1000-0000-0000-000000000002'::uuid,  -- operations page
    'aaaa0009-1000-0000-0000-000000000003'::uuid,  -- logs page
    'aaaa0009-0000-0000-0000-000000000001'::uuid   -- directory
);

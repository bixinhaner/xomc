-- +goose Up
-- ============================================================
-- 000063_seed_role_menus_builtin.sql
-- 为 3 个内置角色（admin/operator/viewer）补 role_menus 关联，避免
-- 切换到动态菜单数据源（PRD docs/prd/system/menu-dynamic-loading.md
-- §4.2.4 折中方案 C）后非 builtIn 用户菜单空白。
--
-- 策略：
--   admin    (10000000-0000-0000-0000-000000000001) — 全量菜单（虽然 admin
--            走超管旁路 source='builtIn' 不依赖 role_menus，但绑全做兜底）
--   operator (10000000-0000-0000-0000-000000000002) — 除"系统管理"目录
--            及其子树之外的全部菜单
--   viewer   (10000000-0000-0000-0000-000000000003) — 全部 directory +
--            menu，仅"查询"/"导出"两类 button
--
-- test 角色（81a186f4-...）保持现状，由用户通过 RolePermission UI 重新配置。
--
-- 幂等：role_menus 有 UNIQUE(role_id, menu_id)，ON CONFLICT 兜底。
-- ============================================================

-- admin：绑定全部 status='normal' 的菜单
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, id
FROM menus
WHERE status = 'normal'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- operator：除"系统管理"目录（11111111-1111-1111-1111-111111111108）及其后代外的菜单。
-- 用 CTE 递归收集系统管理目录的全部后代 ID，作为 NOT IN 排除集合。
-- +goose StatementBegin
WITH RECURSIVE system_descendants AS (
    SELECT id FROM menus WHERE id = '11111111-1111-1111-1111-111111111108'::uuid
    UNION ALL
    SELECT m.id FROM menus m
    INNER JOIN system_descendants sd ON m.parent_id = sd.id
)
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000002'::uuid, m.id
FROM menus m
WHERE m.status = 'normal'
  AND m.id NOT IN (SELECT id FROM system_descendants)
ON CONFLICT (role_id, menu_id) DO NOTHING;
-- +goose StatementEnd

-- viewer：全部 directory + menu，仅 button 类型的"查询"/"导出"
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000003'::uuid, id
FROM menus
WHERE status = 'normal'
  AND (
    type IN ('directory', 'menu')
    OR (type = 'button' AND name IN ('查询', '导出'))
  )
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down
-- 仅清空本迁移写入的 3 个内置角色 role_menus（test 角色不动）
DELETE FROM role_menus
WHERE role_id IN (
    '10000000-0000-0000-0000-000000000001'::uuid,
    '10000000-0000-0000-0000-000000000002'::uuid,
    '10000000-0000-0000-0000-000000000003'::uuid
);

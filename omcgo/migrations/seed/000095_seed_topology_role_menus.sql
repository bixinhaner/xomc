-- +goose Up
-- ============================================================
-- 绑定拓扑管理子菜单到 admin/super_admin 角色
-- 确保动态菜单模式下拓扑管理子菜单可见
-- ============================================================
--
-- 拓扑管理已有 6 个子菜单（GIS地图/拓扑图/域管理/站点管理/拓扑设置/图例管理），
-- 但 role_menus 表中只绑定了"拓扑图"，其他 5 个子菜单未绑定。
--
-- 本迁移补全缺失的角色菜单绑定。

-- 1. 绑定 5 个缺失的拓扑子菜单到 admin 角色
INSERT INTO role_menus (role_id, menu_id, created_by, created_at)
SELECT
  '10000000-0000-0000-0000-000000000001'::uuid,  -- admin role_id
  m.id,
  NULL,
  NOW()
FROM menus m
WHERE m.id::text LIKE 'aaaa0004-1000-%'
  AND m.type = 'menu'
  AND NOT EXISTS (
    SELECT 1 FROM role_menus rm
    WHERE rm.role_id = '10000000-0000-0000-0000-000000000001'::uuid
      AND rm.menu_id = m.id
  );

-- 2. 绑定 5 个缺失的拓扑子菜单到 super_admin 角色（如果存在）
DO $$
DECLARE
  super_admin_role_id UUID;
BEGIN
  SELECT id INTO super_admin_role_id FROM roles WHERE code = 'super_admin' LIMIT 1;

  IF super_admin_role_id IS NOT NULL THEN
    INSERT INTO role_menus (role_id, menu_id, created_by, created_at)
    SELECT
      super_admin_role_id,
      m.id,
      NULL,
      NOW()
    FROM menus m
    WHERE m.id::text LIKE 'aaaa0004-1000-%'
      AND m.type = 'menu'
      AND NOT EXISTS (
        SELECT 1 FROM role_menus rm
        WHERE rm.role_id = super_admin_role_id
          AND rm.menu_id = m.id
      );
  END IF;
END $$;

-- +goose Down
-- ============================================================
-- 删除本次迁移新增的角色菜单绑定
-- ============================================================

-- 删除 admin 角色的绑定
DELETE FROM role_menus
WHERE role_id = '10000000-0000-0000-0000-000000000001'::uuid
  AND menu_id::text LIKE 'aaaa0004-1000-%'
  AND EXISTS (
    SELECT 1 FROM menus m
    WHERE m.id = role_menus.menu_id
      AND m.id::text LIKE 'aaaa0004-1000-%'
      AND m.type = 'menu'
  );

-- 删除 super_admin 角色的绑定（如果存在）
DO $$
DECLARE
  super_admin_role_id UUID;
BEGIN
  SELECT id INTO super_admin_role_id FROM roles WHERE code = 'super_admin' LIMIT 1;

  IF super_admin_role_id IS NOT NULL THEN
    DELETE FROM role_menus
    WHERE role_id = super_admin_role_id
      AND menu_id::text LIKE 'aaaa0004-1000-%'
      AND EXISTS (
        SELECT 1 FROM menus m
        WHERE m.id = role_menus.menu_id
          AND m.id::text LIKE 'aaaa0004-1000-%'
          AND m.type = 'menu'
      );
  END IF;
END $$;

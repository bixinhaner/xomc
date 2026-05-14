-- +goose Up
-- ============================================================
-- 000095_seed_topology_role_menus.sql
-- 补全拓扑管理子菜单 + 绑定角色权限
-- ============================================================
--
-- 问题：000057_refresh_menu_seed.sql 只创建了 1 个拓扑子菜单（拓扑图），
--      但前端 navConfig.ts 定义了 6 个子菜单，导致动态菜单模式下子菜单不可见。
--
-- 解决：
--   1. 创建缺失的 5 个拓扑子菜单（GIS地图/域管理/站点管理/拓扑设置/图例管理）
--   2. 绑定全部 6 个拓扑子菜单到 admin/super_admin 角色
--
-- UUID 命名约定（沿用 000057 的 aaaa0004-1000-XXXX 前缀）：
--   - aaaa0004-0000-0000-0000-000000000001 : 拓扑管理目录（已存在）
--   - aaaa0004-1000-0000-0000-000000000001 : 拓扑图（已存在）
--   - aaaa0004-1000-0001-0000-000000000001 : GIS地图（新增）
--   - aaaa0004-1000-0002-0000-000000000001 : 域管理（新增）
--   - aaaa0004-1000-0003-0000-000000000001 : 站点管理（新增）
--   - aaaa0004-1000-0004-0000-000000000001 : 拓扑设置（新增）
--   - aaaa0004-1000-0005-0000-000000000001 : 图例管理（新增）
-- ============================================================

-- ------------------------------------------------------------
-- 1. 创建缺失的 5 个拓扑子菜单
-- ------------------------------------------------------------
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status) VALUES
  ('aaaa0004-1000-0001-0000-000000000001', 'GIS地图', 'menu', 'topology:gis-map',   'aaaa0004-0000-0000-0000-000000000001', 0, '/topology/gis-map',   'EnvironmentOutlined', 'normal'),
  ('aaaa0004-1000-0002-0000-000000000001', '域管理',   'menu', 'topology:domain',   'aaaa0004-0000-0000-0000-000000000001', 2, '/topology/domain',   'ApartmentOutlined',    'normal'),
  ('aaaa0004-1000-0003-0000-000000000001', '站点管理', 'menu', 'topology:site',     'aaaa0004-0000-0000-0000-000000000001', 3, '/topology/site',     'BuildOutlined',        'normal'),
  ('aaaa0004-1000-0004-0000-000000000001', '拓扑设置', 'menu', 'topology:settings', 'aaaa0004-0000-0000-0000-000000000001', 4, '/topology/settings', 'SettingOutlined',      'normal'),
  ('aaaa0004-1000-0005-0000-000000000001', '图例管理', 'menu', 'topology:legend',   'aaaa0004-0000-0000-0000-000000000001', 5, '/topology/legend',   'BgColorsOutlined',     'normal')
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name,
  permission_key=EXCLUDED.permission_key,
  parent_id=EXCLUDED.parent_id,
  sort_order=EXCLUDED.sort_order,
  route_path=EXCLUDED.route_path,
  icon=EXCLUDED.icon,
  status=EXCLUDED.status,
  updated_at=NOW();

-- ------------------------------------------------------------
-- 2. 绑定全部 6 个拓扑子菜单到 admin 角色
-- ------------------------------------------------------------
INSERT INTO role_menus (role_id, menu_id, created_by, created_at)
SELECT
  '10000000-0000-0000-0000-000000000001'::uuid,
  m.id,
  NULL,
  NOW()
FROM menus m
WHERE m.id::text LIKE 'aaaa0004-1000-%'
  AND m.type = 'menu'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- ------------------------------------------------------------
-- 3. 绑定全部 6 个拓扑子菜单到 super_admin 角色（如果存在）
-- ------------------------------------------------------------
-- +goose StatementBegin
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
    ON CONFLICT (role_id, menu_id) DO NOTHING;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- ============================================================
-- 删除本次迁移新增的菜单和角色绑定
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
-- +goose StatementBegin
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
-- +goose StatementEnd

-- 删除新增的 5 个菜单（保留原有的拓扑图菜单）
DELETE FROM menus
WHERE id IN (
  'aaaa0004-1000-0001-0000-000000000001',
  'aaaa0004-1000-0002-0000-000000000001',
  'aaaa0004-1000-0003-0000-000000000001',
  'aaaa0004-1000-0004-0000-000000000001',
  'aaaa0004-1000-0005-0000-000000000001'
);

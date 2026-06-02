-- +goose Up
-- 把"仪表板"从【目录 + 同名子菜单】两条记录,规整为【单个顶层菜单】一条。
--
-- 背景:seed/000001 里仪表板是 directory(顶层,aaaa0001-0000)+ menu(子,
-- aaaa0001-1000,/dashboard)两条;侧边栏显示为"仪表板"组里套一个同名"仪表板",
-- 冗余。本迁移保留菜单节点(路由 /dashboard、权限 key=dashboard:home 不变),
-- 把它提升为顶层,删掉空目录。NavMenu 对 type='menu' 的节点在任意层级都渲染为
-- 可点击项,故顶层菜单显示正常。
--
-- 三处约束(务必按此顺序,否则报错或误删):
--   1. menus 自引用 FK menus_parent_id_fkey ON DELETE CASCADE —— 不能先删目录,
--      否则会把子菜单一起级联删掉。
--   2. uniq_menu_name_per_parent (parent_id NULLS FIRST, name) WHERE status='normal'
--      —— 菜单提升到顶层前,必须先腾出 (NULL,'仪表板') 唯一槽(目录占着)。
--   3. role_menus FK ON DELETE CASCADE —— 删目录会带走目录的角色授权(seed 里 3 个
--      角色授了该目录),故先把授权改挂到菜单上(超管走 builtIn 旁路不依赖此表,
--      普通角色需要 menu 行才看得到)。

-- (1) 目录被授权的角色,补一份菜单授权(菜单授权多数已存在,ON CONFLICT 跳过)
INSERT INTO role_menus (role_id, menu_id, created_by, created_at)
SELECT role_id, 'aaaa0001-1000-0000-0000-000000000001', created_by, now()
FROM role_menus
WHERE menu_id = 'aaaa0001-0000-0000-0000-000000000001'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- (2) 目录临时改名,腾出 (NULL,'仪表板') 唯一槽
UPDATE menus
SET name = '仪表板(目录-待删)', updated_at = now()
WHERE id = 'aaaa0001-0000-0000-0000-000000000001' AND type = 'directory';

-- (3) 仪表板菜单提升为顶层,排到最前(原目录 sort_order=0)
UPDATE menus
SET parent_id = NULL, sort_order = 0, updated_at = now()
WHERE id = 'aaaa0001-1000-0000-0000-000000000001';

-- (4) 删除空目录(子菜单已 reparent 走,不级联;目录的 role_menus 由 FK 级联清除)
DELETE FROM menus WHERE id = 'aaaa0001-0000-0000-0000-000000000001';

-- +goose Down
-- 还原为【目录 + 子菜单】结构。

-- (1) 重建目录(临时名避免与当前顶层菜单 (NULL,'仪表板') 撞唯一约束)
INSERT INTO menus
  (id, name, type, permission_key, parent_id, sort_order, route_path, component_path, icon, show_status, status, created_at, updated_at, name_i18n)
VALUES
  ('aaaa0001-0000-0000-0000-000000000001', '仪表板(目录-待删)', 'directory', 'dashboard', NULL, 0, NULL, NULL, 'DashboardOutlined', 'show', 'normal', now(), now(),
   '{"en-US": "Dashboard", "zh-CN": "仪表板"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- (2) 菜单挂回目录下,腾出 (NULL,'仪表板') 槽
UPDATE menus
SET parent_id = 'aaaa0001-0000-0000-0000-000000000001', sort_order = 1, updated_at = now()
WHERE id = 'aaaa0001-1000-0000-0000-000000000001';

-- (3) 目录恢复正式名
UPDATE menus
SET name = '仪表板', updated_at = now()
WHERE id = 'aaaa0001-0000-0000-0000-000000000001';

-- (4) 菜单的授权角色补一份目录授权(还原 role_menus)
INSERT INTO role_menus (role_id, menu_id, created_by, created_at)
SELECT role_id, 'aaaa0001-0000-0000-0000-000000000001', created_by, now()
FROM role_menus
WHERE menu_id = 'aaaa0001-1000-0000-0000-000000000001'
ON CONFLICT (role_id, menu_id) DO NOTHING;

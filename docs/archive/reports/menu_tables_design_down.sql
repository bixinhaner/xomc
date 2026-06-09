-- ============================================================
-- 回滚菜单表设计
-- ============================================================

-- 删除视图
DROP VIEW IF EXISTS v_role_menus CASCADE;
DROP VIEW IF EXISTS v_menu_tree CASCADE;

-- 删除函数
DROP FUNCTION IF EXISTS get_role_menus(UUID) CASCADE;
DROP FUNCTION IF EXISTS update_menu_timestamp(UUID, UUID) CASCADE;

-- 删除表（注意顺序：先删除依赖表）
DROP TABLE IF EXISTS menu_operation_templates CASCADE;
DROP TABLE IF EXISTS role_menus CASCADE;
DROP TABLE IF EXISTS menus CASCADE;

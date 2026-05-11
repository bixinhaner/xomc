-- 菜单图标功能 seed：
--   1. sys_configs 加 system.show_menu_icon（前端启动期读取，控制 NavMenu 是否显示图标）
--   2. directory / menu 类型菜单 icon 兜底 backfill（已有 icon 的不动；button 不动）
--
-- 设计依据：用户审核后确定的"菜单管理 > 顶部 Switch + 图标库选择"方案。

-- +goose Up

-- ============================================================
-- 1. 全局开关：是否在动态加载菜单时显示菜单图标（默认开启）
--    is_public=true：前端无鉴权即可拉到（与 ui-customization 配置同等级），
--    避免登录前需要这个开关时拿不到。
-- ============================================================
INSERT INTO sys_configs (category, key, value, value_type, description, is_public)
VALUES ('system', 'show_menu_icon', 'true', 'bool', '动态菜单是否显示图标（true=显示，false=隐藏）', TRUE)
ON CONFLICT (category, key) DO NOTHING;

-- ============================================================
-- 2. directory / menu 类型菜单 icon 兜底
--    现状：seed/000057 / 000078 / 000079 已为绝大多数菜单配齐 icon，
--    这里仅作历史遗漏 / 未来手工 INSERT 漏填的兜底，给 AppstoreOutlined。
--    NavMenu.renderIcon() 同样兜底 AppstoreOutlined，前端永远不会渲染裸 icon
--    缺失的菜单——本 backfill 是为了让 DB 数据自洽，便于 UI 编辑时回显默认值。
-- ============================================================
UPDATE menus
SET icon = 'AppstoreOutlined'
WHERE type IN ('directory', 'menu')
  AND (icon IS NULL OR icon = '');

-- +goose Down
DELETE FROM sys_configs WHERE category = 'system' AND key = 'show_menu_icon';
-- icon 兜底不回滚：恢复 NULL 没有业务意义，且会破坏前端默认渲染体验。

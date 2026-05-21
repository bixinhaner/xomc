-- 隐藏告警管理下的“自定义告警”菜单。
--
-- 背景：前端侧边栏在动态菜单模式下读取 /auth/menus，数据来自 menus 表，
-- 即使静态 NAV_CONFIG 已移除该入口，只要 DB 里仍有 status='normal' 的节点，
-- 菜单栏就会继续显示。
--
-- 本迁移做的事：
--   1. 把 /alarm/custom-stats 菜单节点置为 disabled + hide
--   2. 把其下 4 个 button 节点一并置为 disabled + hide，保持权限树一致

-- +goose Up
UPDATE menus
SET status = 'disabled',
    show_status = 'hide',
    updated_at = NOW()
WHERE permission_key = 'alarm:custom-stats'
   OR permission_key LIKE 'alarm:custom-stats:%';

-- +goose Down
UPDATE menus
SET status = 'normal',
    show_status = 'show',
    updated_at = NOW()
WHERE permission_key = 'alarm:custom-stats'
   OR permission_key LIKE 'alarm:custom-stats:%';
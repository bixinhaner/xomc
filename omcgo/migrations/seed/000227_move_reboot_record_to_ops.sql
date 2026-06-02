-- +goose Up
-- 将「重启记录」菜单从「设备管理」移动到「运维管理」下，并改名为「启动记录」。
--
-- 目标菜单：aaaa0007-1000-0000-0000-000000000002
--   · parent_id  11111111-...-111111111101（设备管理）→ aaaa000a-...-000000000001（运维管理）
--   · name       重启记录 → 启动记录
--   · name_i18n   zh-CN/en-US 同步改为「启动记录 / Startup Records」
--
-- 路由（/device/abnormal-reboot）与组件（device/AbnormalReboot）保持不变，仅调整菜单归属与显示名，
-- 故无需改前端路由代码。
--
-- 角色绑定（系统超管 + 两个运营商管理员）按 menu_id 走，移动父级不影响；且这三个角色对
-- 「运维管理」父目录也均已绑定，移动后不会出现孤儿菜单。
--
-- 幂等：按 id 直接赋目标值，重复执行结果一致。
UPDATE menus
   SET parent_id = 'aaaa000a-0000-0000-0000-000000000001',
       name      = '启动记录',
       name_i18n = '{"en-US": "Startup Records", "zh-CN": "启动记录"}'::jsonb,
       updated_at = now()
 WHERE id = 'aaaa0007-1000-0000-0000-000000000002';

-- +goose Down
-- 回滚：归还「设备管理」并恢复原名「重启记录」。
UPDATE menus
   SET parent_id = '11111111-1111-1111-1111-111111111101',
       name      = '重启记录',
       name_i18n = '{"en-US": "Reboot Records", "zh-CN": "重启记录"}'::jsonb,
       updated_at = now()
 WHERE id = 'aaaa0007-1000-0000-0000-000000000002';

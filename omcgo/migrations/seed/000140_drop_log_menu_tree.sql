-- +goose Up
-- +goose StatementBegin

-- 「日志管理」整个一级菜单下线（用户决策：内容全部已迁移到其他菜单或不再需要）
--
-- 涉及节点（参考 000057_refresh_menu_seed.sql）：
--   - aaaa0007-0000-0000-0000-000000000001 「日志管理」directory
--   - aaaa0007-1000-0000-0000-000000000001 「设备上报日志」 → /log/device
--   - aaaa0007-1000-0000-0000-000000000003 「事件日志」     → /log/event
--                                            （已合并到「设备管理 / 重启记录」EventLogTab）
--   - aaaa0007-1000-0000-0000-000000000002 「重启记录」     ← 注意：T-0158 已 reparent
--                                            到「设备管理 / 11111111-...111101」分组下，
--                                            不在 log 分组里，本 seed 不动它。
--
-- DELETE 父 directory 会通过 menus 表 parent_id ON DELETE CASCADE 自动删
-- 剩余子菜单（除已 reparent 出去的「重启记录」），role_menus 也通过 menu_id
-- ON DELETE CASCADE 自动清理关联。

DELETE FROM menus WHERE id = 'aaaa0007-0000-0000-0000-000000000001';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 回滚仅恢复 directory + 两个直属子菜单（按 000057 原值），按钮权限 / 角色绑定
-- 用户需自行重做（CASCADE 删除后无法精确还原）。
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status)
VALUES
  ('aaaa0007-0000-0000-0000-000000000001', '日志管理',     'directory', 'log',         NULL,                                   8, NULL,            'FileTextOutlined', 'normal'),
  ('aaaa0007-1000-0000-0000-000000000001', '设备上报日志', 'menu',      'log:device',  'aaaa0007-0000-0000-0000-000000000001', 1, '/log/device',   'FileTextOutlined', 'normal'),
  ('aaaa0007-1000-0000-0000-000000000003', '事件日志',     'menu',      'log:event',   'aaaa0007-0000-0000-0000-000000000001', 3, '/log/event',    'BellOutlined',     'normal')
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd

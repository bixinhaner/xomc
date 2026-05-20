-- 调整产品中心子菜单顺序：产品管理 下放到 告警库 下面
-- 旧顺序：产品管理 / 参数模型 / KPI 指标库 / 告警库 / 孤儿设备
-- 新顺序：参数模型 / KPI 指标库 / 告警库 / 产品管理 / 孤儿设备
--
-- 仅调整 sort_order，不动 id / permission_key / route_path / role_menus

-- +goose Up
UPDATE menus SET sort_order = 1, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000002'::uuid; -- 参数模型
UPDATE menus SET sort_order = 2, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000003'::uuid; -- KPI 指标库
UPDATE menus SET sort_order = 3, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000004'::uuid; -- 告警库
UPDATE menus SET sort_order = 4, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000001'::uuid; -- 产品管理
UPDATE menus SET sort_order = 5, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000005'::uuid; -- 孤儿设备

-- +goose Down
UPDATE menus SET sort_order = 1, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000001'::uuid;
UPDATE menus SET sort_order = 2, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000002'::uuid;
UPDATE menus SET sort_order = 3, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000003'::uuid;
UPDATE menus SET sort_order = 4, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000004'::uuid;
UPDATE menus SET sort_order = 5, updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000005'::uuid;

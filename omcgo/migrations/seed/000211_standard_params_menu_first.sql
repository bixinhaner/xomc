-- 2026-05-28 用户决策:把"标准参数树"排到产品中心子菜单的**第一位**。
--
-- seed/000210 默认 sort_order=6(产品管理=1 / 参数模型=2 / KPI 指标库=3 /
-- 告警库=4 / 孤儿设备=5 之后)。本迁移把它调到 sort_order=0,自然排到
-- 首位(menus.sort_order ASC),其它项无需移动。

-- +goose Up
UPDATE menus
   SET sort_order = 0,
       updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000006'::uuid;


-- +goose Down
-- 还原 seed/000210 的 sort_order=6
UPDATE menus
   SET sort_order = 6,
       updated_at = NOW()
 WHERE id = 'aaaa0098-1000-0000-0000-000000000006'::uuid;

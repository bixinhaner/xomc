-- +goose Up
-- issue #419：菜单调整（菜单为 DB 驱动，VITE_DYNAMIC_MENU 默认开 → 三皮肤共用本表）。
--  1) 「拓扑管理」排到「仪表板」(0) 之后、「设备管理」之前：topology 5→1，其后顶级菜单整体后移一位。
--  2) 「脚本任务」(mml:script) 由隐藏改为显示。
-- 幂等：UPDATE 到固定值，重跑结果一致；按 permission_key 定位（顶级菜单 key 稳定唯一）。

-- 1) 顶级菜单重排：仪表板(0) → 拓扑管理(1) → 设备管理(2) → 告警(3) → 性能(4) → MML/配置(5) → 文件传输(6)…
UPDATE public.menus SET sort_order = 1 WHERE permission_key = 'topology';
UPDATE public.menus SET sort_order = 2 WHERE permission_key = 'device';
UPDATE public.menus SET sort_order = 3 WHERE permission_key = 'alarm';
UPDATE public.menus SET sort_order = 4 WHERE permission_key = 'performance';
UPDATE public.menus SET sort_order = 5 WHERE permission_key IN ('mml', 'config');

-- 2) 脚本任务显示
UPDATE public.menus SET show_status = 'show' WHERE permission_key = 'mml:script';

-- +goose Down
-- 还原到调整前：topology 1→5、device/alarm/performance/mml/config 各 -1、mml:script 复隐。
UPDATE public.menus SET sort_order = 5 WHERE permission_key = 'topology';
UPDATE public.menus SET sort_order = 1 WHERE permission_key = 'device';
UPDATE public.menus SET sort_order = 2 WHERE permission_key = 'alarm';
UPDATE public.menus SET sort_order = 3 WHERE permission_key = 'performance';
UPDATE public.menus SET sort_order = 4 WHERE permission_key IN ('mml', 'config');

UPDATE public.menus SET show_status = 'hide' WHERE permission_key = 'mml:script';

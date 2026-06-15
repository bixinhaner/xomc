-- +goose Up
-- issue #409：
--  1) 把「MML控制台V2」菜单改名为「MML控制台」（旧 v1 菜单已由 000003 删除）。
--  2) 为 ConsoleV2（/mml/console-v2）新增 3 个按钮级权限（精简集）：
--     execute（执行下发，高危）/ export（导出 CSV）/ history（清空命令记录）。
--  3) role_menus 授权：execute+history → admin+operator；export → admin+operator+viewer。
-- 全部幂等（ON CONFLICT DO NOTHING / 按 id 操作）。

-- 1) 改名
UPDATE public.menus
SET name      = 'MML控制台',
    name_i18n = '{"en-US": "MML Console", "zh-CN": "MML控制台"}'
WHERE id = 'aaaa0003-1000-0000-0000-000000000004';

-- 2) 新增 3 个 button 子节点（parent = V2 菜单）
INSERT INTO public.menus
  (id, name, type, permission_key, parent_id, sort_order, route_path, component_path, icon, show_status, status, created_by, created_at, updated_by, updated_at, name_i18n)
VALUES
  ('aaaa0003-1004-0000-0000-000000000001', '执行',     'button', 'mml:console-v2:execute', 'aaaa0003-1000-0000-0000-000000000004', 1, NULL, NULL, NULL, 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "Execute", "zh-CN": "执行"}'),
  ('aaaa0003-1004-0000-0000-000000000002', '导出',     'button', 'mml:console-v2:export',  'aaaa0003-1000-0000-0000-000000000004', 2, NULL, NULL, NULL, 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "Export", "zh-CN": "导出"}'),
  ('aaaa0003-1004-0000-0000-000000000003', '清空记录', 'button', 'mml:console-v2:history', 'aaaa0003-1000-0000-0000-000000000004', 3, NULL, NULL, NULL, 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "Clear History", "zh-CN": "清空记录"}')
ON CONFLICT (id) DO NOTHING;

-- 3) role_menus 授权（admin=…001 / operator=…002 / viewer=…003）
INSERT INTO public.role_menus (id, role_id, menu_id, created_by, created_at) VALUES
  ('bbbb0409-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'aaaa0003-1004-0000-0000-000000000001', NULL, now()),
  ('bbbb0409-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000002', 'aaaa0003-1004-0000-0000-000000000001', NULL, now()),
  ('bbbb0409-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000001', 'aaaa0003-1004-0000-0000-000000000002', NULL, now()),
  ('bbbb0409-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000002', 'aaaa0003-1004-0000-0000-000000000002', NULL, now()),
  ('bbbb0409-0000-0000-0000-000000000005', '10000000-0000-0000-0000-000000000003', 'aaaa0003-1004-0000-0000-000000000002', NULL, now()),
  ('bbbb0409-0000-0000-0000-000000000006', '10000000-0000-0000-0000-000000000001', 'aaaa0003-1004-0000-0000-000000000003', NULL, now()),
  ('bbbb0409-0000-0000-0000-000000000007', '10000000-0000-0000-0000-000000000002', 'aaaa0003-1004-0000-0000-000000000003', NULL, now())
ON CONFLICT (id) DO NOTHING;

-- +goose Down
-- 先删授权，再删按钮，最后还原菜单名（满足 FK，且不依赖级联是否存在）。
DELETE FROM public.role_menus WHERE id IN (
  'bbbb0409-0000-0000-0000-000000000001',
  'bbbb0409-0000-0000-0000-000000000002',
  'bbbb0409-0000-0000-0000-000000000003',
  'bbbb0409-0000-0000-0000-000000000004',
  'bbbb0409-0000-0000-0000-000000000005',
  'bbbb0409-0000-0000-0000-000000000006',
  'bbbb0409-0000-0000-0000-000000000007'
);

DELETE FROM public.menus WHERE id IN (
  'aaaa0003-1004-0000-0000-000000000001',
  'aaaa0003-1004-0000-0000-000000000002',
  'aaaa0003-1004-0000-0000-000000000003'
);

UPDATE public.menus
SET name      = 'MML控制台V2',
    name_i18n = '{"en-US": "MML Console V2", "zh-CN": "MML控制台V2"}'
WHERE id = 'aaaa0003-1000-0000-0000-000000000004';

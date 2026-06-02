-- +goose Up
-- 系统菜单调整（动态菜单 menus 表，VITE_DYNAMIC_MENU=true 下生效）：
--   1. 「系统管理」目录移至最后：sort_order 9 → 99（当前最大可见顶级目录为「产品中心」=12）。
--   2. 放出「运维管理」目录 + 「TR069报文跟踪」菜单：show_status hide → show。
--      其余运维子菜单（运维任务/命令/网络诊断/下载/PM 聚合手动触发）保持 hide。
--   3. 为「TR069报文跟踪」页面按功能新增 5 个 button 权限点（查看/新建/停止/导出/删除）。
--   4. 把上述新增/放出的菜单绑定到 admin 角色（admin 登录走 role_menus，非超管旁路）。
--
-- 幂等：UPDATE 带状态条件；菜单 INSERT 走 ON CONFLICT(id) DO NOTHING；
--       role_menus INSERT 走 NOT EXISTS 守门。重复执行为 no-op。
--
-- 关键 ID：
--   系统管理目录       11111111-1111-1111-1111-111111111108
--   运维管理目录       aaaa000a-0000-0000-0000-000000000001
--   TR069报文跟踪菜单  aaaa000a-1000-0000-0000-000000000006
--   admin 角色         10000000-0000-0000-0000-000000000001

-- 1. 系统管理放最后
UPDATE public.menus
   SET sort_order = 99, updated_at = now()
 WHERE id = '11111111-1111-1111-1111-111111111108'
   AND sort_order <> 99;

-- 2. 放出运维管理目录 + TR069报文跟踪菜单
UPDATE public.menus
   SET show_status = 'show', updated_at = now()
 WHERE id IN (
         'aaaa000a-0000-0000-0000-000000000001',  -- 运维管理目录
         'aaaa000a-1000-0000-0000-000000000006'   -- TR069报文跟踪
       )
   AND show_status = 'hide';

-- 3. TR069报文跟踪 按钮权限点（parent = 报文跟踪菜单）
INSERT INTO public.menus
  (id, name, type, permission_key, parent_id, sort_order, route_path, component_path, icon, show_status, status, created_by, created_at, updated_by, updated_at, name_i18n)
VALUES
  ('aaaa000a-1a00-0000-0000-000000000001', '查看',   'button', 'ops:message-trace:query',  'aaaa000a-1000-0000-0000-000000000006', 1, '', NULL, NULL, 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "View", "zh-CN": "查看"}'),
  ('aaaa000a-1a00-0000-0000-000000000002', '新建跟踪', 'button', 'ops:message-trace:create', 'aaaa000a-1000-0000-0000-000000000006', 2, '', NULL, NULL, 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "Create Trace", "zh-CN": "新建跟踪"}'),
  ('aaaa000a-1a00-0000-0000-000000000003', '停止',   'button', 'ops:message-trace:stop',   'aaaa000a-1000-0000-0000-000000000006', 3, '', NULL, NULL, 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "Stop", "zh-CN": "停止"}'),
  ('aaaa000a-1a00-0000-0000-000000000004', '导出',   'button', 'ops:message-trace:export', 'aaaa000a-1000-0000-0000-000000000006', 4, '', NULL, NULL, 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "Export", "zh-CN": "导出"}'),
  ('aaaa000a-1a00-0000-0000-000000000005', '删除',   'button', 'ops:message-trace:delete', 'aaaa000a-1000-0000-0000-000000000006', 5, '', NULL, NULL, 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "Delete", "zh-CN": "删除"}')
ON CONFLICT (id) DO NOTHING;

-- 4. 绑定到 admin 角色（目录 + 菜单 + 5 个按钮）
INSERT INTO public.role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, m.menu_id
FROM (VALUES
        ('aaaa000a-0000-0000-0000-000000000001'::uuid),
        ('aaaa000a-1000-0000-0000-000000000006'::uuid),
        ('aaaa000a-1a00-0000-0000-000000000001'::uuid),
        ('aaaa000a-1a00-0000-0000-000000000002'::uuid),
        ('aaaa000a-1a00-0000-0000-000000000003'::uuid),
        ('aaaa000a-1a00-0000-0000-000000000004'::uuid),
        ('aaaa000a-1a00-0000-0000-000000000005'::uuid)
     ) AS m(menu_id)
WHERE NOT EXISTS (
        SELECT 1 FROM public.role_menus rm
         WHERE rm.role_id = '10000000-0000-0000-0000-000000000001'
           AND rm.menu_id = m.menu_id
      );

-- +goose Down
-- 回滚：删绑定 → 删按钮 → 恢复隐藏 → 系统管理 sort_order 还原 9。
DELETE FROM public.role_menus
 WHERE role_id = '10000000-0000-0000-0000-000000000001'
   AND menu_id IN (
         'aaaa000a-0000-0000-0000-000000000001',
         'aaaa000a-1000-0000-0000-000000000006',
         'aaaa000a-1a00-0000-0000-000000000001',
         'aaaa000a-1a00-0000-0000-000000000002',
         'aaaa000a-1a00-0000-0000-000000000003',
         'aaaa000a-1a00-0000-0000-000000000004',
         'aaaa000a-1a00-0000-0000-000000000005'
       );

DELETE FROM public.menus
 WHERE id IN (
         'aaaa000a-1a00-0000-0000-000000000001',
         'aaaa000a-1a00-0000-0000-000000000002',
         'aaaa000a-1a00-0000-0000-000000000003',
         'aaaa000a-1a00-0000-0000-000000000004',
         'aaaa000a-1a00-0000-0000-000000000005'
       );

UPDATE public.menus
   SET show_status = 'hide', updated_at = now()
 WHERE id IN (
         'aaaa000a-0000-0000-0000-000000000001',
         'aaaa000a-1000-0000-0000-000000000006'
       );

UPDATE public.menus
   SET sort_order = 9, updated_at = now()
 WHERE id = '11111111-1111-1111-1111-111111111108';

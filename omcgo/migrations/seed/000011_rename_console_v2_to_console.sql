-- +goose Up
-- issue #475：MML 控制台彻底去「V2」后缀（旧 v1 console 已由 000003 删除，V2 现为唯一控制台）。
--   父菜单（id=aaaa0003-1000-…-004）：
--     component_path  mml/ConsoleV2  → mml/Console
--     route_path      /mml/console-v2 → /mml/console
--     permission_key  mml:console-v2  → mml:console
--   3 个按钮权限（按 id）：
--     mml:console-v2:{execute,export,history} → mml:console:{execute,export,history}
-- role_menus 按 menu_id 授权，不依赖 permission_key → 本次仅改 perm_key，已有授权自动保留。
-- 旧名 mml:console / /mml/console / mml/Console 已空出（000003 删干净），无冲突。
-- 全部按 id UPDATE，幂等。前端 import/路由/usePermission 同 PR 同步改。

UPDATE public.menus
SET component_path = 'mml/Console',
    route_path     = '/mml/console',
    permission_key = 'mml:console',
    updated_at     = now()
WHERE id = 'aaaa0003-1000-0000-0000-000000000004';

UPDATE public.menus SET permission_key = 'mml:console:execute', updated_at = now()
WHERE id = 'aaaa0003-1004-0000-0000-000000000001';
UPDATE public.menus SET permission_key = 'mml:console:export',  updated_at = now()
WHERE id = 'aaaa0003-1004-0000-0000-000000000002';
UPDATE public.menus SET permission_key = 'mml:console:history', updated_at = now()
WHERE id = 'aaaa0003-1004-0000-0000-000000000003';

-- +goose Down
-- 还原为 V2 命名（与 000004 之后、本迁移之前的状态一致）。
UPDATE public.menus
SET component_path = 'mml/ConsoleV2',
    route_path     = '/mml/console-v2',
    permission_key = 'mml:console-v2',
    updated_at     = now()
WHERE id = 'aaaa0003-1000-0000-0000-000000000004';

UPDATE public.menus SET permission_key = 'mml:console-v2:execute', updated_at = now()
WHERE id = 'aaaa0003-1004-0000-0000-000000000001';
UPDATE public.menus SET permission_key = 'mml:console-v2:export',  updated_at = now()
WHERE id = 'aaaa0003-1004-0000-0000-000000000002';
UPDATE public.menus SET permission_key = 'mml:console-v2:history', updated_at = now()
WHERE id = 'aaaa0003-1004-0000-0000-000000000003';

-- +goose Up
-- 为 29 个尚无操作按钮的菜单补全标准 4 按钮（查询/添加/修改/删除）。
-- 命名规范：name='查询'/'添加'/'修改'/'删除'，permission_key=<父 menu key>:<action>。
-- 幂等：用 WHERE NOT EXISTS 跳过已存在按钮（按 parent_id + name 唯一）。
-- 需要更细的按钮（如 导出 / 同步 / 强制下线 等）由 admin 在 /system/menus 页面手工追加。

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111106', 'button', '查询', 'alarm:current:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111106' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111106', 'button', '添加', 'alarm:current:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111106' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111106', 'button', '修改', 'alarm:current:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111106' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111106', 'button', '删除', 'alarm:current:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111106' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0011-1000-0000-0000-000000000003', 'button', '查询', 'alarm:custom-stats:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0011-1000-0000-0000-000000000003' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0011-1000-0000-0000-000000000003', 'button', '添加', 'alarm:custom-stats:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0011-1000-0000-0000-000000000003' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0011-1000-0000-0000-000000000003', 'button', '修改', 'alarm:custom-stats:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0011-1000-0000-0000-000000000003' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0011-1000-0000-0000-000000000003', 'button', '删除', 'alarm:custom-stats:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0011-1000-0000-0000-000000000003' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111107', 'button', '查询', 'alarm:history:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111107' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111107', 'button', '添加', 'alarm:history:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111107' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111107', 'button', '修改', 'alarm:history:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111107' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111107', 'button', '删除', 'alarm:history:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111107' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0011-1000-0000-0000-000000000002', 'button', '查询', 'alarm:library:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0011-1000-0000-0000-000000000002' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0011-1000-0000-0000-000000000002', 'button', '添加', 'alarm:library:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0011-1000-0000-0000-000000000002' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0011-1000-0000-0000-000000000002', 'button', '修改', 'alarm:library:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0011-1000-0000-0000-000000000002' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0011-1000-0000-0000-000000000002', 'button', '删除', 'alarm:library:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0011-1000-0000-0000-000000000002' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000003', 'button', '查询', 'backup:restore:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000003' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000003', 'button', '添加', 'backup:restore:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000003' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000003', 'button', '修改', 'backup:restore:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000003' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000003', 'button', '删除', 'backup:restore:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000003' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000002', 'button', '查询', 'backup:schedule:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000002' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000002', 'button', '添加', 'backup:schedule:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000002' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000002', 'button', '修改', 'backup:schedule:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000002' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000002', 'button', '删除', 'backup:schedule:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000002' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000001', 'button', '查询', 'backup:tasks:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000001', 'button', '添加', 'backup:tasks:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000001', 'button', '修改', 'backup:tasks:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0005-1000-0000-0000-000000000001', 'button', '删除', 'backup:tasks:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0005-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0001-1000-0000-0000-000000000001', 'button', '查询', 'dashboard:home:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0001-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0001-1000-0000-0000-000000000001', 'button', '添加', 'dashboard:home:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0001-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0001-1000-0000-0000-000000000001', 'button', '修改', 'dashboard:home:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0001-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0001-1000-0000-0000-000000000001', 'button', '删除', 'dashboard:home:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0001-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111103', 'button', '查询', 'device:group:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111103' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111103', 'button', '添加', 'device:group:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111103' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111103', 'button', '修改', 'device:group:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111103' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111103', 'button', '删除', 'device:group:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111103' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000001', 'button', '查询', 'device:plug-and-play:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000001', 'button', '添加', 'device:plug-and-play:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000001', 'button', '修改', 'device:plug-and-play:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000001', 'button', '删除', 'device:plug-and-play:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000003', 'button', '查询', 'device:recycle:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000003' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000003', 'button', '添加', 'device:recycle:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000003' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000003', 'button', '修改', 'device:recycle:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000003' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000003', 'button', '删除', 'device:recycle:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000003' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111104', 'button', '查询', 'device:register:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111104' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111104', 'button', '添加', 'device:register:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111104' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111104', 'button', '修改', 'device:register:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111104' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111104', 'button', '删除', 'device:register:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111104' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000002', 'button', '查询', 'device:rules:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000002' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000002', 'button', '添加', 'device:rules:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000002' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000002', 'button', '修改', 'device:rules:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000002' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0010-1000-0000-0000-000000000002', 'button', '删除', 'device:rules:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0010-1000-0000-0000-000000000002' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000001', 'button', '查询', 'log:device:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000001', 'button', '添加', 'log:device:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000001', 'button', '修改', 'log:device:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000001', 'button', '删除', 'log:device:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000003', 'button', '查询', 'log:event:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000003' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000003', 'button', '添加', 'log:event:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000003' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000003', 'button', '修改', 'log:event:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000003' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000003', 'button', '删除', 'log:event:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000003' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000002', 'button', '查询', 'log:exception:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000002', 'button', '添加', 'log:exception:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000002', 'button', '修改', 'log:exception:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0007-1000-0000-0000-000000000002', 'button', '删除', 'log:exception:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0007-1000-0000-0000-000000000002' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000001', 'button', '查询', 'mml:console:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000001', 'button', '添加', 'mml:console:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000001', 'button', '修改', 'mml:console:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000001', 'button', '删除', 'mml:console:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000002', 'button', '查询', 'mml:script:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000002' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000002', 'button', '添加', 'mml:script:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000002' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000002', 'button', '修改', 'mml:script:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000002' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000002', 'button', '删除', 'mml:script:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000002' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000003', 'button', '查询', 'mml:task-records:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000003' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000003', 'button', '添加', 'mml:task-records:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000003' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000003', 'button', '修改', 'mml:task-records:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000003' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0003-1000-0000-0000-000000000003', 'button', '删除', 'mml:task-records:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0003-1000-0000-0000-000000000003' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000003', 'button', '查询', 'performance:kpi-standard:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000003' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000003', 'button', '添加', 'performance:kpi-standard:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000003' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000003', 'button', '修改', 'performance:kpi-standard:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000003' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000003', 'button', '删除', 'performance:kpi-standard:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000003' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000002', 'button', '查询', 'performance:kpi-station:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000002' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000002', 'button', '添加', 'performance:kpi-station:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000002' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000002', 'button', '修改', 'performance:kpi-station:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000002' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000002', 'button', '删除', 'performance:kpi-station:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000002' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000001', 'button', '查询', 'performance:query:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000001', 'button', '添加', 'performance:query:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000001', 'button', '修改', 'performance:query:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0002-1000-0000-0000-000000000001', 'button', '删除', 'performance:query:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0002-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000002', 'button', '查询', 'software:firmware:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000002' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000002', 'button', '添加', 'software:firmware:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000002' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000002', 'button', '修改', 'software:firmware:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000002' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000002', 'button', '删除', 'software:firmware:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000002' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000003', 'button', '查询', 'software:rollback:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000003' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000003', 'button', '添加', 'software:rollback:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000003' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000003', 'button', '修改', 'software:rollback:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000003' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000003', 'button', '删除', 'software:rollback:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000003' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000001', 'button', '查询', 'software:upgrade-plan:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000001', 'button', '添加', 'software:upgrade-plan:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000001', 'button', '修改', 'software:upgrade-plan:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0006-1000-0000-0000-000000000001', 'button', '删除', 'software:upgrade-plan:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0006-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0008-1000-0000-0000-000000000001', 'button', '查询', 'system:config:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0008-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0008-1000-0000-0000-000000000001', 'button', '添加', 'system:config:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0008-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0008-1000-0000-0000-000000000001', 'button', '修改', 'system:config:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0008-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0008-1000-0000-0000-000000000001', 'button', '删除', 'system:config:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0008-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111112', 'button', '查询', 'system:operation-log:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111112' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111112', 'button', '添加', 'system:operation-log:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111112' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111112', 'button', '修改', 'system:operation-log:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111112' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '11111111-1111-1111-1111-111111111112', 'button', '删除', 'system:operation-log:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '11111111-1111-1111-1111-111111111112' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0008-1000-0000-0000-000000000002', 'button', '查询', 'system:ui-custom:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0008-1000-0000-0000-000000000002' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0008-1000-0000-0000-000000000002', 'button', '添加', 'system:ui-custom:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0008-1000-0000-0000-000000000002' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0008-1000-0000-0000-000000000002', 'button', '修改', 'system:ui-custom:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0008-1000-0000-0000-000000000002' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0008-1000-0000-0000-000000000002', 'button', '删除', 'system:ui-custom:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0008-1000-0000-0000-000000000002' AND name = '删除' AND status = 'normal'
);

INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0004-1000-0000-0000-000000000001', 'button', '查询', 'topology:canvas:query', 1, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0004-1000-0000-0000-000000000001' AND name = '查询' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0004-1000-0000-0000-000000000001', 'button', '添加', 'topology:canvas:add', 2, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0004-1000-0000-0000-000000000001' AND name = '添加' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0004-1000-0000-0000-000000000001', 'button', '修改', 'topology:canvas:edit', 3, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0004-1000-0000-0000-000000000001' AND name = '修改' AND status = 'normal'
);
INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT 'aaaa0004-1000-0000-0000-000000000001', 'button', '删除', 'topology:canvas:delete', 4, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = 'aaaa0004-1000-0000-0000-000000000001' AND name = '删除' AND status = 'normal'
);

-- +goose Down
-- 回滚：删除本次脚本插入的 4 类按钮（按 permission_key 后缀精确匹配，避免误删手工按钮）。
DELETE FROM menus WHERE type = 'button' AND permission_key IN (
    'alarm:current:query',
    'alarm:current:add',
    'alarm:current:edit',
    'alarm:current:delete',
    'alarm:custom-stats:query',
    'alarm:custom-stats:add',
    'alarm:custom-stats:edit',
    'alarm:custom-stats:delete',
    'alarm:history:query',
    'alarm:history:add',
    'alarm:history:edit',
    'alarm:history:delete',
    'alarm:library:query',
    'alarm:library:add',
    'alarm:library:edit',
    'alarm:library:delete',
    'backup:restore:query',
    'backup:restore:add',
    'backup:restore:edit',
    'backup:restore:delete',
    'backup:schedule:query',
    'backup:schedule:add',
    'backup:schedule:edit',
    'backup:schedule:delete',
    'backup:tasks:query',
    'backup:tasks:add',
    'backup:tasks:edit',
    'backup:tasks:delete',
    'dashboard:home:query',
    'dashboard:home:add',
    'dashboard:home:edit',
    'dashboard:home:delete',
    'device:group:query',
    'device:group:add',
    'device:group:edit',
    'device:group:delete',
    'device:plug-and-play:query',
    'device:plug-and-play:add',
    'device:plug-and-play:edit',
    'device:plug-and-play:delete',
    'device:recycle:query',
    'device:recycle:add',
    'device:recycle:edit',
    'device:recycle:delete',
    'device:register:query',
    'device:register:add',
    'device:register:edit',
    'device:register:delete',
    'device:rules:query',
    'device:rules:add',
    'device:rules:edit',
    'device:rules:delete',
    'log:device:query',
    'log:device:add',
    'log:device:edit',
    'log:device:delete',
    'log:event:query',
    'log:event:add',
    'log:event:edit',
    'log:event:delete',
    'log:exception:query',
    'log:exception:add',
    'log:exception:edit',
    'log:exception:delete',
    'mml:console:query',
    'mml:console:add',
    'mml:console:edit',
    'mml:console:delete',
    'mml:script:query',
    'mml:script:add',
    'mml:script:edit',
    'mml:script:delete',
    'mml:task-records:query',
    'mml:task-records:add',
    'mml:task-records:edit',
    'mml:task-records:delete',
    'performance:kpi-standard:query',
    'performance:kpi-standard:add',
    'performance:kpi-standard:edit',
    'performance:kpi-standard:delete',
    'performance:kpi-station:query',
    'performance:kpi-station:add',
    'performance:kpi-station:edit',
    'performance:kpi-station:delete',
    'performance:query:query',
    'performance:query:add',
    'performance:query:edit',
    'performance:query:delete',
    'software:firmware:query',
    'software:firmware:add',
    'software:firmware:edit',
    'software:firmware:delete',
    'software:rollback:query',
    'software:rollback:add',
    'software:rollback:edit',
    'software:rollback:delete',
    'software:upgrade-plan:query',
    'software:upgrade-plan:add',
    'software:upgrade-plan:edit',
    'software:upgrade-plan:delete',
    'system:config:query',
    'system:config:add',
    'system:config:edit',
    'system:config:delete',
    'system:operation-log:query',
    'system:operation-log:add',
    'system:operation-log:edit',
    'system:operation-log:delete',
    'system:ui-custom:query',
    'system:ui-custom:add',
    'system:ui-custom:edit',
    'system:ui-custom:delete',
    'topology:canvas:query',
    'topology:canvas:add',
    'topology:canvas:edit',
    'topology:canvas:delete'
);

-- F06 System License 重构 P1 Step 5.B — 清理老 multi-license 模型的菜单 / 权限种子。
--
-- PRD: docs/project/prd/F06-system-license-redesign.md §8.1 §10.2
--
-- 旧种子（要清掉）：
--   seed/000075 — license:operate button 节点（孤儿）
--   seed/000077 — /api/v1/licenses/:id/export + /api/v1/licenses/export 端点 + 绑定
--   seed/000078 — 一级目录 license + 3 个 page menus (list/operations/logs) + 3 个 button
--
-- 本迁移做的事：
--   1. DELETE 上述 7 个 menus 节点（CASCADE 自然带走 role_menus 绑定）
--   2. DELETE /api/v1/licenses/* 系列 api_endpoints + role_api_permissions
--   3. INSERT 新 system_license 一级菜单 + 新端点 + 默认绑定（super_admin/operator/viewer）
--
-- 幂等：所有 DELETE 用 IN (...) 列表，重复执行不报错；INSERT 用 ON CONFLICT DO NOTHING。

-- +goose Up

-- ============================================================
-- 1. 清理老 license 菜单（含 role_menus 绑定，FK ON DELETE CASCADE）
-- ============================================================
DELETE FROM menus WHERE id IN (
    'aaaa0009-0000-0000-0000-000000000001'::uuid,   -- 一级目录 /license（老）
    'aaaa0009-1000-0000-0000-000000000001'::uuid,   -- /license/list
    'aaaa0009-1000-0000-0000-000000000002'::uuid,   -- /license/operations
    'aaaa0009-1000-0000-0000-000000000003'::uuid,   -- /license/logs
    'aaaa0009-1100-0000-0000-000000000001'::uuid,   -- button system:license:view
    'aaaa0009-1100-0000-0000-000000000002'::uuid,   -- button system:license:operate
    'aaaa0009-1100-0000-0000-000000000003'::uuid    -- button system:license:audit
);

-- ============================================================
-- 2. 清理老 /api/v1/licenses/* api_endpoints + role 绑定
-- ============================================================
DELETE FROM role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM api_endpoints WHERE path LIKE '/api/v1/licenses%' OR path LIKE '/api/v1/licenses/%'
);
DELETE FROM api_endpoints WHERE path LIKE '/api/v1/licenses%' OR path LIKE '/api/v1/licenses/%';

-- ============================================================
-- 3. 新 system_license 菜单（一级目录直接对应单页 /license）
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, show_status)
VALUES
    (
        'aaaa0009-0000-0000-0000-000000000010'::uuid,
        'License',
        'menu',
        'system_license',
        NULL,
        10,
        '/license',
        'SafetyOutlined',
        'normal',
        'show'
    )
ON CONFLICT (id) DO UPDATE SET
    name         = EXCLUDED.name,
    permission_key = EXCLUDED.permission_key,
    route_path   = EXCLUDED.route_path,
    icon         = EXCLUDED.icon;

-- ============================================================
-- 4. 新 /api/v1/system-license/* api_endpoints
-- ============================================================
INSERT INTO api_endpoints (id, path, method, name, description, module, deprecated)
VALUES
    ('50000000-0001-0000-0000-000000000010'::uuid, '/api/v1/system-license',         'GET',  'GET /api/v1/system-license',         '系统级 License - 获取当前生效 license', 'license', FALSE),
    ('50000000-0001-0000-0000-000000000011'::uuid, '/api/v1/system-license',         'POST', 'POST /api/v1/system-license',        '系统级 License - 上传新文件覆盖当前', 'license', FALSE),
    ('50000000-0001-0000-0000-000000000012'::uuid, '/api/v1/system-license/history', 'GET',  'GET /api/v1/system-license/history', '系统级 License - 历史分页',           'license', FALSE)
ON CONFLICT (path, method) DO NOTHING;

-- ============================================================
-- 5. 默认 role 绑定：super_admin / operator / viewer
--    - super_admin：3 个端点全部
--    - operator   ：GET（看）+ POST（更新）
--    - viewer     ：GET 系列只读
--    （与老 seed/000067 的"viewer 默认 grant 全部 GET"约定一致，本 seed 走显式 INSERT 兜底）
-- ============================================================
INSERT INTO role_api_permissions (role_id, endpoint_id)
SELECT r.id, e.id
FROM roles r
CROSS JOIN api_endpoints e
WHERE r.code IN ('super_admin', 'operator', 'viewer')
  AND e.path IN ('/api/v1/system-license', '/api/v1/system-license/history')
  AND NOT (
      r.code = 'viewer' AND e.method = 'POST'
  )
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down

-- Down: 仅恢复 system_license 菜单 / 端点（保留 super_admin 绑定），老菜单不重建。
-- 真正需要回退到 P4 状态时请 goose down 到 000118 之前（seed 列表里 000078 等会重新执行）。
DELETE FROM role_api_permissions
WHERE endpoint_id IN (
    SELECT id FROM api_endpoints WHERE path LIKE '/api/v1/system-license%'
);
DELETE FROM api_endpoints WHERE path LIKE '/api/v1/system-license%';
DELETE FROM menus WHERE id = 'aaaa0009-0000-0000-0000-000000000010'::uuid;

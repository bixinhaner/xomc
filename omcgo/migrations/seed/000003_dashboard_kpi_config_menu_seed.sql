-- +goose Up
-- issue #213 S3：系统管理下新增「首页 KPI 配置」菜单（管理员可见可编辑）。
--
-- 运行态前端走动态菜单（VITE_DYNAMIC_MENU=true）：菜单来自 menus 表而非前端 navConfig.ts，
-- 故新页必须在此灌一行 menus，否则即便 super_admin 也看不到入口。
--
-- 归属系统管理目录（parent_id = 系统管理目录 11111111-...-108，与字典管理等同级）。
-- type=menu / route_path=/system/kpi-config / component_path=system/KpiConfig（与前端 lazy import 对齐）。
-- show_status=show / status=normal，super_admin（builtIn admin）登录后返回全部 active 菜单即可见。
-- ON CONFLICT (id) DO NOTHING：全新库前向重复应用安全（行数稳定，不报主键冲突）。
INSERT INTO public.menus
    (id, name, type, permission_key, parent_id, sort_order, route_path, component_path, icon, show_status, status, created_by, created_at, updated_by, updated_at, name_i18n)
VALUES
    ('aaaa0008-1000-0000-0000-000000000009', '首页 KPI 配置', 'menu', 'system:kpi-config', '11111111-1111-1111-1111-111111111108', 20, '/system/kpi-config', 'system/KpiConfig', 'LineChartOutlined', 'show', 'normal', NULL, now(), NULL, now(), '{"en-US": "Dashboard KPI Config", "zh-CN": "首页 KPI 配置"}')
ON CONFLICT (id) DO NOTHING;

-- 绑定到 admin 角色（10000000-...-001），使非超管的普通 admin 也能看到该菜单。
-- super_admin（builtIn）走旁路返回全部 active 菜单，不依赖此绑定。
INSERT INTO public.role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001', m.id
FROM public.menus m
WHERE m.id = 'aaaa0008-1000-0000-0000-000000000009'
ON CONFLICT DO NOTHING;

-- +goose Down
-- consolidated seed 风格：baseline 内置参考数据无回滚（建表回滚由 schema Down 处理）。
SELECT 1;

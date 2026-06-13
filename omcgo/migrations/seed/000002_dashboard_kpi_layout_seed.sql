-- +goose Up
-- issue #213 S1：Dashboard KPI 全局布局初始三行（lte/nr/gsm）。
--
-- 把前端写死的三制式布局（omcmb/webcode/src/pages/dashboard/kpi-config.ts：
-- LTE 6 panel / NR 2 panel / GSM 3 panel）灌成 dashboard_kpi_layouts 初始行，
-- 上线当天首页零变化（首页改读本表后回退/读取均得现状布局）。
--
-- 网格：12 列。半宽图 w=6 / 满宽图 w=12，统一 h=8。
-- 各图 metrics 为该 Panel 的全部 symbolic key（与 kpi-config.ts 的 indicators.key 对齐）；
-- title 用 Panel i18n key（与前端 PANEL_LABELS 一致），前端按 key 取本地化文案。
-- ON CONFLICT DO NOTHING：全新库重复前向应用安全（行数稳定，不报主键冲突）。

INSERT INTO public.dashboard_kpi_layouts (tech, layout) VALUES
    ('lte', '{
      "panels": [
        {"title": "dashboard.panel.traffic",        "metrics": ["LTE_PDCP_VOLUME_DL", "LTE_PDCP_VOLUME_UL", "LTE_PDCP_RATE_DL", "LTE_PDCP_RATE_UL"], "x": 0, "y": 0,  "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.availability",   "metrics": ["LTE_CELL_AVAILABLE"], "x": 6, "y": 0,  "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.utilization",    "metrics": ["LTE_PRB_UTIL_DL", "LTE_PRB_UTIL_UL"], "x": 0, "y": 8,  "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.accessibility",  "metrics": ["WIRELESS_SETUP_SR", "RRC_CONN_SETUP_SR", "ERAB_SETUP_SR", "CSFB_SR"], "x": 6, "y": 8,  "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.retainability",  "metrics": ["ERAB_DROP_RATE"], "x": 0, "y": 16, "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.mobility",       "metrics": ["HO_INTRA_ENB_OUT_SR", "HO_INTRA_ENB_IN_SR", "HO_INTER_ENB_OUT_SR", "HO_INTER_ENB_IN_SR"], "x": 6, "y": 16, "w": 6, "h": 8, "chartType": "line"}
      ]
    }'::jsonb),
    ('nr', '{
      "panels": [
        {"title": "dashboard.panel.traffic",     "metrics": ["NR_PDCP_VOLUME_DL", "NR_PDCP_VOLUME_UL", "NR_PDCP_RATE_DL", "NR_PDCP_RATE_UL"], "x": 0, "y": 0, "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.utilization", "metrics": ["NR_PRB_UTIL_DL", "NR_PRB_UTIL_UL"], "x": 6, "y": 0, "w": 6, "h": 8, "chartType": "line"}
      ]
    }'::jsonb),
    ('gsm', '{
      "panels": [
        {"title": "dashboard.panel.accessibility", "metrics": ["GSM_CALL_SETUP_SR"], "x": 0, "y": 0, "w": 6,  "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.retainability", "metrics": ["GSM_CALL_DROP_RATE"], "x": 6, "y": 0, "w": 6,  "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.mobility",      "metrics": ["GSM_HO_SR"], "x": 0, "y": 8, "w": 12, "h": 8, "chartType": "line"}
      ]
    }'::jsonb)
ON CONFLICT DO NOTHING;

-- issue #213 S1：读/存全局布局两接口的端点级权限点（仿 dashboard/widgets GET/PUT 写法）。
-- 端点启动期会被 SyncApiEndpoints 自动 upsert，但 seed 在 migrate 期执行（早于启动同步），
-- 须先有确定的 api_endpoints 行。读接口（GET）所有登录用户可读 —— 授权三内置角色；
-- 存接口（PUT）仅管理员 —— 授权 admin 角色（与配置页菜单/路由守卫一致：菜单绑 admin 角色、
-- 页面 withAdminRole 守卫，故 admin 角色应能看也能存；operator/viewer 不授、看不到也存不了；
-- super_admin builtIn 旁路 casbin 始终可达）。设计 §3「operator/viewer 看不到菜单也调不动存盘接口」
-- 的言外即 admin 层可配——故此处授 admin（区别于 dashboard/widgets 那个 per-user PUT 的 super_admin-only 策略）。
INSERT INTO public.api_endpoints (id, path, method, name, api_group, is_auto)
VALUES
    (gen_random_uuid(), '/api/v1/dashboard/kpi-layout', 'GET', 'GET /api/v1/dashboard/kpi-layout', 'dashboard', true),
    (gen_random_uuid(), '/api/v1/dashboard/kpi-layout', 'PUT', 'PUT /api/v1/dashboard/kpi-layout', 'dashboard', true)
ON CONFLICT (path, method) DO NOTHING;

-- 读接口授权 admin / operator / viewer（所有登录用户可读）。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT r.id, ae.id
FROM public.roles r
JOIN public.api_endpoints ae
    ON ae.path = '/api/v1/dashboard/kpi-layout' AND ae.method = 'GET'
WHERE r.name IN ('admin', 'operator', 'viewer')
ON CONFLICT DO NOTHING;

-- 存接口授权 admin（仅管理员可写全局布局；operator/viewer 不授，super_admin 旁路始终可达）。
-- 与 000003 菜单绑 admin 角色 + 页面 withAdminRole 守卫保持一致，使普通 admin 用户能看也能存。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT r.id, ae.id
FROM public.roles r
JOIN public.api_endpoints ae
    ON ae.path = '/api/v1/dashboard/kpi-layout' AND ae.method = 'PUT'
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- +goose Down
-- consolidated seed 风格：baseline 内置参考数据无回滚（建表回滚由 schema Down 处理）。
SELECT 1;

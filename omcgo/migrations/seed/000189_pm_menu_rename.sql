-- +goose Up
-- ============================================================
-- 000189_T0173_pm_menu_rename
-- ----------------------------------------------------------------
-- T-0173 阶段 1：性能管理菜单命名重构 + 顺序调整
--
-- 旧菜单（来自 seed/000057 + seed/000188）：
--   sort=0: 性能查看        (aaaa0002-1000-...-010)
--   sort=1: 性能查询        (aaaa0002-1000-...-001)
--   sort=2: 基站KPI         (aaaa0002-1000-...-002)
--   sort=3: 标准KPI         (aaaa0002-1000-...-003)
--   sort=4: 自定义聚合      (aaaa0002-1000-...-011)
--
-- 新菜单（消费类在前，配置类在后，行业通用术语）：
--   sort=0: 性能仪表盘      (Performance Dashboard)
--   sort=1: 指标查询        (Metric Query)
--   sort=2: 自定义聚合      (Custom Aggregation)              ← 保留中文，英文统一改 Custom Aggregation
--   sort=3: 指标库          (Indicator Library)
--   sort=4: 测量任务管理    (Measurement Task Management)
--
-- 路由 path 保持不变（路由不动只改文案）。
-- permission_key 保持不变（RBAC 配置零影响）。
-- ============================================================

-- 性能仪表盘（原"性能查看"）
UPDATE menus SET
    name = '性能仪表盘',
    name_i18n = '{"zh-CN":"性能仪表盘","en-US":"Performance Dashboard"}'::jsonb,
    sort_order = 0,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000010'::uuid;

-- 指标查询（原"性能查询"）
UPDATE menus SET
    name = '指标查询',
    name_i18n = '{"zh-CN":"指标查询","en-US":"Metric Query"}'::jsonb,
    sort_order = 1,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000001'::uuid;

-- 自定义聚合（保留名称，仅前移 sort_order；英文统一为 Custom Aggregation）
UPDATE menus SET
    name = '自定义聚合',
    name_i18n = '{"zh-CN":"自定义聚合","en-US":"Custom Aggregation"}'::jsonb,
    sort_order = 2,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000011'::uuid;

-- 指标库（原"标准KPI"）
UPDATE menus SET
    name = '指标库',
    name_i18n = '{"zh-CN":"指标库","en-US":"Indicator Library"}'::jsonb,
    sort_order = 3,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000003'::uuid;

-- 测量任务管理（原"基站KPI"）
UPDATE menus SET
    name = '测量任务管理',
    name_i18n = '{"zh-CN":"测量任务管理","en-US":"Measurement Task Management"}'::jsonb,
    sort_order = 4,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000002'::uuid;


-- +goose Down
-- ============================================================
-- 000189_T0173_pm_menu_rename (Down)
-- ----------------------------------------------------------------
-- 还原到 seed/000057 + seed/000188 落地后的状态。
-- 注意：三条老菜单（001/002/003）的 name_i18n 在 000083 之后从未填过，
--      回滚时设回 NULL；新挂的两条（010/011）的 name_i18n 在 000188 已填，
--      回滚时还原成 000188 的初始值。
-- ============================================================

-- 性能查看
UPDATE menus SET
    name = '性能查看',
    name_i18n = '{"zh-CN":"性能查看","en-US":"PM Dashboard"}'::jsonb,
    sort_order = 0,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000010'::uuid;

-- 性能查询
UPDATE menus SET
    name = '性能查询',
    name_i18n = NULL,
    sort_order = 1,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000001'::uuid;

-- 自定义聚合
UPDATE menus SET
    name = '自定义聚合',
    name_i18n = '{"zh-CN":"自定义聚合","en-US":"Ad-hoc Aggregation"}'::jsonb,
    sort_order = 4,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000011'::uuid;

-- 标准KPI
UPDATE menus SET
    name = '标准KPI',
    name_i18n = NULL,
    sort_order = 3,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000003'::uuid;

-- 基站KPI
UPDATE menus SET
    name = '基站KPI',
    name_i18n = NULL,
    sort_order = 2,
    updated_at = NOW()
WHERE id = 'aaaa0002-1000-0000-0000-000000000002'::uuid;

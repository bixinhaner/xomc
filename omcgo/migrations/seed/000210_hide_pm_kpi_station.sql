-- +goose Up
-- ============================================================
-- 2026-05-28 用户决策:
--   隐藏「性能管理 → 测量任务管理」菜单(及其全部子按钮)
--
-- 目标菜单:
--   name          = '测量任务管理'(seed/000189 重命名后)
--                  或 '基站KPI'(seed/000189 前的旧名)
--   permission_key = 'performance:kpi-station'
--   route_path     = '/performance/kpi-station'
--
-- 隐藏原因: 当前阶段 PM 不走任务驱动上报,该菜单对应的页面与实际数据流脱节。
--           前端 navConfig.ts 已注释相应入口(commit 配套);此迁移确保 DB
--           主导的菜单接口(GET /menus)对前端返回 show_status='hide'。
--
-- 跨环境匹配策略与 seed/000205 一致:name + permission_key + route_path
-- 三重 OR 兜底,任一命中即纳入;CTE 递归把 4 个子按钮(query/add/edit/delete)
-- 一并隐藏。
-- ============================================================

-- +goose StatementBegin
WITH RECURSIVE target_menus AS (
    SELECT id FROM menus
     WHERE name IN ('测量任务管理', '基站KPI')
        OR permission_key = 'performance:kpi-station'
        OR route_path = '/performance/kpi-station'
    UNION ALL
    SELECT m.id FROM menus m
    JOIN target_menus t ON m.parent_id = t.id
)
UPDATE menus
   SET show_status = 'hide',
       updated_at  = NOW()
 WHERE id IN (SELECT id FROM target_menus)
   AND show_status IS DISTINCT FROM 'hide';
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
WITH RECURSIVE target_menus AS (
    SELECT id FROM menus
     WHERE name IN ('测量任务管理', '基站KPI')
        OR permission_key = 'performance:kpi-station'
        OR route_path = '/performance/kpi-station'
    UNION ALL
    SELECT m.id FROM menus m
    JOIN target_menus t ON m.parent_id = t.id
)
UPDATE menus
   SET show_status = 'show',
       updated_at  = NOW()
 WHERE id IN (SELECT id FROM target_menus);
-- +goose StatementEnd

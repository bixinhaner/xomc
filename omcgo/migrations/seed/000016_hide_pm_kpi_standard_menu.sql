-- +goose Up
-- ============================================================
-- 隐藏「性能管理 → 指标库」菜单（及其全部子按钮）
--
-- 目标菜单：
--   name           = '指标库'
--   permission_key = 'performance:kpi-standard'
--   route_path     = '/performance/kpi-standard'
--
-- 隐藏原因：指标库统一收敛到「产品中心 → KPI 指标库」（product:kpi-library），
--           性能侧入口下线。页面代码与路由保留（兼容历史链接），仅从菜单隐藏。
--           前端 navConfig.ts 已注释 perf-kpi-std（静态兜底，commit 配套）；
--           此迁移确保 DB 主导的菜单接口（GET /menus，VITE_DYNAMIC_MENU=true）
--           对前端返回 show_status='hide'。
--
-- 匹配策略与 seed/000012 一致：name + permission_key + route_path 三重 OR 兜底，
-- 任一命中即纳入；CTE 递归把子按钮（query/add/edit/delete/detail）一并隐藏。
-- 三个字段均与「产品中心 → KPI 指标库」(product:kpi-library) 不同，不会误伤。
-- ============================================================

-- +goose StatementBegin
WITH RECURSIVE target_menus AS (
    SELECT id FROM menus
     WHERE name = '指标库'
        OR permission_key = 'performance:kpi-standard'
        OR route_path = '/performance/kpi-standard'
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
     WHERE name = '指标库'
        OR permission_key = 'performance:kpi-standard'
        OR route_path = '/performance/kpi-standard'
    UNION ALL
    SELECT m.id FROM menus m
    JOIN target_menus t ON m.parent_id = t.id
)
UPDATE menus
   SET show_status = 'show',
       updated_at  = NOW()
 WHERE id IN (SELECT id FROM target_menus);
-- +goose StatementEnd

-- +goose Up
-- ============================================================
-- T-0190 旧件下线：隐藏「性能管理 → 指标查询」菜单（及其全部子按钮）
--
-- 目标菜单：
--   name           = '指标查询'（seed/000189 重命名后）
--                   或 '性能查询'（seed/000189 前的旧名）
--   permission_key = 'performance:query'
--   route_path     = '/performance/query'
--   id             = aaaa0002-1000-0000-0000-000000000001
--
-- 隐藏原因：PM 分析体系重做后，指标查询页与新设计矛盾、bug 不修（设计 §3.3）。
--           页面代码与路由保留（兼容历史链接），仅从菜单隐藏入口。
--           前端 navConfig.ts 已注释 perf-query（静态兜底，commit 配套）；
--           此迁移确保 DB 主导的菜单接口（GET /menus，VITE_DYNAMIC_MENU=true）
--           对前端返回 show_status='hide'。
--
-- 匹配策略与 seed/000210 一致：name + permission_key + route_path 三重 OR 兜底，
-- 任一命中即纳入；CTE 递归把 4 个子按钮（query/add/edit/delete）一并隐藏。
-- ============================================================

-- +goose StatementBegin
WITH RECURSIVE target_menus AS (
    SELECT id FROM menus
     WHERE name IN ('指标查询', '性能查询')
        OR permission_key = 'performance:query'
        OR route_path = '/performance/query'
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
     WHERE name IN ('指标查询', '性能查询')
        OR permission_key = 'performance:query'
        OR route_path = '/performance/query'
    UNION ALL
    SELECT m.id FROM menus m
    JOIN target_menus t ON m.parent_id = t.id
)
UPDATE menus
   SET show_status = 'show',
       updated_at  = NOW()
 WHERE id IN (SELECT id FROM target_menus);
-- +goose StatementEnd

-- +goose Up
-- ============================================================
-- 调整「性能管理」子菜单顺序：把「指标查询」排到最后（第三项）。
--
-- 调整前：性能仪表盘(0) → 指标查询(1) → 自定义聚合(2)
-- 调整后：性能仪表盘(0) → 自定义聚合(1) → 指标查询(2)
--
-- 原因：用户要求指标查询放在性能管理子菜单末位。
--       菜单顺序由 menus.sort_order 主导（GET /menus，VITE_DYNAMIC_MENU=true）；
--       前端 navConfig.ts 静态兜底顺序已同步调整（配套 commit）。
-- ============================================================

-- +goose StatementBegin
UPDATE menus SET sort_order = 1, updated_at = NOW() WHERE route_path = '/performance/pm-adhoc';
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE menus SET sort_order = 2, updated_at = NOW() WHERE route_path = '/performance/query';
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
UPDATE menus SET sort_order = 1, updated_at = NOW() WHERE route_path = '/performance/query';
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE menus SET sort_order = 2, updated_at = NOW() WHERE route_path = '/performance/pm-adhoc';
-- +goose StatementEnd

-- +goose Up
-- ISSUE-389：修复主仪表板 LTE 布局缺「可用性」面板。
--
-- 背景：dashboard_kpi_layouts 的 lte 行在 000001 init seed 里以 ON CONFLICT DO NOTHING
-- 灌入。早期已存在的库行（缺 availability / retainability 两面板的 4 面板旧版）因 DO NOTHING
-- 不被后续 re-seed 覆盖，导致现网 lte 布局停留旧版——主仪表板永不渲染「可用性」面板，
-- 小区可用率（LTE_CELL_AVAILABLE→K900010076）即便算出也无处出图。
--
-- 修复：把 lte 行对齐到内置默认布局（与 internal/dashboard/kpi_layout_default.go 同一套
-- 6 面板：traffic / availability / utilization / accessibility / retainability / mobility）。
-- 仅当当前 lte 布局缺 availability 面板时才覆盖，避免冲掉运维已自定义且已含可用性的布局（幂等）。
-- +goose StatementBegin
UPDATE public.dashboard_kpi_layouts
SET layout = '{
      "panels": [
        {"title": "dashboard.panel.traffic",        "metrics": ["LTE_PDCP_VOLUME_DL", "LTE_PDCP_VOLUME_UL", "LTE_PDCP_RATE_DL", "LTE_PDCP_RATE_UL"], "x": 0, "y": 0,  "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.availability",   "metrics": ["LTE_CELL_AVAILABLE"], "x": 6, "y": 0,  "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.utilization",    "metrics": ["LTE_PRB_UTIL_DL", "LTE_PRB_UTIL_UL"], "x": 0, "y": 8,  "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.accessibility",  "metrics": ["WIRELESS_SETUP_SR", "RRC_CONN_SETUP_SR", "ERAB_SETUP_SR", "CSFB_SR"], "x": 6, "y": 8,  "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.retainability",  "metrics": ["ERAB_DROP_RATE"], "x": 0, "y": 16, "w": 6, "h": 8, "chartType": "line"},
        {"title": "dashboard.panel.mobility",       "metrics": ["HO_INTRA_ENB_OUT_SR", "HO_INTRA_ENB_IN_SR", "HO_INTER_ENB_OUT_SR", "HO_INTER_ENB_IN_SR"], "x": 6, "y": 16, "w": 6, "h": 8, "chartType": "line"}
      ]
    }'::jsonb,
    updated_at = now()
WHERE tech = 'lte'
  AND NOT (layout -> 'panels' @> '[{"title": "dashboard.panel.availability"}]'::jsonb);
-- +goose StatementEnd

-- +goose Down
-- 回滚不还原旧 4 面板布局（旧版是缺陷态，无保留价值）；空操作保持幂等。
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd

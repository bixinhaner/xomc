-- +goose Up
-- 「自定义聚合结果」独立 Panel 类型（方向 B 落地）补 DB 侧白名单：
-- 前端 + 后端 DTO 已放行 adhoc_result，但 pm_panels.panel_type CHECK 当初漏加，
-- 导致此类 Panel 保存时被约束拒绝（23514）。此处把 adhoc_result 并入白名单。
ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_panel_type_check;
ALTER TABLE pm_panels ADD CONSTRAINT pm_panels_panel_type_check
    CHECK (panel_type IN ('kpi_card','line_chart','bar_chart','table','gauge','topn','big_number','adhoc_result'));

-- +goose Down
ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_panel_type_check;
ALTER TABLE pm_panels ADD CONSTRAINT pm_panels_panel_type_check
    CHECK (panel_type IN ('kpi_card','line_chart','bar_chart','table','gauge','topn','big_number'));

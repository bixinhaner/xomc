-- +goose Up
-- G6-Gap-5 + G6-Gap-4 (T-0164 收尾 P2 第 1 批)
--
-- 1. 扩展 pm_panels.panel_type CHECK 加 'topn' + 'big_number' 两种新类型
--    · topn        - 排行榜（前 N + 排序方向）
--    · big_number  - 数值大屏（单值放大显示 + 警戒色）
--
-- 2. pm_dashboards 增 is_builtin BOOLEAN 列，区分"系统内置 readonly"与用户自定义
--    · seed/000169 会插入 12 个内置 dashboard（3 制式 × 4 报表）
--    · 前端 DashboardList 按 is_builtin 分"系统内置 / 我的 / 来自分享"三组
--    · 前端 DashboardEditor 检测 is_builtin → hide 编辑 / 删除 按钮，仅允许"另存为派生"

ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_panel_type_check;
ALTER TABLE pm_panels ADD CONSTRAINT pm_panels_panel_type_check
    CHECK (panel_type IN ('kpi_card','line_chart','bar_chart','table','gauge','topn','big_number'));

ALTER TABLE pm_dashboards ADD COLUMN IF NOT EXISTS is_builtin BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_pm_dashboards_is_builtin
    ON pm_dashboards (is_builtin) WHERE is_builtin = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_pm_dashboards_is_builtin;
ALTER TABLE pm_dashboards DROP COLUMN IF EXISTS is_builtin;

ALTER TABLE pm_panels DROP CONSTRAINT IF EXISTS pm_panels_panel_type_check;
ALTER TABLE pm_panels ADD CONSTRAINT pm_panels_panel_type_check
    CHECK (panel_type IN ('kpi_card','line_chart','bar_chart','table','gauge'));

-- +goose Up
-- +goose StatementBegin
-- T-0164-P6 / G6 PM 性能查看：Dashboard / Panel / 用户偏好
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.6
-- 实施 plan：docs/project/plan-T-0164-P6-frontend-dashboard.md
--
-- 与现有 internal/dashboard（运营总览 — 设备/告警统计）解耦：
--   - 表名前缀 pm_ 区分（pm_dashboards / pm_panels / pm_user_dashboard_preferences）
--   - 业务域是"PM 性能查看"：用户可配置仪表盘 + 面板 + 对比 + 派生 + 分享
--
-- 设计要点：
--   - pm_dashboards.shared_with UUID[] 直接存被分享用户列表（小数量场景，单数组比关联表更轻）
--   - parent_dashboard_id FK 自身（ON DELETE SET NULL，源 dashboard 删除时 fork 仍可见但失去 parent 引用）
--   - technology lte/nr/gsm 顶层切换字段
--   - panels.dashboard_id CASCADE — 仪表盘删除时面板随删
--   - panels.adhoc_task_id 软引用 pm_tasks（FK 留 G7 task 表跨域；这里仅冗余 UUID 引用，避免循环依赖）
--   - kpi_card_layout per-user 独立持久化（与 dashboard 分离，避免每改 KPI 卡片就动 dashboard）

CREATE TABLE pm_dashboards (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 TEXT NOT NULL,
    description          TEXT,
    owner_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_with          UUID[] NOT NULL DEFAULT ARRAY[]::UUID[],
    parent_dashboard_id  UUID REFERENCES pm_dashboards(id) ON DELETE SET NULL,
    technology           TEXT NOT NULL CHECK (technology IN ('lte','nr','gsm')),
    layout               JSONB NOT NULL DEFAULT '{"panels":[]}'::JSONB,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pm_dashboards_owner ON pm_dashboards (owner_id);
CREATE INDEX idx_pm_dashboards_shared ON pm_dashboards USING GIN (shared_with);
CREATE INDEX idx_pm_dashboards_parent ON pm_dashboards (parent_dashboard_id) WHERE parent_dashboard_id IS NOT NULL;

CREATE TRIGGER trigger_pm_dashboards_updated_at
    BEFORE UPDATE ON pm_dashboards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================== pm_panels ====================
CREATE TABLE pm_panels (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id      UUID NOT NULL REFERENCES pm_dashboards(id) ON DELETE CASCADE,
    panel_type        TEXT NOT NULL CHECK (panel_type IN ('kpi_card','line_chart','bar_chart','table','gauge')),
    title             TEXT NOT NULL,
    metric_paths      TEXT[] NOT NULL,
    granularity       TEXT NOT NULL CHECK (granularity IN ('15min','hourly','daily','weekly','monthly')),
    dimension         TEXT NOT NULL CHECK (dimension IN ('device','device_group')),
    device_sns        TEXT[],
    device_group_ids  UUID[],
    time_range        JSONB NOT NULL DEFAULT '{}'::JSONB,
    compare_mode      TEXT CHECK (compare_mode IS NULL OR compare_mode IN ('same_window_other_devices','previous_window')),
    adhoc_task_id     UUID,   -- 软引用 pm_tasks(id) where task_subtype='adhoc_aggregation'（避免跨表 FK）
    config            JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pm_panels_dashboard ON pm_panels (dashboard_id);
CREATE INDEX idx_pm_panels_adhoc_task ON pm_panels (adhoc_task_id) WHERE adhoc_task_id IS NOT NULL;

CREATE TRIGGER trigger_pm_panels_updated_at
    BEFORE UPDATE ON pm_panels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================== pm_user_dashboard_preferences ====================
-- per-user KPI 卡片 layout 持久化（与具体 dashboard 解耦，全局生效）
CREATE TABLE pm_user_dashboard_preferences (
    user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    kpi_card_layout  JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trigger_pm_user_dashboard_prefs_updated_at
    BEFORE UPDATE ON pm_user_dashboard_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS pm_user_dashboard_preferences CASCADE;
DROP TABLE IF EXISTS pm_panels CASCADE;
DROP TABLE IF EXISTS pm_dashboards CASCADE;
-- +goose StatementEnd

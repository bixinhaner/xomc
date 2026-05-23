-- +goose Up
-- +goose StatementBegin
-- T-0164 收尾 G6-Gap-3 + G6-Gap-13：制式切换持久化 + KPI 卡片按制式分键
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §7 G6 DoD
-- 实施 plan：docs/project/plan-T-0164-followup-gaps.md §G6-Gap-3, §G6-Gap-13
--
-- 变更：
--   把 pm_user_dashboard_preferences 主键从 user_id 改为 (user_id, technology)，
--   每个用户在每种制式下持有独立的 KPI 卡片 layout + 当前选中仪表盘 + 当前筛选条状态。
--
-- 新增列：
--   - technology TEXT NOT NULL CHECK (lte/nr/gsm)
--   - current_dashboard_id UUID — 切回该制式时自动打开的仪表盘
--   - shared_filters JSONB — 全局筛选条快照（时间窗 / 设备组 / 设备多选；G6-Gap-2 用）
--
-- 迁移已有数据：把现有行 LTE 默认。生产无数据所以是 nop。

-- 先 drop 老 PK + trigger，新建带 technology 的 PK
ALTER TABLE pm_user_dashboard_preferences DROP CONSTRAINT pm_user_dashboard_preferences_pkey;
ALTER TABLE pm_user_dashboard_preferences ADD COLUMN technology TEXT NOT NULL DEFAULT 'lte'
    CHECK (technology IN ('lte', 'nr', 'gsm'));
ALTER TABLE pm_user_dashboard_preferences ADD COLUMN current_dashboard_id UUID;
ALTER TABLE pm_user_dashboard_preferences ADD COLUMN shared_filters JSONB NOT NULL DEFAULT '{}'::JSONB;

ALTER TABLE pm_user_dashboard_preferences
    ADD CONSTRAINT pm_user_dashboard_preferences_pkey PRIMARY KEY (user_id, technology);

-- 软引用：dashboard 被删时 current_dashboard_id 走 SET NULL（不级联删 prefs 整行）
ALTER TABLE pm_user_dashboard_preferences
    ADD CONSTRAINT fk_pm_user_prefs_dashboard
    FOREIGN KEY (current_dashboard_id) REFERENCES pm_dashboards(id) ON DELETE SET NULL;

-- 既有 default 'lte' 占位完后，去掉 default 强制后续显式指定（避免误插）
ALTER TABLE pm_user_dashboard_preferences ALTER COLUMN technology DROP DEFAULT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE pm_user_dashboard_preferences DROP CONSTRAINT IF EXISTS fk_pm_user_prefs_dashboard;
ALTER TABLE pm_user_dashboard_preferences DROP CONSTRAINT IF EXISTS pm_user_dashboard_preferences_pkey;
ALTER TABLE pm_user_dashboard_preferences DROP COLUMN IF EXISTS shared_filters;
ALTER TABLE pm_user_dashboard_preferences DROP COLUMN IF EXISTS current_dashboard_id;
ALTER TABLE pm_user_dashboard_preferences DROP COLUMN IF EXISTS technology;
ALTER TABLE pm_user_dashboard_preferences
    ADD CONSTRAINT pm_user_dashboard_preferences_pkey PRIMARY KEY (user_id);
-- +goose StatementEnd

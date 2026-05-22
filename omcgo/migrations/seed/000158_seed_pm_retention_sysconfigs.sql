-- +goose Up
-- +goose StatementBegin
-- T-0164-P2 / G2 PM 保留策略：注入 5 个 sys_configs 键（金字塔保留默认值）
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.2
-- 引用：omcgo/internal/pm/retention/policies.go DefaultDays
--
-- 5 键 (category='pm.retention', key=<下方 key>)：
--   raw_15min_days → 30   (15min 原始表保留 30 天)
--   hourly_days    → 180  (小时聚合表保留 180 天 ≈ 6 个月)
--   daily_days     → 730  (日聚合表保留 2 年)
--   weekly_days    → 730  (周聚合表保留 2 年)
--   monthly_days   → 1825 (月聚合表保留 5 年)
--
-- 是否公开（is_public）：FALSE — 仅 super_admin / ops 角色在系统设置页可见，
-- 与 ui_custom 类（公开）+ system.session_timeout（非公开）保持一致。
--
-- 实际 hypertable compression / retention policy 的"挂动作"不在本 migration：
--   - pm_metrics 15min 的 policy 在 G3 migration 内挂（引用本表 raw_15min_days）
--   - pm_metrics_hourly / daily / weekly / monthly 的 policy 在 G5 migration 内挂
--
-- 运行时通过 retention.Service.OnSysConfigSaved 监听 category='pm.retention' 保存事件，
-- Reload 后通知 ReloadListener（G3/G5 实施时挂触发 alter_compression_policy / alter_retention_policy）。

INSERT INTO sys_configs (category, key, value, value_type, description, is_public) VALUES
  ('pm.retention', 'raw_15min_days', '30',   'int', 'PM 15min 原始表保留天数（默认 30d）',     FALSE),
  ('pm.retention', 'hourly_days',    '180',  'int', 'PM 小时聚合表保留天数（默认 180d）',       FALSE),
  ('pm.retention', 'daily_days',     '730',  'int', 'PM 日聚合表保留天数（默认 2y）',           FALSE),
  ('pm.retention', 'weekly_days',    '730',  'int', 'PM 周聚合表保留天数（默认 2y）',           FALSE),
  ('pm.retention', 'monthly_days',   '1825', 'int', 'PM 月聚合表保留天数（默认 5y）',           FALSE)
ON CONFLICT (category, key) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM sys_configs
WHERE category = 'pm.retention'
  AND key IN ('raw_15min_days', 'hourly_days', 'daily_days', 'weekly_days', 'monthly_days');
-- +goose StatementEnd

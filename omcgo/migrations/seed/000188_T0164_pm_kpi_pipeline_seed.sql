-- T-0164 PM/KPI 流水线 G1-G8 种子数据合并迁移（draft/pm-kpi-impl → main 合并整理）
--
-- 合并自原 draft 分支 4 个 sub-seed（按 Up 顺序应用 / Down 反向回滚）：
--   - 000158_seed_pm_retention_sysconfigs
--   - 000166_seed_pm_async_jobs_sysconfigs
--   - 000169_seed_pm_builtin_dashboards
--   - 000172_seed_pm_performance_layout_menus
--
-- 合并原因同 omcgo/migrations/000188_T0164_pm_kpi_pipeline_full.sql 文件头。

-- +goose Up

-- ============================================================
-- 000158_seed_pm_retention_sysconfigs
-- ============================================================
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

-- ============================================================
-- 000166_seed_pm_async_jobs_sysconfigs
-- ============================================================
-- +goose StatementBegin
-- T-0164 收尾 G7-Gap-5 + G8-Gap-3：sys_configs 暴露 adhoc retention + asyncjob 阈值
--
-- 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.7 G7 / §4.8 G8
-- 实施 plan：docs/project/plan-T-0164-followup-gaps.md §G7-Gap-5, §G8-Gap-3
--
-- 4 个新 key：
--   - pm.adhoc.retention_days   — adhoc 聚合结果保留期（默认 365 / 1 年）
--   - asyncjob.heartbeat_interval_seconds — 心跳间隔（默认 30s）
--   - asyncjob.zombie_threshold_seconds  — 僵尸任务阈值（默认 300s = 5min）
--   - asyncjob.sweeper_interval_seconds  — Sweeper 扫描间隔（默认 60s）
--
-- 运行时通过 SysConfigSaved 事件 hook 通知 asyncjob.Sweeper Reload。

INSERT INTO sys_configs (category, key, value, value_type, description, is_public)
VALUES
  ('pm.adhoc', 'retention_days', '365', 'int', 'PM adhoc 聚合任务结果保留天数（默认 365d）', FALSE),
  ('asyncjob', 'heartbeat_interval_seconds', '30',  'int', '异步任务心跳上报间隔（秒）', FALSE),
  ('asyncjob', 'zombie_threshold_seconds',  '300', 'int', '心跳超过此值视为僵尸任务（秒）', FALSE),
  ('asyncjob', 'sweeper_interval_seconds',  '60',  'int', 'Sweeper 扫描僵尸任务的间隔（秒）', FALSE)
ON CONFLICT (category, key) DO NOTHING;
-- +goose StatementEnd

-- ============================================================
-- 000169_seed_pm_builtin_dashboards
-- ============================================================
-- G6-Gap-4: 12 个系统内置 readonly 仪表盘 seed
--
-- 3 制式（lte / nr / gsm）× 4 报表类型（全网概览 / 日报 / 周报 / 月报）= 12 dashboard
-- owner_id 用 admin 用户（与 000001_seed_data.sql 一致）；is_builtin=TRUE 标记为系统内置
-- 前端按 is_builtin 区分"系统内置 / 我的 / 来自分享"三组。
-- panels 暂留空（layout='{"panels":[]}'）— P3 期间可补默认 panel 配置。

-- +goose StatementBegin
DO $$
DECLARE
    admin_id UUID := '20000000-0000-0000-0000-000000000001';
BEGIN
    -- LTE × 4
    INSERT INTO pm_dashboards (id, name, description, owner_id, technology, layout, is_builtin)
    VALUES
        ('30000000-1100-0000-0000-000000000001', '[Built-in] LTE 全网概览', '系统内置 - LTE 全网总览仪表盘', admin_id, 'lte', '{"panels":[]}'::JSONB, TRUE),
        ('30000000-1200-0000-0000-000000000002', '[Built-in] LTE 性能日报', '系统内置 - LTE 性能日报',     admin_id, 'lte', '{"panels":[]}'::JSONB, TRUE),
        ('30000000-1300-0000-0000-000000000003', '[Built-in] LTE 性能周报', '系统内置 - LTE 性能周报',     admin_id, 'lte', '{"panels":[]}'::JSONB, TRUE),
        ('30000000-1400-0000-0000-000000000004', '[Built-in] LTE 性能月报', '系统内置 - LTE 性能月报',     admin_id, 'lte', '{"panels":[]}'::JSONB, TRUE)
    ON CONFLICT (id) DO NOTHING;

    -- NR × 4
    INSERT INTO pm_dashboards (id, name, description, owner_id, technology, layout, is_builtin)
    VALUES
        ('30000000-2100-0000-0000-000000000001', '[Built-in] NR 全网概览',  '系统内置 - 5G NR 全网总览仪表盘', admin_id, 'nr',  '{"panels":[]}'::JSONB, TRUE),
        ('30000000-2200-0000-0000-000000000002', '[Built-in] NR 性能日报',  '系统内置 - 5G NR 性能日报',       admin_id, 'nr',  '{"panels":[]}'::JSONB, TRUE),
        ('30000000-2300-0000-0000-000000000003', '[Built-in] NR 性能周报',  '系统内置 - 5G NR 性能周报',       admin_id, 'nr',  '{"panels":[]}'::JSONB, TRUE),
        ('30000000-2400-0000-0000-000000000004', '[Built-in] NR 性能月报',  '系统内置 - 5G NR 性能月报',       admin_id, 'nr',  '{"panels":[]}'::JSONB, TRUE)
    ON CONFLICT (id) DO NOTHING;

    -- GSM × 4
    INSERT INTO pm_dashboards (id, name, description, owner_id, technology, layout, is_builtin)
    VALUES
        ('30000000-3100-0000-0000-000000000001', '[Built-in] GSM 全网概览', '系统内置 - GSM 全网总览仪表盘', admin_id, 'gsm', '{"panels":[]}'::JSONB, TRUE),
        ('30000000-3200-0000-0000-000000000002', '[Built-in] GSM 性能日报', '系统内置 - GSM 性能日报',       admin_id, 'gsm', '{"panels":[]}'::JSONB, TRUE),
        ('30000000-3300-0000-0000-000000000003', '[Built-in] GSM 性能周报', '系统内置 - GSM 性能周报',       admin_id, 'gsm', '{"panels":[]}'::JSONB, TRUE),
        ('30000000-3400-0000-0000-000000000004', '[Built-in] GSM 性能月报', '系统内置 - GSM 性能月报',       admin_id, 'gsm', '{"panels":[]}'::JSONB, TRUE)
    ON CONFLICT (id) DO NOTHING;
END $$;
-- +goose StatementEnd

-- ============================================================
-- 000172_seed_pm_performance_layout_menus
-- ============================================================
-- 1. 性能查看（新主入口，sort_order=0 排第一）
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa0002-1000-0000-0000-000000000010'::uuid,
    '性能查看',
    '{"zh-CN":"性能查看","en-US":"PM Dashboard"}'::jsonb,
    'menu',
    'performance:pm-dashboard',
    'aaaa0002-0000-0000-0000-000000000001'::uuid,
    0,
    '/performance',
    'performance/PmDashboard/PerformanceLayout',
    'DashboardOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 2. 自定义聚合（adhoc 任务）
INSERT INTO menus (
    id, name, name_i18n, type, permission_key, parent_id, sort_order,
    route_path, component_path, icon, status, show_status
)
VALUES (
    'aaaa0002-1000-0000-0000-000000000011'::uuid,
    '自定义聚合',
    '{"zh-CN":"自定义聚合","en-US":"Ad-hoc Aggregation"}'::jsonb,
    'menu',
    'performance:pm-adhoc',
    'aaaa0002-0000-0000-0000-000000000001'::uuid,
    4,
    '/performance/pm-adhoc',
    'performance/PmAdhoc',
    'FundProjectionScreenOutlined',
    'normal',
    'show'
)
ON CONFLICT (id) DO NOTHING;

-- 3. 角色绑定：admin / operator / viewer 全部可见
INSERT INTO role_menus (role_id, menu_id)
SELECT role_id, menu_id
FROM (
    VALUES
        ('10000000-0000-0000-0000-000000000001'::uuid, 'aaaa0002-1000-0000-0000-000000000010'::uuid),
        ('10000000-0000-0000-0000-000000000001'::uuid, 'aaaa0002-1000-0000-0000-000000000011'::uuid),
        ('10000000-0000-0000-0000-000000000002'::uuid, 'aaaa0002-1000-0000-0000-000000000010'::uuid),
        ('10000000-0000-0000-0000-000000000002'::uuid, 'aaaa0002-1000-0000-0000-000000000011'::uuid),
        ('10000000-0000-0000-0000-000000000003'::uuid, 'aaaa0002-1000-0000-0000-000000000010'::uuid),
        ('10000000-0000-0000-0000-000000000003'::uuid, 'aaaa0002-1000-0000-0000-000000000011'::uuid)
) AS bindings(role_id, menu_id)
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- +goose Down

-- ============================================================
-- 000172_seed_pm_performance_layout_menus (Down)
-- ============================================================
DELETE FROM role_menus
WHERE menu_id IN (
    'aaaa0002-1000-0000-0000-000000000010'::uuid,
    'aaaa0002-1000-0000-0000-000000000011'::uuid
);

DELETE FROM menus
WHERE id IN (
    'aaaa0002-1000-0000-0000-000000000010'::uuid,
    'aaaa0002-1000-0000-0000-000000000011'::uuid
);

-- ============================================================
-- 000169_seed_pm_builtin_dashboards (Down)
-- ============================================================
DELETE FROM pm_dashboards WHERE is_builtin = TRUE
  AND id::text LIKE '30000000-_%00-0000-0000-_____________';

-- ============================================================
-- 000166_seed_pm_async_jobs_sysconfigs (Down)
-- ============================================================
-- +goose StatementBegin
DELETE FROM sys_configs WHERE category IN ('pm.adhoc', 'asyncjob')
  AND key IN ('retention_days', 'heartbeat_interval_seconds', 'zombie_threshold_seconds', 'sweeper_interval_seconds');
-- +goose StatementEnd

-- ============================================================
-- 000158_seed_pm_retention_sysconfigs (Down)
-- ============================================================
-- +goose StatementBegin
DELETE FROM sys_configs
WHERE category = 'pm.retention'
  AND key IN ('raw_15min_days', 'hourly_days', 'daily_days', 'weekly_days', 'monthly_days');
-- +goose StatementEnd


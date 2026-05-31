-- +goose Up
-- T-0190 旧件下线：清理旧拖拽仪表盘的 12 个空壳内置盘（is_builtin=TRUE，layout='{"panels":[]}'）。
-- 来源：000188_T0164_pm_kpi_pipeline_seed.sql 第 89-112 行。新流程（左右栏布局 + 即席聚合）
-- 不再读 pm_dashboards 表，这 12 行属于死数据，随旧拖拽编辑器下线一并清理。
--
-- 版本号说明：DDL 用 000224（pm_adhoc_task_runs），本 seed 取下一个全局唯一号 000225。
--   migrate-up 走 `--paths migrations,migrations/seed` 对两目录在同一 goose_db_version 表里
--   各跑一次 goose.Up，DDL 与 seed 不能同号，故 seed 取 DDL 之后的下一个空号 225。
DELETE FROM pm_dashboards WHERE is_builtin = TRUE;

-- +goose Down
-- 复原 000188 第 89-112 行插入的 12 个空壳内置盘（LTE/NR/GSM × {全网概览/日报/周报/月报}）。
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

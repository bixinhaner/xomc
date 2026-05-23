-- +goose Up
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

-- +goose Down
DELETE FROM pm_dashboards WHERE is_builtin = TRUE
  AND id::text LIKE '30000000-_%00-0000-0000-_____________';

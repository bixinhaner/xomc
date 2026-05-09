-- T-0100-P0: license 治理层审计日志表
-- 来源 PRD: docs/project/prd/F06-license.md §6.2
-- 关闭 Risk: R-109（License 治理层无审计日志，合规缺口）
--
-- 用途：T-0015 enforcement 引擎的拒绝事件 / 容量与过期阈值告警 / 用户写操作
-- （Import / Activate / Revoke）统一落审计；为前端 LicenseLogs 页面与等保 2.0
-- 三级 8.1.4.7「重要操作日志保留 ≥ 6 个月」合规提供数据源。
--
-- log_type 取值（9 种）：
--   import / activate / revoke / query_detail —— 用户写 / 敏感读
--   enforcement_capacity / enforcement_expiry —— enforcer 拒绝
--   capacity_alert / expiry_alert / auto_expire —— monitor cron 触发
--
-- result 取值：success / failed / denied / warning
-- actor_user_id NULL 表示 system 操作（cron / enforcer 触发）

-- +goose Up
CREATE TABLE IF NOT EXISTS license_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id      UUID REFERENCES licenses(id) ON DELETE SET NULL,
    log_type        VARCHAR(32) NOT NULL,
    actor_user_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    result          VARCHAR(16) NOT NULL,
    details         JSONB NOT NULL DEFAULT '{}'::jsonb,
    client_ip       INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_license_logs_log_type CHECK (log_type IN (
        'import', 'activate', 'revoke', 'query_detail',
        'enforcement_capacity', 'enforcement_expiry',
        'capacity_alert', 'expiry_alert', 'auto_expire'
    )),
    CONSTRAINT chk_license_logs_result CHECK (result IN (
        'success', 'failed', 'denied', 'warning'
    ))
);

CREATE INDEX IF NOT EXISTS idx_license_logs_license_id  ON license_logs(license_id);
CREATE INDEX IF NOT EXISTS idx_license_logs_created_at  ON license_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_license_logs_log_type    ON license_logs(log_type);
CREATE INDEX IF NOT EXISTS idx_license_logs_actor       ON license_logs(actor_user_id) WHERE actor_user_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_license_logs_actor;
DROP INDEX IF EXISTS idx_license_logs_log_type;
DROP INDEX IF EXISTS idx_license_logs_created_at;
DROP INDEX IF EXISTS idx_license_logs_license_id;
DROP TABLE IF EXISTS license_logs;

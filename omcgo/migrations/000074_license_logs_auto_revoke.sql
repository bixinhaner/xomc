-- T-0100-P3: 扩展 license_logs.log_type CHECK 约束，加入 'auto_revoke_by_activate'
--
-- 来源 PRD: docs/project/prd/F06-license.md §5.3.2、§15 Q1=B
--
-- 业务背景：Activate 同 (device_type, region) 维度已有 active license 时，
-- 用户在前端 Modal 二次确认通过后，调用 POST /licenses/activate { force: true }。
-- 后端先 revoke 同维度旧 license（写一条 log_type=auto_revoke_by_activate），
-- 再 activate 新 license（log_type=activate）。
--
-- 与普通 revoke 区分：
--   * revoke           — 用户主动调用 POST /licenses/:id/revoke
--   * auto_revoke_by_activate — Activate 流程联动触发，details 含 triggered_by_activate
--
-- 幂等性：DROP CONSTRAINT IF EXISTS + 重新 ADD，保证多次执行结果一致。

-- +goose Up
ALTER TABLE license_logs DROP CONSTRAINT IF EXISTS chk_license_logs_log_type;

ALTER TABLE license_logs ADD CONSTRAINT chk_license_logs_log_type CHECK (log_type IN (
    'import', 'activate', 'revoke', 'query_detail',
    'enforcement_capacity', 'enforcement_expiry',
    'capacity_alert', 'expiry_alert', 'auto_expire',
    'auto_revoke_by_activate'
));

-- +goose Down
ALTER TABLE license_logs DROP CONSTRAINT IF EXISTS chk_license_logs_log_type;

ALTER TABLE license_logs ADD CONSTRAINT chk_license_logs_log_type CHECK (log_type IN (
    'import', 'activate', 'revoke', 'query_detail',
    'enforcement_capacity', 'enforcement_expiry',
    'capacity_alert', 'expiry_alert', 'auto_expire'
));

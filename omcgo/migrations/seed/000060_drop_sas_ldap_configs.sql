-- +goose Up
-- ============================================================
-- 000060_drop_sas_ldap_configs.sql
-- 配合 PRD docs/prd/system/config.md v1.0：下线 SAS / LDAP 两个 Tab。
-- 清理 sys_configs 表里所有 category IN ('sas','ldap') 的历史行。
-- 不动 schema；纯数据清理；不可逆（前端入口已删，无需保留）。
-- ============================================================

DELETE FROM sys_configs WHERE category IN ('sas', 'ldap');

-- +goose Down
-- 不可逆：原始 SAS/LDAP 配置行无种子来源，无法自动恢复。如需回滚，
-- 请回滚 PRD config.md v1.0 + 对应前端 commit 后由用户重新录入。
SELECT 1;

-- +goose Up
-- 放宽 ops_audit_logs.target_type 至 varchar(64)（#124）。
--
-- 根因：建表时该列为 varchar(16)，容不下代码写入的 'maintenance_window'
-- （18 字符，internal/ops/service_ext.go），INSERT 报
-- `value too long for type character varying(16)`（SQLSTATE 22001）；
-- AuditLogService.Log 失败仅 Warn 不阻塞业务 → 维护窗口创建/审批的审计留痕
-- 全部静默丢失，合规审计链路断。
-- 现查全仓写入值：task / device / user / scope / maintenance_window（最长 18），
-- 放宽至 varchar(64) 留足余量。放宽不重写数据、不丢数据，重复执行幂等。
ALTER TABLE ops_audit_logs ALTER COLUMN target_type TYPE varchar(64);

-- +goose Down
-- 回退到 varchar(16)；若存量已有 >16 字符的值（如 maintenance_window）会失败
-- （属预期，避免静默截断审计数据）。
ALTER TABLE ops_audit_logs ALTER COLUMN target_type TYPE varchar(16);

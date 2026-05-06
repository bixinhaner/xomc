-- v1.0：删除 users.carrier 字段。
--
-- 决议：超管判定改用 source = 'builtIn'；多 carrier 隔离能力由 role_device_groups
-- 通过设备分组层提供。详见 omcgo/docs/prd/system/users.md §11.11。
--
-- ⚠️ 不可逆操作。回滚需要：
--   1. ALTER TABLE users ADD COLUMN carrier VARCHAR(4) NULL（自动恢复）
--   2. CREATE INDEX idx_users_carrier ON users(carrier) WHERE carrier IS NOT NULL（自动恢复）
--   3. 业务数据需手动从备份恢复（DROP COLUMN 已物理删除全部 carrier 值）
--
-- 部署前置审计建议：
--   SELECT id, username, source, carrier FROM users
--   WHERE carrier IS NOT NULL OR source = 'builtIn'
--   ORDER BY created_at;
-- 业务确认所有 source != 'builtIn' 的"假超管"（carrier IS NULL 的非 admin 用户）
-- 已被合理处置（保持 source='admin' 或显式删除），再执行本迁移。

-- +goose Up
DROP INDEX IF EXISTS idx_users_carrier;
ALTER TABLE users DROP COLUMN IF EXISTS carrier;

-- +goose Down
ALTER TABLE users ADD COLUMN IF NOT EXISTS carrier VARCHAR(4);
CREATE INDEX IF NOT EXISTS idx_users_carrier ON users(carrier) WHERE carrier IS NOT NULL;
-- 注：业务数据无法自动恢复，需从备份恢复 carrier 值。

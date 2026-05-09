-- +goose Up
-- B2 修复：device_registrations 表补 carrier 列。
-- internal/device/device_registration_pg_repository.go 的 regColumns / scanRegistration /
-- 4 处 SQL 都引用 carrier，但 migrations/000003_devices.sql 建表时漏了此列，
-- 每次 RegisterFromInform → GetBySerialNumber 报 SQLSTATE 42703 column "carrier" does not exist。
-- pre-registration 流程在 silent fail 模式下工作（查不到就当作 nil → 走默认分组），不阻塞业务但日志噪声大。
ALTER TABLE device_registrations ADD COLUMN IF NOT EXISTS carrier VARCHAR(4) NOT NULL DEFAULT 'cmcc';

-- +goose Down
ALTER TABLE device_registrations DROP COLUMN IF EXISTS carrier;

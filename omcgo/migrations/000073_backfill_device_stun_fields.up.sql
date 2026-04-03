-- 000073_backfill_device_stun_fields.up.sql
-- 补齐 000034 遗漏的 STUN/NAT 字段（当 DB 已跳过 000034 时）
-- 使用 IF NOT EXISTS 保证幂等性

ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS nat_detected BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS udp_connection_request_address VARCHAR(64);

COMMENT ON COLUMN devices.nat_detected IS 'STUN 检测到设备位于 NAT 后';
COMMENT ON COLUMN devices.udp_connection_request_address IS 'UDP Connection Request 地址 (IP:Port)';

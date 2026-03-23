-- 000034_add_device_stun_fields.up.sql
-- 添加 STUN/NAT 相关字段到 devices 表

ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS nat_detected BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS udp_connection_request_address VARCHAR(64);

COMMENT ON COLUMN devices.nat_detected IS 'STUN 检测到设备位于 NAT 后';
COMMENT ON COLUMN devices.udp_connection_request_address IS 'UDP Connection Request 地址 (IP:Port)';

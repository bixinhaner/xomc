-- 000073_backfill_device_stun_fields.down.sql
-- 回滚：删除 STUN/NAT 字段

ALTER TABLE devices
    DROP COLUMN IF EXISTS nat_detected,
    DROP COLUMN IF EXISTS udp_connection_request_address;

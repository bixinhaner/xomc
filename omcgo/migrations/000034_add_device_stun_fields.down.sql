-- 000034_add_device_stun_fields.down.sql

ALTER TABLE devices
    DROP COLUMN IF EXISTS nat_detected,
    DROP COLUMN IF EXISTS udp_connection_request_address;

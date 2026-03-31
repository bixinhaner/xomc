-- ============================================================
-- 000067_create_device_registrations.up.sql
-- 设备预注册表：手动/批量导入注册的设备 SN 及附属信息
-- ============================================================

CREATE TABLE device_registrations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial_number   VARCHAR(64)  NOT NULL,
    group_id        UUID         NOT NULL REFERENCES device_groups(id),
    device_id       UUID,                    -- 设备实际上线后关联
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending',
                    -- pending: 已注册待上线
                    -- online:  设备已上线关联
                    -- expired: 超时未上线

    -- 导入时可携带的规划信息
    site_name       VARCHAR(128),
    device_name     VARCHAR(128),
    longitude       DOUBLE PRECISION,
    latitude        DOUBLE PRECISION,
    height          DECIMAL(10,2),
    azimuth         SMALLINT,                -- 水平方位角 [0, 359]
    tilt_angle      SMALLINT,                -- 机械下倾角 [0, 9]
    beam_width      SMALLINT,                -- 垂直3dB波束宽度 [1, 9]
    remark          TEXT,

    -- 审计
    created_by      VARCHAR(64),
    import_batch_id UUID,                    -- Excel 批量导入的批次ID
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 只对待上线的 SN 做唯一约束
CREATE UNIQUE INDEX idx_dr_sn_pending ON device_registrations (serial_number)
    WHERE status = 'pending';
CREATE INDEX idx_dr_group ON device_registrations (group_id);
CREATE INDEX idx_dr_status ON device_registrations (status);
CREATE INDEX idx_dr_batch ON device_registrations (import_batch_id)
    WHERE import_batch_id IS NOT NULL;

CREATE TRIGGER trigger_dr_updated_at
    BEFORE UPDATE ON device_registrations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- device_info: 设备运维管理扩展信息（与 devices 表 1:1 关联）
-- devices 存储 ACS 自动写入的核心身份信息
-- device_info 存储运维人员手动填写 + TR069 参数自动同步的扩展信息

CREATE TABLE IF NOT EXISTS device_info (
    device_id         UUID PRIMARY KEY,

    -- 运维标识（手动填写）
    device_name       VARCHAR(128),
    address           VARCHAR(256),
    remark            TEXT,
    project_status    VARCHAR(20),
    height            DECIMAL(10,2),

    -- 无线参数（从 TR069 参数同步）
    eci               VARCHAR(64),
    pci               VARCHAR(64),
    cell_id           VARCHAR(64),
    freq_point        VARCHAR(32),
    bandwidth         DECIMAL(8,2),
    transmit_power    DECIMAL(8,2),
    plmn              VARCHAR(32),

    -- 状态（从设备/参数/告警同步）
    rf_status         VARCHAR(20),
    cell_status       VARCHAR(20),
    mme_status        VARCHAR(20),
    sync_status       VARCHAR(32),
    kpi_status        VARCHAR(20),

    -- 硬件信息
    mac               VARCHAR(64),
    hardware_version  VARCHAR(64),

    -- 时间记录
    first_online_time TIMESTAMPTZ,
    last_offline_time TIMESTAMPTZ,
    run_time          BIGINT DEFAULT 0,

    -- 审计
    creator           VARCHAR(64),
    updater           VARCHAR(64),

    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE device_info IS '设备运维管理扩展信息';
COMMENT ON COLUMN device_info.device_id IS '关联 devices.id，1:1 关系';
COMMENT ON COLUMN device_info.project_status IS '工程状态：building/delivered/operating/deactivated';
COMMENT ON COLUMN device_info.rf_status IS '射频状态：on/off/error';
COMMENT ON COLUMN device_info.cell_status IS '小区状态：normal/fault/unconfigured/decommissioned';
COMMENT ON COLUMN device_info.run_time IS '累计运行时长(秒)';

-- 索引
CREATE INDEX idx_device_info_rf_status      ON device_info (rf_status);
CREATE INDEX idx_device_info_cell_status    ON device_info (cell_status);
CREATE INDEX idx_device_info_project_status ON device_info (project_status);

-- updated_at 触发器（复用已有函数）
CREATE TRIGGER trigger_device_info_updated_at
    BEFORE UPDATE ON device_info
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

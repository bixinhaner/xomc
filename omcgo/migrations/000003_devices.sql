-- +goose Up
-- ============================================================
-- 000003_devices.up.sql
-- 设备核心表
-- ============================================================

-- 1. 设备分组表
CREATE TABLE device_groups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(128) NOT NULL,
    parent_id       UUID REFERENCES device_groups(id) ON DELETE CASCADE,
    -- 修复 2026-05-22：原 VARCHAR(4) 装不下 'other' (5 字符) 的国际运营商 carrier code
    -- (赞比亚 ZED 等)。在源头直接 VARCHAR(16) 避免后续 ALTER（且 devices.carrier 是
    -- partition key，PG 16 完全禁止 ALTER 分区键列 type — 见 000159 注释）。
    carrier         VARCHAR(16),
    description     TEXT,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    remark          TEXT,
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    level           SMALLINT NOT NULL DEFAULT 1,
    created_by      VARCHAR(64),
    updated_by      VARCHAR(64),
    matching_mode   VARCHAR(16),
    name_rule_list  JSONB,
    lac_list        INTEGER[],
    tac_list        INTEGER[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_dg_level CHECK (level IN (1, 2)),
    CONSTRAINT chk_dg_level_parent CHECK ((level = 1 AND parent_id IS NULL) OR (level = 2 AND parent_id IS NOT NULL)),
    CONSTRAINT chk_dg_matching_mode CHECK (matching_mode IS NULL OR matching_mode IN ('deviceName', 'lac', 'tac'))
);
CREATE INDEX idx_dg_parent ON device_groups (parent_id);
CREATE INDEX idx_dg_carrier ON device_groups (carrier) WHERE carrier IS NOT NULL;
CREATE INDEX idx_dg_name ON device_groups (name);
CREATE UNIQUE INDEX idx_dg_name_parent ON device_groups (name, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid));
CREATE INDEX idx_dg_matching_mode ON device_groups (matching_mode) WHERE matching_mode IS NOT NULL;
CREATE TRIGGER trigger_dg_updated_at BEFORE UPDATE ON device_groups FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 2. 设备分组成员
CREATE TABLE device_group_members (
    group_id   UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    device_id  UUID NOT NULL,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, device_id),
    CONSTRAINT uq_dgm_device UNIQUE (device_id)
);
CREATE INDEX idx_dgm_device ON device_group_members (device_id);
CREATE INDEX idx_dgm_group ON device_group_members (group_id);

-- 3. 角色-设备组数据权限
CREATE TABLE role_device_groups (
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    group_id   UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, group_id)
);
CREATE INDEX idx_rdg_role ON role_device_groups (role_id);
CREATE INDEX idx_rdg_group ON role_device_groups (group_id);
COMMENT ON TABLE role_device_groups IS '角色数据权限：角色可见的设备组';

-- 4. 设备核心表（分区表）
CREATE TABLE devices (
    id                     UUID NOT NULL DEFAULT gen_random_uuid(),
    serial_number          VARCHAR(64) NOT NULL,
    oui                    VARCHAR(6) NOT NULL,
    product_class          VARCHAR(64),
    manufacturer           VARCHAR(128),
    model_name             VARCHAR(128),
    -- 修复 2026-05-22：见同文件 device_groups.carrier 注释 — VARCHAR(16) 直供，
    -- 避免 partition key 列 type 不可 ALTER 限制。
    carrier                VARCHAR(16) NOT NULL,
    technology             VARCHAR(3) NOT NULL,
    data_model_id          UUID,
    status                 VARCHAR(20) NOT NULL DEFAULT 'discovered',
    firmware_version       VARCHAR(64),
    ip_address             INET,
    connection_request_url VARCHAR(256),
    nat_detected           BOOLEAN NOT NULL DEFAULT false,
    udp_connection_request_address VARCHAR(64),
    last_inform_at         TIMESTAMPTZ,
    last_inform_events     JSONB,
    inform_interval        INTEGER DEFAULT 300,
    site_name              VARCHAR(128),
    site_id                VARCHAR(64),
    latitude               DOUBLE PRECISION,
    longitude              DOUBLE PRECISION,
    extension_data         JSONB,
    deleted_at             TIMESTAMPTZ,
    deleted_by             TEXT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, carrier)
) PARTITION BY LIST (carrier);
CREATE TABLE devices_cmcc PARTITION OF devices FOR VALUES IN ('cmcc');
CREATE TABLE devices_ctcc PARTITION OF devices FOR VALUES IN ('ctcc');
CREATE TABLE devices_cucc PARTITION OF devices FOR VALUES IN ('cucc');
CREATE UNIQUE INDEX idx_devices_serial_number ON devices (serial_number, carrier);
CREATE INDEX idx_devices_carrier_status ON devices (carrier, status);
CREATE INDEX idx_devices_oui ON devices (oui);
CREATE INDEX idx_devices_carrier_tech ON devices (carrier, technology);
CREATE INDEX idx_devices_status ON devices (status);
CREATE INDEX idx_devices_last_inform ON devices (last_inform_at);
CREATE INDEX idx_devices_deleted_at ON devices (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE TRIGGER trigger_devices_updated_at BEFORE UPDATE ON devices FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 5. 设备运维扩展信息
CREATE TABLE device_info (
    device_id         UUID PRIMARY KEY,
    device_name       VARCHAR(128),
    address           VARCHAR(256),
    remark            TEXT,
    project_status    VARCHAR(20),
    height            DECIMAL(10,2),
    eci               VARCHAR(64),
    pci               VARCHAR(64),
    cell_id           VARCHAR(64),
    freq_point        VARCHAR(32),
    bandwidth         DECIMAL(8,2),
    transmit_power    DECIMAL(8,2),
    plmn              VARCHAR(32),
    rf_status         VARCHAR(20),
    cell_status       VARCHAR(20),
    mme_status        VARCHAR(20),
    sync_status       VARCHAR(32),
    kpi_status        VARCHAR(20),
    num_of_cells      INTEGER DEFAULT 1,
    gps_status        VARCHAR(20),
    alarm_severity    VARCHAR(20),
    license_status    VARCHAR(20),
    mac               VARCHAR(64),
    hardware_version  VARCHAR(64),
    first_online_time TIMESTAMPTZ,
    last_online_time  TIMESTAMPTZ,
    last_offline_time TIMESTAMPTZ,
    run_time          BIGINT DEFAULT 0,
    creator           VARCHAR(64),
    updater           VARCHAR(64),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_device_info_rf_status ON device_info (rf_status);
CREATE INDEX idx_device_info_cell_status ON device_info (cell_status);
CREATE INDEX idx_device_info_project_status ON device_info (project_status);
CREATE INDEX idx_device_info_gps_status ON device_info (gps_status);
CREATE INDEX idx_device_info_alarm_severity ON device_info (alarm_severity);
CREATE INDEX idx_device_info_license_status ON device_info (license_status);
CREATE INDEX idx_device_info_last_online_time ON device_info (last_online_time);
CREATE TRIGGER trigger_device_info_updated_at BEFORE UPDATE ON device_info FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 6. 设备参数表（Hash 分区）
CREATE TABLE device_parameters (
    device_id        UUID         NOT NULL,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN      DEFAULT false,
    last_updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    fap_instance     SMALLINT     NOT NULL DEFAULT 0,
    param_group      VARCHAR(32)  NOT NULL DEFAULT 'other',
    PRIMARY KEY (device_id, parameter_path)
) PARTITION BY HASH (device_id);
-- +goose StatementBegin
DO $$
BEGIN
    FOR i IN 0..31 LOOP
        EXECUTE format('CREATE TABLE device_parameters_p%s PARTITION OF device_parameters FOR VALUES WITH (MODULUS 32, REMAINDER %s)', lpad(i::text, 2, '0'), i);
    END LOOP;
END $$;
-- +goose StatementEnd
CREATE INDEX idx_device_params_device ON device_parameters (device_id);
CREATE INDEX idx_device_params_path_prefix ON device_parameters (device_id, parameter_path varchar_pattern_ops);

-- 7. 设备预注册表
CREATE TABLE device_registrations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial_number   VARCHAR(64) NOT NULL,
    group_id        UUID NOT NULL REFERENCES device_groups(id),
    device_id       UUID,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    site_name       VARCHAR(128),
    device_name     VARCHAR(128),
    longitude       DOUBLE PRECISION,
    latitude        DOUBLE PRECISION,
    height          DECIMAL(10,2),
    azimuth         SMALLINT,
    tilt_angle      SMALLINT,
    beam_width      SMALLINT,
    remark          TEXT,
    created_by      VARCHAR(64),
    import_batch_id UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_dr_sn_pending ON device_registrations (serial_number) WHERE status = 'pending';
CREATE INDEX idx_dr_group ON device_registrations (group_id);
CREATE INDEX idx_dr_status ON device_registrations (status);
CREATE INDEX idx_dr_batch ON device_registrations (import_batch_id) WHERE import_batch_id IS NOT NULL;
CREATE TRIGGER trigger_dr_updated_at BEFORE UPDATE ON device_registrations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 8. 设备规则表
CREATE TABLE device_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    priority        INTEGER NOT NULL DEFAULT 0,
    target_group_id UUID REFERENCES device_groups(id) ON DELETE SET NULL,
    enabled         BOOLEAN NOT NULL DEFAULT false,
    matching_mode   VARCHAR(16),
    name_rule_list  JSONB,
    lac_list        INTEGER[],
    tac_list        INTEGER[],
    description     TEXT,
    operators       TEXT,
    created_by      VARCHAR(64),
    updated_by      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_device_rules_priority_unique ON device_rules (priority) WHERE enabled = true;
CREATE INDEX idx_device_rules_target_group ON device_rules (target_group_id);
CREATE INDEX idx_device_rules_enabled ON device_rules (enabled);
CREATE INDEX idx_device_rules_priority ON device_rules (priority);
CREATE TRIGGER trigger_device_rules_updated_at BEFORE UPDATE ON device_rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 9. 设备规则应用任务
CREATE TABLE device_rule_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id         UUID NOT NULL REFERENCES device_rules(id) ON DELETE CASCADE,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    total_devices   INTEGER DEFAULT 0,
    matched_count   INTEGER DEFAULT 0,
    failed_count    INTEGER DEFAULT 0,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    error_message   TEXT,
    created_by      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_device_rule_tasks_rule ON device_rule_tasks (rule_id);
CREATE INDEX idx_device_rule_tasks_status ON device_rule_tasks (status);
CREATE INDEX idx_device_rule_tasks_created ON device_rule_tasks (created_at DESC);

-- 10. 循环依赖字段
ALTER TABLE device_groups ADD COLUMN bound_rule_id UUID REFERENCES device_rules(id) ON DELETE SET NULL;
COMMENT ON COLUMN device_groups.bound_rule_id IS '绑定的设备规则ID';

-- 11. 设备任务表
CREATE TABLE device_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,
    method          VARCHAR(64) NOT NULL,
    params          JSONB,
    priority        INTEGER DEFAULT 10,
    command_key     VARCHAR(128),
    cwmp_id         VARCHAR(256),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    retry_count     INTEGER DEFAULT 0,
    max_retries     INTEGER DEFAULT 3,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    result          JSONB,
    error_code      INTEGER,
    error_message   TEXT,
    source          VARCHAR(32) DEFAULT 'api',
    creator_id      VARCHAR(64),
    description     TEXT
);
CREATE INDEX idx_device_tasks_device_sn ON device_tasks(device_sn);
CREATE INDEX idx_device_tasks_status ON device_tasks(status);
CREATE INDEX idx_device_tasks_created_at ON device_tasks(created_at);
CREATE INDEX idx_device_tasks_cwmp_id ON device_tasks(cwmp_id);
CREATE INDEX idx_device_tasks_pending ON device_tasks(device_sn, status, priority, created_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE IF EXISTS device_tasks CASCADE;
DROP TABLE IF EXISTS device_rule_tasks CASCADE;
DROP TABLE IF EXISTS device_rules CASCADE;
DROP TABLE IF EXISTS device_registrations CASCADE;
DROP TABLE IF EXISTS device_parameters CASCADE;
DROP TABLE IF EXISTS device_info CASCADE;
DROP TABLE IF EXISTS devices CASCADE;
DROP TABLE IF EXISTS role_device_groups CASCADE;
DROP TABLE IF EXISTS device_group_members CASCADE;
DROP TABLE IF EXISTS device_groups CASCADE;

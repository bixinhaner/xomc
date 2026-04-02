-- ============================================================
-- 000070_add_device_table_comments.down.sql
-- 清除设备相关表的字段注释
-- ============================================================

-- ===== devices 表 =====
COMMENT ON TABLE devices IS NULL;
COMMENT ON COLUMN devices.id IS NULL;
COMMENT ON COLUMN devices.serial_number IS NULL;
COMMENT ON COLUMN devices.oui IS NULL;
COMMENT ON COLUMN devices.product_class IS NULL;
COMMENT ON COLUMN devices.manufacturer IS NULL;
COMMENT ON COLUMN devices.model_name IS NULL;
COMMENT ON COLUMN devices.carrier IS NULL;
COMMENT ON COLUMN devices.technology IS NULL;
COMMENT ON COLUMN devices.data_model_id IS NULL;
COMMENT ON COLUMN devices.status IS NULL;
COMMENT ON COLUMN devices.firmware_version IS NULL;
COMMENT ON COLUMN devices.ip_address IS NULL;
COMMENT ON COLUMN devices.connection_request_url IS NULL;
-- nat_detected 和 udp_connection_request_address 由迁移 000034 管理，此处不清除
COMMENT ON COLUMN devices.last_inform_at IS NULL;
COMMENT ON COLUMN devices.last_inform_events IS NULL;
COMMENT ON COLUMN devices.inform_interval IS NULL;
COMMENT ON COLUMN devices.site_name IS NULL;
COMMENT ON COLUMN devices.site_id IS NULL;
COMMENT ON COLUMN devices.latitude IS NULL;
COMMENT ON COLUMN devices.longitude IS NULL;
COMMENT ON COLUMN devices.extension_data IS NULL;
COMMENT ON COLUMN devices.created_at IS NULL;
COMMENT ON COLUMN devices.updated_at IS NULL;
-- deleted_at 由迁移 000069 管理，此处不清除

-- ===== device_info 表 =====
COMMENT ON TABLE device_info IS NULL;
COMMENT ON COLUMN device_info.device_id IS NULL;
COMMENT ON COLUMN device_info.device_name IS NULL;
COMMENT ON COLUMN device_info.address IS NULL;
COMMENT ON COLUMN device_info.remark IS NULL;
COMMENT ON COLUMN device_info.project_status IS NULL;
COMMENT ON COLUMN device_info.height IS NULL;
COMMENT ON COLUMN device_info.eci IS NULL;
COMMENT ON COLUMN device_info.pci IS NULL;
COMMENT ON COLUMN device_info.cell_id IS NULL;
COMMENT ON COLUMN device_info.freq_point IS NULL;
COMMENT ON COLUMN device_info.bandwidth IS NULL;
COMMENT ON COLUMN device_info.transmit_power IS NULL;
COMMENT ON COLUMN device_info.plmn IS NULL;
COMMENT ON COLUMN device_info.rf_status IS NULL;
COMMENT ON COLUMN device_info.cell_status IS NULL;
COMMENT ON COLUMN device_info.mme_status IS NULL;
COMMENT ON COLUMN device_info.sync_status IS NULL;
COMMENT ON COLUMN device_info.kpi_status IS NULL;
COMMENT ON COLUMN device_info.gps_status IS NULL;
COMMENT ON COLUMN device_info.alarm_severity IS NULL;
COMMENT ON COLUMN device_info.license_status IS NULL;
COMMENT ON COLUMN device_info.mac IS NULL;
COMMENT ON COLUMN device_info.hardware_version IS NULL;
COMMENT ON COLUMN device_info.first_online_time IS NULL;
COMMENT ON COLUMN device_info.last_offline_time IS NULL;
COMMENT ON COLUMN device_info.run_time IS NULL;
COMMENT ON COLUMN device_info.creator IS NULL;
COMMENT ON COLUMN device_info.updater IS NULL;
COMMENT ON COLUMN device_info.created_at IS NULL;
COMMENT ON COLUMN device_info.updated_at IS NULL;

-- ===== device_parameters 表 =====
COMMENT ON TABLE device_parameters IS NULL;
COMMENT ON COLUMN device_parameters.device_id IS NULL;
COMMENT ON COLUMN device_parameters.parameter_path IS NULL;
COMMENT ON COLUMN device_parameters.parameter_value IS NULL;
COMMENT ON COLUMN device_parameters.parameter_type IS NULL;
COMMENT ON COLUMN device_parameters.writable IS NULL;
COMMENT ON COLUMN device_parameters.last_updated_at IS NULL;

-- ===== device_registrations 表 =====
COMMENT ON TABLE device_registrations IS NULL;
COMMENT ON COLUMN device_registrations.id IS NULL;
COMMENT ON COLUMN device_registrations.serial_number IS NULL;
COMMENT ON COLUMN device_registrations.group_id IS NULL;
COMMENT ON COLUMN device_registrations.device_id IS NULL;
COMMENT ON COLUMN device_registrations.status IS NULL;
COMMENT ON COLUMN device_registrations.site_name IS NULL;
COMMENT ON COLUMN device_registrations.device_name IS NULL;
COMMENT ON COLUMN device_registrations.longitude IS NULL;
COMMENT ON COLUMN device_registrations.latitude IS NULL;
COMMENT ON COLUMN device_registrations.height IS NULL;
COMMENT ON COLUMN device_registrations.azimuth IS NULL;
COMMENT ON COLUMN device_registrations.tilt_angle IS NULL;
COMMENT ON COLUMN device_registrations.beam_width IS NULL;
COMMENT ON COLUMN device_registrations.remark IS NULL;
COMMENT ON COLUMN device_registrations.created_by IS NULL;
COMMENT ON COLUMN device_registrations.import_batch_id IS NULL;
COMMENT ON COLUMN device_registrations.created_at IS NULL;
COMMENT ON COLUMN device_registrations.updated_at IS NULL;

-- ===== device_groups 表 =====
COMMENT ON TABLE device_groups IS NULL;
COMMENT ON COLUMN device_groups.id IS NULL;
COMMENT ON COLUMN device_groups.name IS NULL;
COMMENT ON COLUMN device_groups.parent_id IS NULL;
COMMENT ON COLUMN device_groups.carrier IS NULL;
COMMENT ON COLUMN device_groups.description IS NULL;
COMMENT ON COLUMN device_groups.sort_order IS NULL;
COMMENT ON COLUMN device_groups.created_at IS NULL;
COMMENT ON COLUMN device_groups.updated_at IS NULL;

-- ===== device_group_members 表 =====
COMMENT ON TABLE device_group_members IS NULL;
COMMENT ON COLUMN device_group_members.group_id IS NULL;
COMMENT ON COLUMN device_group_members.device_id IS NULL;
COMMENT ON COLUMN device_group_members.added_at IS NULL;

-- ===== user_column_configs 表 =====
COMMENT ON TABLE user_column_configs IS NULL;
COMMENT ON COLUMN user_column_configs.user_id IS NULL;
COMMENT ON COLUMN user_column_configs.page_key IS NULL;
COMMENT ON COLUMN user_column_configs.columns IS NULL;
COMMENT ON COLUMN user_column_configs.created_at IS NULL;
COMMENT ON COLUMN user_column_configs.updated_at IS NULL;

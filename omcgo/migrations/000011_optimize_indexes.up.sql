-- ============================================================
-- 000011_optimize_indexes.up.sql
-- GIN/BRIN/优化索引
-- ============================================================

-- devices
CREATE INDEX IF NOT EXISTS idx_devices_extension_data_gin ON devices USING GIN (extension_data jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_devices_last_inform_events_gin ON devices USING GIN (last_inform_events jsonb_path_ops);

-- mr_records
CREATE INDEX IF NOT EXISTS idx_mr_records_measurement_data_gin ON mr_records USING GIN (measurement_data jsonb_path_ops);

-- alarms_active
CREATE INDEX IF NOT EXISTS idx_alarms_active_additional_info_gin ON alarms_active USING GIN (additional_info jsonb_path_ops);

-- device_tasks
CREATE INDEX IF NOT EXISTS idx_device_tasks_params_gin ON device_tasks USING GIN (params jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_device_tasks_result_gin ON device_tasks USING GIN (result jsonb_path_ops);

-- audit_logs
CREATE INDEX IF NOT EXISTS idx_audit_logs_details_gin ON audit_logs USING GIN (details jsonb_path_ops);

-- system_logs
CREATE INDEX IF NOT EXISTS idx_system_logs_details_gin ON system_logs USING GIN (details jsonb_path_ops);

-- alarm_rules
CREATE INDEX IF NOT EXISTS idx_alarm_rules_condition_config_gin ON alarm_rules USING GIN (condition_config jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_alarm_rules_action_config_gin ON alarm_rules USING GIN (action_config jsonb_path_ops);

-- mml_tasks
CREATE INDEX IF NOT EXISTS idx_mml_tasks_commands_gin ON mml_tasks USING GIN (commands jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_mml_tasks_results_gin ON mml_tasks USING GIN (results jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_mml_tasks_device_sns_gin ON mml_tasks USING GIN (device_sns);

-- pm_tasks
CREATE INDEX IF NOT EXISTS idx_pm_tasks_device_sns_gin ON pm_tasks USING GIN (device_sns);
CREATE INDEX IF NOT EXISTS idx_pm_tasks_kpi_codes_gin ON pm_tasks USING GIN (kpi_codes);

-- data_model_definitions
CREATE INDEX IF NOT EXISTS idx_dm_parameter_tree_gin ON data_model_definitions USING GIN (parameter_tree);

-- config_templates
CREATE INDEX IF NOT EXISTS idx_ct_parameters_gin ON config_templates USING GIN (parameters);

-- config_baselines
CREATE INDEX IF NOT EXISTS idx_config_baselines_params_gin ON config_baselines USING GIN (params);

-- config_tasks
CREATE INDEX IF NOT EXISTS idx_config_tasks_device_sns_gin ON config_tasks USING GIN (device_sns);
CREATE INDEX IF NOT EXISTS idx_config_tasks_params_gin ON config_tasks USING GIN (params);

-- config_neighbors
CREATE INDEX IF NOT EXISTS idx_config_neighbors_params_gin ON config_neighbors USING GIN (params);

-- kpi_definitions
CREATE INDEX IF NOT EXISTS idx_kpi_definitions_counters_gin ON kpi_definitions USING GIN (counters);

-- ops_templates
CREATE INDEX IF NOT EXISTS idx_ops_templates_steps_gin ON ops_templates USING GIN (steps jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_ops_templates_tags_gin ON ops_templates USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_ops_templates_target_device_types_gin ON ops_templates USING GIN (target_device_types);

-- ops_tasks
CREATE INDEX IF NOT EXISTS idx_ops_tasks_device_sns_gin ON ops_tasks USING GIN (device_sns);

-- mml_commands
CREATE INDEX IF NOT EXISTS idx_mml_commands_param_template_gin ON mml_commands USING GIN (param_template);
CREATE INDEX IF NOT EXISTS idx_mml_commands_product_types_gin ON mml_commands USING GIN (product_types);

-- mml_scripts
CREATE INDEX IF NOT EXISTS idx_mml_scripts_tags_gin ON mml_scripts USING GIN (tags);

-- licenses
CREATE INDEX IF NOT EXISTS idx_licenses_features_gin ON licenses USING GIN (features);

-- firmware_versions
CREATE INDEX IF NOT EXISTS idx_firmware_versions_compatible_oui_gin ON firmware_versions USING GIN (compatible_oui);

-- report_definitions
CREATE INDEX IF NOT EXISTS idx_report_defs_kpi_codes_gin ON report_definitions USING GIN (kpi_codes);
CREATE INDEX IF NOT EXISTS idx_report_defs_device_groups_gin ON report_definitions USING GIN (device_groups);

-- backup
CREATE INDEX IF NOT EXISTS idx_backup_tasks_target_ids_gin ON backup_tasks USING GIN (target_ids);
CREATE INDEX IF NOT EXISTS idx_backup_schedules_target_ids_gin ON backup_schedules USING GIN (target_ids);

-- data_model_import_log
CREATE INDEX IF NOT EXISTS idx_dm_import_log_changes_gin ON data_model_import_log USING GIN (changes_summary jsonb_path_ops);

-- BRIN indexes
CREATE INDEX IF NOT EXISTS idx_system_logs_created_at_brin ON system_logs USING BRIN (created_at) WITH (pages_per_range = 128);
CREATE INDEX IF NOT EXISTS idx_ne_message_logs_created_at_brin ON ne_message_logs USING BRIN (created_at) WITH (pages_per_range = 128);
CREATE INDEX IF NOT EXISTS idx_alarms_active_raised_at_brin ON alarms_active USING BRIN (raised_at) WITH (pages_per_range = 128);

-- Additional optimization
CREATE INDEX IF NOT EXISTS idx_devices_status_last_inform ON devices (status, last_inform_at) WHERE deleted_at IS NULL AND status = 'active';

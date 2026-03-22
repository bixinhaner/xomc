-- 000050 down: 回滚 JSONB GIN 索引

-- 第一优先级：大表
DROP INDEX IF EXISTS idx_devices_extension_data_gin;
DROP INDEX IF EXISTS idx_devices_last_inform_events_gin;
DROP INDEX IF EXISTS idx_mr_records_measurement_data_gin;
DROP INDEX IF EXISTS idx_alarms_active_additional_info_gin;
DROP INDEX IF EXISTS idx_device_tasks_params_gin;
DROP INDEX IF EXISTS idx_device_tasks_result_gin;
DROP INDEX IF EXISTS idx_audit_logs_details_gin;
DROP INDEX IF EXISTS idx_system_logs_details_gin;

-- 第二优先级：中等规模表
DROP INDEX IF EXISTS idx_alarm_rules_condition_config_gin;
DROP INDEX IF EXISTS idx_alarm_rules_action_config_gin;
DROP INDEX IF EXISTS idx_mml_tasks_commands_gin;
DROP INDEX IF EXISTS idx_mml_tasks_results_gin;
DROP INDEX IF EXISTS idx_mml_tasks_device_sns_gin;
DROP INDEX IF EXISTS idx_pm_tasks_device_sns_gin;
DROP INDEX IF EXISTS idx_pm_tasks_kpi_codes_gin;

-- 第三优先级：配置/元数据表
DROP INDEX IF EXISTS idx_dm_parameter_tree_gin;
DROP INDEX IF EXISTS idx_ct_parameters_gin;
DROP INDEX IF EXISTS idx_config_baselines_params_gin;
DROP INDEX IF EXISTS idx_config_tasks_device_sns_gin;
DROP INDEX IF EXISTS idx_config_tasks_params_gin;
DROP INDEX IF EXISTS idx_config_neighbors_params_gin;
DROP INDEX IF EXISTS idx_kpi_definitions_counters_gin;
DROP INDEX IF EXISTS idx_ops_templates_steps_gin;
DROP INDEX IF EXISTS idx_ops_templates_tags_gin;
DROP INDEX IF EXISTS idx_ops_templates_target_device_types_gin;
DROP INDEX IF EXISTS idx_ops_tasks_device_sns_gin;
DROP INDEX IF EXISTS idx_mml_commands_param_template_gin;
DROP INDEX IF EXISTS idx_mml_commands_product_types_gin;
DROP INDEX IF EXISTS idx_mml_scripts_tags_gin;
DROP INDEX IF EXISTS idx_licenses_features_gin;
DROP INDEX IF EXISTS idx_firmware_versions_compatible_oui_gin;
DROP INDEX IF EXISTS idx_report_defs_kpi_codes_gin;
DROP INDEX IF EXISTS idx_report_defs_device_groups_gin;
DROP INDEX IF EXISTS idx_backup_tasks_target_ids_gin;
DROP INDEX IF EXISTS idx_backup_schedules_target_ids_gin;
DROP INDEX IF EXISTS idx_dm_import_log_changes_gin;

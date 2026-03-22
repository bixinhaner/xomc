-- 000050: 为关键 JSONB 列添加 GIN 索引
--
-- 背景：项目存在大量 JSONB 字段但零 GIN 索引，大表 JSONB 查询会导致全表扫描。
--
-- 索引策略：
--   1. 大表（高频写入/查询）的 JSONB 列使用 jsonb_path_ops（更小更快，适合 @> 包含查询）
--   2. 配置/元数据表的 JSONB 列使用默认 GIN 操作符（支持更多操作符如 ?, ?|, ?&）
--
-- 注意：CREATE INDEX CONCURRENTLY 不能在事务中执行。
-- 如果使用 golang-migrate（默认事务模式），需要以下任一方式：
--   a) 手动执行本迁移：psql -f 000050_add_jsonb_gin_indexes.up.sql
--   b) 使用 migrate 的 -x 选项禁用事务（如支持）
--   c) 去掉 CONCURRENTLY 关键字在事务中执行（会短暂锁表，小表可接受）
--
-- 为兼容事务模式迁移，本文件不使用 CONCURRENTLY。
-- 生产环境建议：对大表（devices, mr_records, alarms_active, device_tasks）手动使用
-- CONCURRENTLY 方式创建索引，避免锁表影响业务。

-- ============================================================================
-- 第一优先级：大表 / 高频查询表（使用 jsonb_path_ops）
-- ============================================================================

-- devices: 10万+ 行，extension_data 存放厂商扩展字段，按条件过滤场景多
CREATE INDEX IF NOT EXISTS idx_devices_extension_data_gin
    ON devices USING GIN (extension_data jsonb_path_ops);

-- devices: last_inform_events 存放 Inform 事件码数组，用于按事件类型过滤
CREATE INDEX IF NOT EXISTS idx_devices_last_inform_events_gin
    ON devices USING GIN (last_inform_events jsonb_path_ops);

-- mr_records: TimescaleDB 超表，measurement_data 是核心查询字段
CREATE INDEX IF NOT EXISTS idx_mr_records_measurement_data_gin
    ON mr_records USING GIN (measurement_data jsonb_path_ops);

-- alarms_active: 活动告警表，additional_info 用于告警关联和过滤
CREATE INDEX IF NOT EXISTS idx_alarms_active_additional_info_gin
    ON alarms_active USING GIN (additional_info jsonb_path_ops);

-- device_tasks: 任务参数和结果的 JSONB 查询
CREATE INDEX IF NOT EXISTS idx_device_tasks_params_gin
    ON device_tasks USING GIN (params jsonb_path_ops);

CREATE INDEX IF NOT EXISTS idx_device_tasks_result_gin
    ON device_tasks USING GIN (result jsonb_path_ops);

-- audit_logs: 审计日志高频查询，按 details 中的操作内容过滤
CREATE INDEX IF NOT EXISTS idx_audit_logs_details_gin
    ON audit_logs USING GIN (details jsonb_path_ops);

-- system_logs: 系统日志 details 字段按内容过滤
CREATE INDEX IF NOT EXISTS idx_system_logs_details_gin
    ON system_logs USING GIN (details jsonb_path_ops);

-- ============================================================================
-- 第二优先级：中等规模表（使用 jsonb_path_ops）
-- ============================================================================

-- alarm_rules: condition_config 和 action_config 用于规则匹配
CREATE INDEX IF NOT EXISTS idx_alarm_rules_condition_config_gin
    ON alarm_rules USING GIN (condition_config jsonb_path_ops);

CREATE INDEX IF NOT EXISTS idx_alarm_rules_action_config_gin
    ON alarm_rules USING GIN (action_config jsonb_path_ops);

-- mml_tasks: commands, results, device_sns 是批量 MML 任务的核心数据
CREATE INDEX IF NOT EXISTS idx_mml_tasks_commands_gin
    ON mml_tasks USING GIN (commands jsonb_path_ops);

CREATE INDEX IF NOT EXISTS idx_mml_tasks_results_gin
    ON mml_tasks USING GIN (results jsonb_path_ops);

CREATE INDEX IF NOT EXISTS idx_mml_tasks_device_sns_gin
    ON mml_tasks USING GIN (device_sns);

-- pm_tasks: device_sns 和 kpi_codes 用于按设备/指标查询任务
CREATE INDEX IF NOT EXISTS idx_pm_tasks_device_sns_gin
    ON pm_tasks USING GIN (device_sns);

CREATE INDEX IF NOT EXISTS idx_pm_tasks_kpi_codes_gin
    ON pm_tasks USING GIN (kpi_codes);

-- ============================================================================
-- 第三优先级：配置/元数据表（使用默认 GIN 操作符，支持 ?, ?|, ?& 查询）
-- ============================================================================

-- data_model_definitions: parameter_tree 是核心数据模型，需支持键存在性查询
CREATE INDEX IF NOT EXISTS idx_dm_parameter_tree_gin
    ON data_model_definitions USING GIN (parameter_tree);

-- config_templates: parameters 存放模板参数定义
CREATE INDEX IF NOT EXISTS idx_ct_parameters_gin
    ON config_templates USING GIN (parameters);

-- config_baselines: params 存放基线参数
CREATE INDEX IF NOT EXISTS idx_config_baselines_params_gin
    ON config_baselines USING GIN (params);

-- config_tasks: device_sns 和 params 存放任务目标设备和参数
CREATE INDEX IF NOT EXISTS idx_config_tasks_device_sns_gin
    ON config_tasks USING GIN (device_sns);

CREATE INDEX IF NOT EXISTS idx_config_tasks_params_gin
    ON config_tasks USING GIN (params);

-- config_neighbors: params 存放邻区参数
CREATE INDEX IF NOT EXISTS idx_config_neighbors_params_gin
    ON config_neighbors USING GIN (params);

-- kpi_definitions: counters 存放计数器数组定义
CREATE INDEX IF NOT EXISTS idx_kpi_definitions_counters_gin
    ON kpi_definitions USING GIN (counters);

-- ops_templates: steps 和 tags 用于模板查询和分类
CREATE INDEX IF NOT EXISTS idx_ops_templates_steps_gin
    ON ops_templates USING GIN (steps jsonb_path_ops);

CREATE INDEX IF NOT EXISTS idx_ops_templates_tags_gin
    ON ops_templates USING GIN (tags);

CREATE INDEX IF NOT EXISTS idx_ops_templates_target_device_types_gin
    ON ops_templates USING GIN (target_device_types);

-- ops_tasks: device_sns 用于按设备查询任务
CREATE INDEX IF NOT EXISTS idx_ops_tasks_device_sns_gin
    ON ops_tasks USING GIN (device_sns);

-- mml_commands: param_template 和 product_types 用于命令匹配
CREATE INDEX IF NOT EXISTS idx_mml_commands_param_template_gin
    ON mml_commands USING GIN (param_template);

CREATE INDEX IF NOT EXISTS idx_mml_commands_product_types_gin
    ON mml_commands USING GIN (product_types);

-- mml_scripts: tags 用于脚本分类查询
CREATE INDEX IF NOT EXISTS idx_mml_scripts_tags_gin
    ON mml_scripts USING GIN (tags);

-- licenses: features 数组用于许可功能查询
CREATE INDEX IF NOT EXISTS idx_licenses_features_gin
    ON licenses USING GIN (features);

-- firmware_versions: compatible_oui 用于兼容性查询
CREATE INDEX IF NOT EXISTS idx_firmware_versions_compatible_oui_gin
    ON firmware_versions USING GIN (compatible_oui);

-- report_definitions: kpi_codes 和 device_groups 用于报表查询
CREATE INDEX IF NOT EXISTS idx_report_defs_kpi_codes_gin
    ON report_definitions USING GIN (kpi_codes);

CREATE INDEX IF NOT EXISTS idx_report_defs_device_groups_gin
    ON report_definitions USING GIN (device_groups);

-- backup_tasks: target_ids 用于按目标查询备份任务
CREATE INDEX IF NOT EXISTS idx_backup_tasks_target_ids_gin
    ON backup_tasks USING GIN (target_ids);

-- backup_schedules: target_ids 用于按目标查询备份计划
CREATE INDEX IF NOT EXISTS idx_backup_schedules_target_ids_gin
    ON backup_schedules USING GIN (target_ids);

-- data_model_import_log: changes_summary 用于变更审计查询
CREATE INDEX IF NOT EXISTS idx_dm_import_log_changes_gin
    ON data_model_import_log USING GIN (changes_summary jsonb_path_ops);

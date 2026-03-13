-- ============================================================================
-- 000035: 初始化种子数据
-- 为系统提供开箱即用的基础配置数据
-- 所有 INSERT 使用 ON CONFLICT DO NOTHING 确保幂等性
-- 种子数据 UUID 使用 30xxxxxx- 前缀以便区分和管理
-- ============================================================================

-- ============================================================================
-- 1. KPI 定义 (kpi_definitions)
--    标准 LTE/NR 关键性能指标
-- ============================================================================

-- LTE KPI
INSERT INTO kpi_definitions (id, name, display_name, formula, unit, category, carrier, technology, counters) VALUES
('30000000-0001-4000-8000-000000000001', 'RRC_CONN_SETUP_SR', 'RRC连接建立成功率',
 '(rrc_conn_setup_succ / rrc_conn_setup_att) * 100', '%', 'accessibility', NULL, 'lte',
 '["rrc_conn_setup_succ", "rrc_conn_setup_att"]'::jsonb),

('30000000-0001-4000-8000-000000000002', 'ERAB_SETUP_SR', 'E-RAB建立成功率',
 '(erab_setup_succ / erab_setup_att) * 100', '%', 'accessibility', NULL, 'lte',
 '["erab_setup_succ", "erab_setup_att"]'::jsonb),

('30000000-0001-4000-8000-000000000003', 'INTRA_FREQ_HO_SR', '同频切换成功率',
 '(intra_freq_ho_succ / intra_freq_ho_att) * 100', '%', 'mobility', NULL, 'lte',
 '["intra_freq_ho_succ", "intra_freq_ho_att"]'::jsonb),

('30000000-0001-4000-8000-000000000004', 'INTER_FREQ_HO_SR', '异频切换成功率',
 '(inter_freq_ho_succ / inter_freq_ho_att) * 100', '%', 'mobility', NULL, 'lte',
 '["inter_freq_ho_succ", "inter_freq_ho_att"]'::jsonb),

('30000000-0001-4000-8000-000000000005', 'CALL_DROP_RATE', '掉话率',
 '(erab_abnormal_release / erab_release_total) * 100', '%', 'retainability', NULL, 'lte',
 '["erab_abnormal_release", "erab_release_total"]'::jsonb),

('30000000-0001-4000-8000-000000000006', 'DL_PRB_UTIL', '下行PRB利用率',
 '(dl_prb_used_avg / dl_prb_available) * 100', '%', 'utilization', NULL, 'lte',
 '["dl_prb_used_avg", "dl_prb_available"]'::jsonb)

ON CONFLICT (name) DO NOTHING;

-- NR KPI
INSERT INTO kpi_definitions (id, name, display_name, formula, unit, category, carrier, technology, counters) VALUES
('30000000-0001-4000-8000-000000000011', 'NR_RRC_CONN_SETUP_SR', '5G RRC连接建立成功率',
 '(nr_rrc_conn_setup_succ / nr_rrc_conn_setup_att) * 100', '%', 'accessibility', NULL, 'nr',
 '["nr_rrc_conn_setup_succ", "nr_rrc_conn_setup_att"]'::jsonb),

('30000000-0001-4000-8000-000000000012', 'NR_SESSION_SETUP_SR', '5G会话建立成功率',
 '(nr_session_setup_succ / nr_session_setup_att) * 100', '%', 'accessibility', NULL, 'nr',
 '["nr_session_setup_succ", "nr_session_setup_att"]'::jsonb),

('30000000-0001-4000-8000-000000000013', 'NR_INTRA_FREQ_HO_SR', '5G同频切换成功率',
 '(nr_intra_freq_ho_succ / nr_intra_freq_ho_att) * 100', '%', 'mobility', NULL, 'nr',
 '["nr_intra_freq_ho_succ", "nr_intra_freq_ho_att"]'::jsonb),

('30000000-0001-4000-8000-000000000014', 'NR_INTER_FREQ_HO_SR', '5G异频切换成功率',
 '(nr_inter_freq_ho_succ / nr_inter_freq_ho_att) * 100', '%', 'mobility', NULL, 'nr',
 '["nr_inter_freq_ho_succ", "nr_inter_freq_ho_att"]'::jsonb),

('30000000-0001-4000-8000-000000000015', 'NR_CALL_DROP_RATE', '5G掉话率',
 '(nr_session_abnormal_release / nr_session_release_total) * 100', '%', 'retainability', NULL, 'nr',
 '["nr_session_abnormal_release", "nr_session_release_total"]'::jsonb),

('30000000-0001-4000-8000-000000000016', 'NR_DL_PRB_UTIL', '5G下行PRB利用率',
 '(nr_dl_prb_used_avg / nr_dl_prb_available) * 100', '%', 'utilization', NULL, 'nr',
 '["nr_dl_prb_used_avg", "nr_dl_prb_available"]'::jsonb)

ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- 2. 告警规则 (alarm_rules)
--    默认告警检测规则
-- ============================================================================

INSERT INTO alarm_rules (id, name, description, alarm_code, severity, condition_type, condition_config, action_type, action_config, enabled) VALUES
('30000000-0002-4000-8000-000000000001', 'CPU使用率过高', '设备CPU使用率超过阈值时触发告警',
 'CPU_HIGH', 2, 'threshold',
 '{"metric": "cpu_usage", "operator": "gt", "value": 90, "duration": 300}'::jsonb,
 'alarm', '{"auto_clear": true, "clear_value": 80}'::jsonb, true),

('30000000-0002-4000-8000-000000000002', '内存使用率过高', '设备内存使用率超过阈值时触发告警',
 'MEM_HIGH', 2, 'threshold',
 '{"metric": "memory_usage", "operator": "gt", "value": 85, "duration": 300}'::jsonb,
 'alarm', '{"auto_clear": true, "clear_value": 75}'::jsonb, true),

('30000000-0002-4000-8000-000000000003', '设备温度过高', '设备温度超过安全阈值时触发告警',
 'TEMP_HIGH', 1, 'threshold',
 '{"metric": "temperature", "operator": "gt", "value": 65, "duration": 60}'::jsonb,
 'alarm', '{"auto_clear": true, "clear_value": 55}'::jsonb, true),

('30000000-0002-4000-8000-000000000004', '传输链路断开', '设备传输链路中断时触发告警',
 'LINK_DOWN', 1, 'event',
 '{"event_type": "link_status", "expected_value": "down"}'::jsonb,
 'alarm', '{"auto_clear": true, "clear_event": "link_up"}'::jsonb, true),

('30000000-0002-4000-8000-000000000005', '天馈驻波比过高', '天馈系统VSWR超过阈值时触发告警',
 'VSWR_HIGH', 2, 'threshold',
 '{"metric": "vswr", "operator": "gt", "value": 3.0, "duration": 600}'::jsonb,
 'alarm', '{"auto_clear": true, "clear_value": 2.5}'::jsonb, true),

('30000000-0002-4000-8000-000000000006', '信号质量劣化', 'RSRP低于阈值时触发告警',
 'SIGNAL_DEGRADE', 3, 'threshold',
 '{"metric": "rsrp", "operator": "lt", "value": -110, "duration": 900}'::jsonb,
 'alarm', '{"auto_clear": true, "clear_value": -105}'::jsonb, true),

('30000000-0002-4000-8000-000000000007', '小区不可用', '小区服务中断时触发告警',
 'CELL_UNAVAIL', 1, 'event',
 '{"event_type": "cell_status", "expected_value": "unavailable"}'::jsonb,
 'alarm', '{"auto_clear": true, "clear_event": "cell_available"}'::jsonb, true),

('30000000-0002-4000-8000-000000000008', '存储空间不足', '设备存储使用率超过阈值时触发告警',
 'DISK_FULL', 3, 'threshold',
 '{"metric": "disk_usage", "operator": "gt", "value": 90, "duration": 600}'::jsonb,
 'alarm', '{"auto_clear": true, "clear_value": 80}'::jsonb, true)

ON CONFLICT DO NOTHING;

-- ============================================================================
-- 3. KPI 阈值 (kpi_thresholds)
--    各 KPI 的四级告警阈值
-- ============================================================================

-- LTE KPI 阈值
INSERT INTO kpi_thresholds (id, kpi_name, technology, warning_threshold, minor_threshold, major_threshold, critical_threshold, comparison, enabled, description) VALUES
('30000000-0003-4000-8000-000000000001', 'RRC_CONN_SETUP_SR', 'lte', 98, 95, 90, 85, 'lt', true, 'RRC连接建立成功率低于阈值告警'),
('30000000-0003-4000-8000-000000000002', 'ERAB_SETUP_SR', 'lte', 98, 95, 90, 85, 'lt', true, 'E-RAB建立成功率低于阈值告警'),
('30000000-0003-4000-8000-000000000003', 'INTRA_FREQ_HO_SR', 'lte', 97, 94, 90, 85, 'lt', true, '同频切换成功率低于阈值告警'),
('30000000-0003-4000-8000-000000000004', 'INTER_FREQ_HO_SR', 'lte', 95, 90, 85, 80, 'lt', true, '异频切换成功率低于阈值告警'),
('30000000-0003-4000-8000-000000000005', 'CALL_DROP_RATE', 'lte', 0.5, 1.0, 2.0, 5.0, 'gt', true, '掉话率超过阈值告警'),
('30000000-0003-4000-8000-000000000006', 'DL_PRB_UTIL', 'lte', 70, 80, 90, 95, 'gt', true, '下行PRB利用率超过阈值告警'),

-- NR KPI 阈值
('30000000-0003-4000-8000-000000000011', 'NR_RRC_CONN_SETUP_SR', 'nr', 98, 95, 90, 85, 'lt', true, '5G RRC连接建立成功率低于阈值告警'),
('30000000-0003-4000-8000-000000000012', 'NR_SESSION_SETUP_SR', 'nr', 98, 95, 90, 85, 'lt', true, '5G会话建立成功率低于阈值告警'),
('30000000-0003-4000-8000-000000000013', 'NR_INTRA_FREQ_HO_SR', 'nr', 97, 94, 90, 85, 'lt', true, '5G同频切换成功率低于阈值告警'),
('30000000-0003-4000-8000-000000000014', 'NR_INTER_FREQ_HO_SR', 'nr', 95, 90, 85, 80, 'lt', true, '5G异频切换成功率低于阈值告警'),
('30000000-0003-4000-8000-000000000015', 'NR_CALL_DROP_RATE', 'nr', 0.5, 1.0, 2.0, 5.0, 'gt', true, '5G掉话率超过阈值告警'),
('30000000-0003-4000-8000-000000000016', 'NR_DL_PRB_UTIL', 'nr', 70, 80, 90, 95, 'gt', true, '5G下行PRB利用率超过阈值告警')

ON CONFLICT DO NOTHING;

-- ============================================================================
-- 4. 设备分组 (device_groups)
--    按运营商的顶层分组
-- ============================================================================

INSERT INTO device_groups (id, name, carrier, description, sort_order) VALUES
('30000000-0004-4000-8000-000000000001', '中国移动基站', 'cmcc', '中国移动所有基站设备分组', 1),
('30000000-0004-4000-8000-000000000002', '中国电信基站', 'ctcc', '中国电信所有基站设备分组', 2),
('30000000-0004-4000-8000-000000000003', '中国联通基站', 'cucc', '中国联通所有基站设备分组', 3)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 5. 配置模板 (config_templates)
--    默认开站和配置模板
-- ============================================================================

INSERT INTO config_templates (id, name, carrier, technology, template_type, parameters, priority, description) VALUES
-- LTE 开站模板（三运营商）
('30000000-0005-4000-8000-000000000001', 'LTE移动基站开站模板', 'cmcc', 'lte', 'provisioning',
 '{"admin_state": "locked", "earfcn": 38400, "bandwidth": 20, "pci": "auto", "tac": "", "cell_id": "", "tx_power": 43}'::jsonb,
 100, '中国移动LTE基站自动开站默认配置模板'),

('30000000-0005-4000-8000-000000000002', 'LTE电信基站开站模板', 'ctcc', 'lte', 'provisioning',
 '{"admin_state": "locked", "earfcn": 1650, "bandwidth": 20, "pci": "auto", "tac": "", "cell_id": "", "tx_power": 43}'::jsonb,
 100, '中国电信LTE基站自动开站默认配置模板'),

('30000000-0005-4000-8000-000000000003', 'LTE联通基站开站模板', 'cucc', 'lte', 'provisioning',
 '{"admin_state": "locked", "earfcn": 1850, "bandwidth": 20, "pci": "auto", "tac": "", "cell_id": "", "tx_power": 43}'::jsonb,
 100, '中国联通LTE基站自动开站默认配置模板'),

-- NR 开站模板
('30000000-0005-4000-8000-000000000004', 'NR移动基站开站模板', 'cmcc', 'nr', 'provisioning',
 '{"admin_state": "locked", "nrarfcn": 633984, "bandwidth": 100, "pci": "auto", "tac": "", "cell_id": "", "tx_power": 37, "subcarrier_spacing": 30}'::jsonb,
 100, '中国移动5G NR基站自动开站默认配置模板'),

-- 批量配置模板
('30000000-0005-4000-8000-000000000005', '批量参数配置模板', 'cmcc', 'lte', 'batch_config',
 '{"parameters": []}'::jsonb,
 50, '通用批量参数下发模板'),

-- 固件升级模板
('30000000-0005-4000-8000-000000000006', '固件升级模板', 'cmcc', 'lte', 'firmware_upgrade',
 '{"retry_count": 3, "retry_interval": 300, "download_timeout": 3600}'::jsonb,
 50, '通用固件升级流程模板')

ON CONFLICT DO NOTHING;

-- ============================================================================
-- 6. 运维模板 (ops_templates)
--    标准运维操作流程模板
-- ============================================================================

INSERT INTO ops_templates (id, template_name, description, category, target_device_types, steps, estimated_duration, creator, tags) VALUES
('30000000-0006-4000-8000-000000000001', '设备重启', '安全重启基站设备',
 'maintenance', '["eNB", "gNB"]'::jsonb,
 '[{"step": 1, "name": "检查设备状态", "action": "GetParameterValues", "params": {"path": "Device.DeviceInfo."}},
   {"step": 2, "name": "发送重启命令", "action": "Reboot", "params": {}},
   {"step": 3, "name": "等待设备上线", "action": "WaitInform", "params": {"timeout": 300}}]'::jsonb,
 600, 'system', '["maintenance", "reboot"]'::jsonb),

('30000000-0006-4000-8000-000000000002', '参数采集', '采集设备全量参数',
 'inspection', '["eNB", "gNB"]'::jsonb,
 '[{"step": 1, "name": "采集设备信息", "action": "GetParameterValues", "params": {"path": "Device.DeviceInfo."}},
   {"step": 2, "name": "采集射频参数", "action": "GetParameterValues", "params": {"path": "Device.Services.FAPService.1.CellConfig."}},
   {"step": 3, "name": "采集传输参数", "action": "GetParameterValues", "params": {"path": "Device.Services.FAPService.1.Transport."}}]'::jsonb,
 120, 'system', '["inspection", "parameter"]'::jsonb),

('30000000-0006-4000-8000-000000000003', '固件升级', '执行设备固件升级流程',
 'upgrade', '["eNB", "gNB"]'::jsonb,
 '[{"step": 1, "name": "检查当前版本", "action": "GetParameterValues", "params": {"path": "Device.DeviceInfo.SoftwareVersion"}},
   {"step": 2, "name": "下发固件包", "action": "Download", "params": {"file_type": "1 Firmware Upgrade Image"}},
   {"step": 3, "name": "等待升级完成", "action": "WaitInform", "params": {"timeout": 1800, "event": "7 TRANSFER COMPLETE"}},
   {"step": 4, "name": "验证新版本", "action": "GetParameterValues", "params": {"path": "Device.DeviceInfo.SoftwareVersion"}}]'::jsonb,
 1800, 'system', '["upgrade", "firmware"]'::jsonb),

('30000000-0006-4000-8000-000000000004', '健康检查', '设备运行状态综合检查',
 'inspection', '["eNB", "gNB"]'::jsonb,
 '[{"step": 1, "name": "检查设备信息", "action": "GetParameterValues", "params": {"path": "Device.DeviceInfo."}},
   {"step": 2, "name": "检查小区状态", "action": "GetParameterValues", "params": {"path": "Device.Services.FAPService.1.FAPControl.LTE.AdminState"}},
   {"step": 3, "name": "检查告警状态", "action": "GetParameterValues", "params": {"path": "Device.FaultMgmt.CurrentAlarm."}}]'::jsonb,
 60, 'system', '["inspection", "health"]'::jsonb)

ON CONFLICT DO NOTHING;

-- ============================================================================
-- 7. 报表定义 (report_definitions)
--    标准报表模板
-- ============================================================================

INSERT INTO report_definitions (id, report_name, report_type, description, format, period, kpi_codes, auto_generate, cron_expression, status, creator) VALUES
('30000000-0007-4000-8000-000000000001', '网络KPI日报', 'kpi',
 '汇总全网KPI指标日度统计数据，包含接入成功率、切换成功率、掉话率等核心指标',
 '["pdf", "xlsx"]'::jsonb, 'daily',
 '["RRC_CONN_SETUP_SR", "ERAB_SETUP_SR", "INTRA_FREQ_HO_SR", "CALL_DROP_RATE", "DL_PRB_UTIL"]'::jsonb,
 true, '0 6 * * *', 'active', 'system'),

('30000000-0007-4000-8000-000000000002', '告警统计周报', 'alarm',
 '统计全网告警趋势，按严重级别、类型、区域汇总，识别高频告警和反复告警',
 '["pdf", "xlsx"]'::jsonb, 'weekly',
 '[]'::jsonb,
 true, '0 8 * * 1', 'active', 'system'),

('30000000-0007-4000-8000-000000000003', '网络质量月报', 'comprehensive',
 '月度网络质量综合分析报告，包含KPI趋势、TOP差小区、优化建议',
 '["pdf"]'::jsonb, 'monthly',
 '["RRC_CONN_SETUP_SR", "ERAB_SETUP_SR", "INTRA_FREQ_HO_SR", "INTER_FREQ_HO_SR", "CALL_DROP_RATE", "DL_PRB_UTIL"]'::jsonb,
 true, '0 8 1 * *', 'active', 'system'),

('30000000-0007-4000-8000-000000000004', '设备运行状态报表', 'device',
 '设备在线率、版本分布、告警分布等运维数据日报',
 '["pdf", "xlsx"]'::jsonb, 'daily',
 '[]'::jsonb,
 true, '0 7 * * *', 'active', 'system')

ON CONFLICT DO NOTHING;

-- ============================================================================
-- 8. 配置基线 (config_baselines)
--    默认参数配置基线
-- ============================================================================

INSERT INTO config_baselines (id, baseline_name, description, device_type, version, params, creator, status) VALUES
('30000000-0008-4000-8000-000000000001', 'LTE基本参数基线', 'LTE基站基本射频和传输参数基线配置',
 'eNB', '1.0',
 '[{"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL", "value": "", "description": "下行频点"},
   {"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth", "value": "20", "description": "下行带宽(MHz)"},
   {"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower", "value": "15", "description": "参考信号功率(dBm)"},
   {"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex", "value": "0", "description": "PRACH根序列"},
   {"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.MAC.RACH.NumberOfRAPreambles", "value": "52", "description": "前导码数量"}]'::jsonb,
 'system', 'active'),

('30000000-0008-4000-8000-000000000002', 'NR基本参数基线', '5G NR基站基本射频和传输参数基线配置',
 'gNB', '1.0',
 '[{"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.RF.NRARFCN", "value": "", "description": "NR频点"},
   {"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.RF.DLBandwidth", "value": "100", "description": "下行带宽(MHz)"},
   {"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.RF.SubcarrierSpacing", "value": "30", "description": "子载波间隔(kHz)"},
   {"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.RF.ReferenceSignalPower", "value": "12", "description": "SSB参考信号功率(dBm)"},
   {"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.PHY.PRACH.RootSequenceIndex", "value": "0", "description": "PRACH根序列"}]'::jsonb,
 'system', 'active'),

('30000000-0008-4000-8000-000000000003', 'LTE邻区配置基线', 'LTE基站邻区关系和切换参数基线配置',
 'eNB', '1.0',
 '[{"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.A1ThresholdRSRP", "value": "46", "description": "A1事件RSRP门限"},
   {"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.A2ThresholdRSRP", "value": "38", "description": "A2事件RSRP门限"},
   {"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.A3Offset", "value": "3", "description": "A3事件偏移量(dB)"},
   {"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.TimeToTrigger", "value": "480", "description": "触发时间(ms)"},
   {"path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.Hysteresis", "value": "2", "description": "迟滞(dB)"}]'::jsonb,
 'system', 'draft'),

('30000000-0008-4000-8000-000000000004', 'NR邻区配置基线', '5G NR基站邻区关系和切换参数基线配置',
 'gNB', '1.0',
 '[{"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.Mobility.ConnMode.NR.A3Offset", "value": "3", "description": "A3事件偏移量(dB)"},
   {"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.Mobility.ConnMode.NR.TimeToTrigger", "value": "480", "description": "触发时间(ms)"},
   {"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.Mobility.ConnMode.NR.Hysteresis", "value": "2", "description": "迟滞(dB)"},
   {"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.Mobility.ConnMode.NR.B1ThresholdNR", "value": "40", "description": "B1事件NR门限"},
   {"path": "Device.Services.FAPService.1.CellConfig.NR.RAN.Mobility.ConnMode.NR.B1ThresholdEUTRA", "value": "38", "description": "B1事件LTE门限"}]'::jsonb,
 'system', 'draft')

ON CONFLICT DO NOTHING;

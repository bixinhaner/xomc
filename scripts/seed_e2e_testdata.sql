-- seed_e2e_testdata.sql — E2E 联调测试数据 (Sprint 1 + Sprint 2 + Sprint 3 + Sprint 4 + Sprint 7 + Sprint 8)
-- 使用固定 UUID，方便验证脚本引用
-- 运行前需已执行全部 migrations (make migrate-up)
-- 使用方法: psql "$DSN" -f scripts/seed_e2e_testdata.sql

BEGIN;

-- ============================================================
-- 0. 清理旧的 E2E 测试数据 (幂等，按 FK 依赖逆序)
-- ============================================================

-- Sprint 8 data
DELETE FROM mml_tasks WHERE creator = 'admin' AND task_name LIKE 'E2E%';
DELETE FROM mml_scripts WHERE id::text LIKE 'e2e00013%';
DELETE FROM managed_files WHERE id::text LIKE 'e2e00012%';
DELETE FROM backup_schedules WHERE id::text LIKE 'e2e00011%';
DELETE FROM backup_tasks WHERE id::text LIKE 'e2e00010%';

-- Sprint 7 data
DELETE FROM alarms_active WHERE id::text LIKE 'e2e00099%';

-- Sprint 4 data
DELETE FROM ne_message_logs WHERE device_id::text LIKE 'd0000000%' OR device_sn LIKE 'CMCC-ENB-%';
DELETE FROM system_logs WHERE source IN ('alarm-engine', 'pm-collector', 'config-sync');
DELETE FROM kpi_thresholds WHERE id::text LIKE 'b0000000%';
DELETE FROM alarm_rules WHERE id::text LIKE 'a0000000%';

-- Sprint 3 data
DELETE FROM mr_records WHERE device_id::text LIKE 'e2e00001%';
DELETE FROM mr_files WHERE id::text LIKE 'e2e00008%';
DELETE FROM kpi_values WHERE device_id::text LIKE 'e2e00001%';
DELETE FROM kpi_definitions WHERE name LIKE 'E2E_%';
DELETE FROM pm_counters WHERE device_id::text LIKE 'e2e00001%';
DELETE FROM audit_logs WHERE id::text LIKE 'e2e00009%';

-- Sprint 2 data
DELETE FROM device_group_members WHERE group_id::text LIKE 'e2e00007%';
DELETE FROM device_groups WHERE id::text LIKE 'e2e00007%';
DELETE FROM upgrade_tasks WHERE id::text LIKE 'e2e00005%';
DELETE FROM upgrade_tasks WHERE firmware_id::text LIKE 'e2e00004%';
DELETE FROM firmware_versions WHERE id::text LIKE 'e2e00004%';
DELETE FROM config_templates WHERE id::text LIKE 'e2e00003%';
DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'e2e-%');
DELETE FROM users WHERE username LIKE 'e2e-%';

-- Sprint 1 data
DELETE FROM alarms_active WHERE id::text LIKE 'e2e00002%';
DELETE FROM devices WHERE serial_number IN ('TEST-SN-001','TEST-SN-002','TEST-SN-003','TEST-SN-004','TEST-SN-005');

-- ============================================================
-- 1. 测试设备 (5 台，覆盖不同运营商/制式/状态)
-- ============================================================

INSERT INTO devices (
    id, serial_number, oui, product_class, manufacturer, model_name,
    carrier, technology, status, firmware_version, ip_address,
    connection_request_url, inform_interval, site_name, site_id,
    latitude, longitude, created_at, updated_at
) VALUES
-- Device 1: CMCC LTE, active
(
    'e2e00001-0000-0000-0000-000000000001',
    'TEST-SN-001', '00A0C6', 'FAP-LTE-100', 'Huawei', 'eLTE-230',
    'cmcc', 'lte', 'active', 'V200R003C10', '10.0.1.101',
    'http://10.0.1.101:7547/ConnectionRequest', 300, 'Beijing-Site-01', 'BJ-S01',
    39.9042, 116.4074, NOW() - INTERVAL '30 days', NOW() - INTERVAL '5 minutes'
),
-- Device 2: CMCC NR, active
(
    'e2e00001-0000-0000-0000-000000000002',
    'TEST-SN-002', '00A0C6', 'gNB-100', 'Huawei', 'AAU5613',
    'cmcc', 'nr', 'active', 'V100R018C10', '10.0.1.102',
    'http://10.0.1.102:7547/ConnectionRequest', 300, 'Beijing-Site-02', 'BJ-S02',
    39.9142, 116.4174, NOW() - INTERVAL '25 days', NOW() - INTERVAL '10 minutes'
),
-- Device 3: CTCC LTE, offline
(
    'e2e00001-0000-0000-0000-000000000003',
    'TEST-SN-003', '001E4F', 'FAP-LTE-200', 'ZTE', 'ZXSDR-B8200',
    'ctcc', 'lte', 'offline', 'V4.16.30P4', NULL,
    '', 600, 'Shanghai-Site-01', 'SH-S01',
    31.2304, 121.4737, NOW() - INTERVAL '60 days', NOW() - INTERVAL '2 days'
),
-- Device 4: CUCC NR, registered (newly registered, not yet active)
(
    'e2e00001-0000-0000-0000-000000000004',
    'TEST-SN-004', '64700E', 'gNB-200', 'Ericsson', 'AIR6488',
    'cucc', 'nr', 'registered', 'CXP9024418/12_R57A', '10.0.3.201',
    'http://10.0.3.201:7547/ConnectionRequest', 300, 'Guangzhou-Site-01', 'GZ-S01',
    23.1291, 113.2644, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'
),
-- Device 5: CMCC LTE, maintenance
(
    'e2e00001-0000-0000-0000-000000000005',
    'TEST-SN-005', '00A0C6', 'FAP-LTE-100', 'Huawei', 'eLTE-230',
    'cmcc', 'lte', 'maintenance', 'V200R003C10', '10.0.4.101',
    'http://10.0.4.101:7547/ConnectionRequest', 300, 'Shenzhen-Site-01', 'SZ-S01',
    22.5431, 114.0579, NOW() - INTERVAL '90 days', NOW() - INTERVAL '1 day'
);

-- ============================================================
-- 2. 测试告警 (8 条，覆盖 4 种严重等级，含 active 和 acknowledged)
-- ============================================================

INSERT INTO alarms_active (
    id, device_id, device_sn, carrier, severity, alarm_type, alarm_code,
    description, status, raised_at, acknowledged_at, acknowledged_by,
    additional_info, created_at, updated_at
) VALUES
-- Alarm 1: Critical, active (Device 1)
(
    'e2e00002-0000-0000-0000-000000000001',
    'e2e00001-0000-0000-0000-000000000001', 'TEST-SN-001', 'cmcc',
    1, 'CommunicationFailure', 'ALM-10001',
    'S1 link failure detected on eNB', 'active',
    NOW() - INTERVAL '2 hours', NULL, '',
    '{"location": "S1-Interface", "affected_cells": "3"}'::jsonb,
    NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours'
),
-- Alarm 2: Critical, acknowledged (Device 2)
(
    'e2e00002-0000-0000-0000-000000000002',
    'e2e00001-0000-0000-0000-000000000002', 'TEST-SN-002', 'cmcc',
    1, 'PowerFailure', 'ALM-10002',
    'Primary power supply failure on gNB', 'acknowledged',
    NOW() - INTERVAL '4 hours', NOW() - INTERVAL '3 hours', 'admin',
    '{"location": "PSU-A"}'::jsonb,
    NOW() - INTERVAL '4 hours', NOW() - INTERVAL '3 hours'
),
-- Alarm 3: Major, active (Device 1)
(
    'e2e00002-0000-0000-0000-000000000003',
    'e2e00001-0000-0000-0000-000000000001', 'TEST-SN-001', 'cmcc',
    2, 'HighCPUUtilization', 'ALM-20001',
    'CPU utilization exceeded 90% threshold', 'active',
    NOW() - INTERVAL '1 hour', NULL, '',
    '{"cpu_usage": "93%"}'::jsonb,
    NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour'
),
-- Alarm 4: Major, active (Device 3)
(
    'e2e00002-0000-0000-0000-000000000004',
    'e2e00001-0000-0000-0000-000000000003', 'TEST-SN-003', 'ctcc',
    2, 'LinkDown', 'ALM-20002',
    'Backhaul link down on eNB', 'active',
    NOW() - INTERVAL '2 days', NULL, '',
    '{"interface": "eth0"}'::jsonb,
    NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'
),
-- Alarm 5: Minor, active (Device 2)
(
    'e2e00002-0000-0000-0000-000000000005',
    'e2e00001-0000-0000-0000-000000000002', 'TEST-SN-002', 'cmcc',
    3, 'HighTemperature', 'ALM-30001',
    'Temperature exceeded warning threshold', 'active',
    NOW() - INTERVAL '30 minutes', NULL, '',
    '{"temperature": "72C", "threshold": "70C"}'::jsonb,
    NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes'
),
-- Alarm 6: Minor, acknowledged (Device 4)
(
    'e2e00002-0000-0000-0000-000000000006',
    'e2e00001-0000-0000-0000-000000000004', 'TEST-SN-004', 'cucc',
    3, 'ConfigMismatch', 'ALM-30002',
    'Configuration parameter mismatch detected', 'acknowledged',
    NOW() - INTERVAL '6 hours', NOW() - INTERVAL '5 hours', 'operator1',
    '{}'::jsonb,
    NOW() - INTERVAL '6 hours', NOW() - INTERVAL '5 hours'
),
-- Alarm 7: Warning, active (Device 5)
(
    'e2e00002-0000-0000-0000-000000000007',
    'e2e00001-0000-0000-0000-000000000005', 'TEST-SN-005', 'cmcc',
    4, 'MaintenanceMode', 'ALM-40001',
    'Device entered maintenance mode', 'active',
    NOW() - INTERVAL '1 day', NULL, '',
    '{}'::jsonb,
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'
),
-- Alarm 8: Warning, active (Device 1)
(
    'e2e00002-0000-0000-0000-000000000008',
    'e2e00001-0000-0000-0000-000000000001', 'TEST-SN-001', 'cmcc',
    4, 'LicenseExpiring', 'ALM-40002',
    'Device license expiring in 30 days', 'active',
    NOW() - INTERVAL '15 minutes', NULL, '',
    '{"expires_in_days": "30"}'::jsonb,
    NOW() - INTERVAL '15 minutes', NOW() - INTERVAL '15 minutes'
);

-- ============================================================
-- 3. 配置模板 (3 条，覆盖不同模板类型)
-- ============================================================

INSERT INTO config_templates (
    id, name, carrier, technology, product_class, template_type,
    parameters, priority, version, active, description,
    created_at, updated_at
) VALUES
-- Template 1: LTE batch config, active
(
    'e2e00003-0000-0000-0000-000000000001',
    'LTE Basic Config', 'cmcc', 'lte', 'FAP-LTE-100', 'batch_config',
    '{"params": [{"name": "AdminState", "value": "1"}, {"name": "MaxTxPower", "value": "30"}]}'::jsonb,
    1, 1, true, 'Basic LTE small cell configuration template',
    NOW() - INTERVAL '20 days', NOW() - INTERVAL '1 day'
),
-- Template 2: NR provisioning, active
(
    'e2e00003-0000-0000-0000-000000000002',
    'NR Provisioning', 'cmcc', 'nr', 'gNB-100', 'provisioning',
    '{"params": [{"name": "NRPCI", "value": "100"}, {"name": "DLBandwidth", "value": "100MHz"}]}'::jsonb,
    2, 1, true, 'NR gNB provisioning template',
    NOW() - INTERVAL '15 days', NOW() - INTERVAL '2 days'
),
-- Template 3: LTE firmware upgrade, inactive
(
    'e2e00003-0000-0000-0000-000000000003',
    'Firmware Upgrade LTE', 'ctcc', 'lte', 'FAP-LTE-200', 'firmware_upgrade',
    '{"params": [{"name": "DownloadTimeout", "value": "3600"}, {"name": "RebootDelay", "value": "60"}]}'::jsonb,
    0, 1, false, 'LTE firmware upgrade template (inactive)',
    NOW() - INTERVAL '30 days', NOW() - INTERVAL '10 days'
);

-- ============================================================
-- 4. 固件版本 (2 条)
-- ============================================================

INSERT INTO firmware_versions (
    id, carrier, product_class, version, file_name, file_size,
    minio_path, compatible_oui, release_notes, status,
    created_at, updated_at
) VALUES
-- Firmware 1: LTE firmware
(
    'e2e00004-0000-0000-0000-000000000001',
    'cmcc', 'FAP-LTE-100', 'V200R003C11',
    'eLTE-230_V200R003C11.bin', 52428800,
    'firmware/cmcc/FAP-LTE-100/V200R003C11.bin',
    '["00A0C6"]'::jsonb, 'Bug fixes and stability improvements',
    'active', NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days'
),
-- Firmware 2: NR firmware (has upgrade tasks referencing it)
(
    'e2e00004-0000-0000-0000-000000000002',
    'cmcc', 'gNB-100', 'V100R019C10',
    'AAU5613_V100R019C10.bin', 104857600,
    'firmware/cmcc/gNB-100/V100R019C10.bin',
    '["00A0C6"]'::jsonb, 'New 5G NR features and performance enhancements',
    'active', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'
),
-- Firmware 3: standalone (no upgrade tasks, safe to delete)
(
    'e2e00004-0000-0000-0000-000000000003',
    'ctcc', 'FAP-LTE-200', 'V4.16.31P1',
    'ZXSDR-B8200_V4.16.31P1.bin', 31457280,
    'firmware/ctcc/FAP-LTE-200/V4.16.31P1.bin',
    '["001E4F"]'::jsonb, 'Minor patch release',
    'active', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'
);

-- ============================================================
-- 5. 升级任务 (2 条，关联固件和设备)
-- ============================================================

INSERT INTO upgrade_tasks (
    id, device_id, firmware_id, batch_id, status,
    error_message, retry_count, max_retries,
    started_at, completed_at, created_at, updated_at
) VALUES
-- Task 1: completed
(
    'e2e00005-0000-0000-0000-000000000001',
    'e2e00001-0000-0000-0000-000000000001',
    'e2e00004-0000-0000-0000-000000000001',
    'e2e00005-ba00-0000-0000-000000000001',
    'completed', NULL, 0, 3,
    NOW() - INTERVAL '8 days', NOW() - INTERVAL '8 days' + INTERVAL '15 minutes',
    NOW() - INTERVAL '8 days', NOW() - INTERVAL '8 days' + INTERVAL '15 minutes'
),
-- Task 2: pending
(
    'e2e00005-0000-0000-0000-000000000002',
    'e2e00001-0000-0000-0000-000000000002',
    'e2e00004-0000-0000-0000-000000000002',
    'e2e00005-ba00-0000-0000-000000000001',
    'pending', NULL, 0, 3,
    NULL, NULL,
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'
);

-- ============================================================
-- 6. 设备分组 (3 条，含层级关系 + 设备绑定)
-- ============================================================

INSERT INTO device_groups (
    id, name, parent_id, carrier, description, sort_order,
    created_at, updated_at
) VALUES
-- Group 1: Beijing Region (root)
(
    'e2e00007-0000-0000-0000-000000000001',
    'Beijing Region', NULL, 'cmcc', 'Beijing metropolitan area device group',
    1, NOW() - INTERVAL '60 days', NOW() - INTERVAL '1 day'
),
-- Group 2: Beijing-Haidian (child of Beijing)
(
    'e2e00007-0000-0000-0000-000000000002',
    'Beijing-Haidian', 'e2e00007-0000-0000-0000-000000000001', 'cmcc',
    'Haidian district sub-group', 1,
    NOW() - INTERVAL '55 days', NOW() - INTERVAL '1 day'
),
-- Group 3: Shanghai Region (root)
(
    'e2e00007-0000-0000-0000-000000000003',
    'Shanghai Region', NULL, 'ctcc', 'Shanghai metropolitan area device group',
    2, NOW() - INTERVAL '50 days', NOW() - INTERVAL '2 days'
);

-- Link devices to groups
INSERT INTO device_group_members (group_id, device_id, added_at) VALUES
('e2e00007-0000-0000-0000-000000000002', 'e2e00001-0000-0000-0000-000000000001', NOW() - INTERVAL '30 days'),
('e2e00007-0000-0000-0000-000000000002', 'e2e00001-0000-0000-0000-000000000002', NOW() - INTERVAL '30 days'),
('e2e00007-0000-0000-0000-000000000003', 'e2e00001-0000-0000-0000-000000000003', NOW() - INTERVAL '25 days');

-- ============================================================
-- 7. PM 性能计数器 (Sprint 3 — 8 条，覆盖 2 台设备 + 2 个计数组 + 2 个时间窗口)
-- ============================================================

INSERT INTO pm_counters (time, device_id, cell_id, counter_group, counter_name, counter_value, granularity) VALUES
-- Device 1, RRC counters, 2 hours ago
(NOW() - INTERVAL '2 hours', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'RRC', 'RRC_CONN_ATTEMPT', 1500, 15),
(NOW() - INTERVAL '2 hours', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'RRC', 'RRC_CONN_SUCCESS', 1485, 15),
-- Device 1, RRC counters, 1 hour ago
(NOW() - INTERVAL '1 hour', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'RRC', 'RRC_CONN_ATTEMPT', 1600, 15),
(NOW() - INTERVAL '1 hour', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'RRC', 'RRC_CONN_SUCCESS', 1592, 15),
-- Device 1, ERAB counters
(NOW() - INTERVAL '2 hours', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'ERAB', 'ERAB_SETUP_ATTEMPT', 1200, 15),
(NOW() - INTERVAL '2 hours', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'ERAB', 'ERAB_SETUP_SUCCESS', 1188, 15),
-- Device 2, RRC counters
(NOW() - INTERVAL '1 hour', 'e2e00001-0000-0000-0000-000000000002', 'CELL-002-1', 'RRC', 'RRC_CONN_ATTEMPT', 800, 15),
(NOW() - INTERVAL '1 hour', 'e2e00001-0000-0000-0000-000000000002', 'CELL-002-1', 'RRC', 'RRC_CONN_SUCCESS', 796, 15);

-- ============================================================
-- 8. KPI 定义 (Sprint 3 — 2 条)
-- ============================================================

INSERT INTO kpi_definitions (name, display_name, formula, unit, category, carrier, technology, counters) VALUES
('E2E_RRC_SR', 'RRC Setup Success Rate', 'RRC_CONN_SUCCESS / RRC_CONN_ATTEMPT * 100', '%', 'accessibility', 'cmcc', 'lte',
 '["RRC_CONN_ATTEMPT","RRC_CONN_SUCCESS"]'::jsonb),
('E2E_ERAB_SR', 'E-RAB Setup Success Rate', 'ERAB_SETUP_SUCCESS / ERAB_SETUP_ATTEMPT * 100', '%', 'accessibility', 'cmcc', 'lte',
 '["ERAB_SETUP_ATTEMPT","ERAB_SETUP_SUCCESS"]'::jsonb);

-- ============================================================
-- 9. KPI 计算值 (Sprint 3 — 4 条)
-- ============================================================

INSERT INTO kpi_values (time, device_id, cell_id, kpi_name, kpi_value, carrier, technology) VALUES
(NOW() - INTERVAL '2 hours', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'E2E_RRC_SR', 99.0, 'cmcc', 'lte'),
(NOW() - INTERVAL '1 hour',  'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'E2E_RRC_SR', 99.5, 'cmcc', 'lte'),
(NOW() - INTERVAL '2 hours', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'E2E_ERAB_SR', 99.0, 'cmcc', 'lte'),
(NOW() - INTERVAL '1 hour',  'e2e00001-0000-0000-0000-000000000002', 'CELL-002-1', 'E2E_RRC_SR', 99.5, 'cmcc', 'nr');

-- ============================================================
-- 10. MR 文件元数据 (Sprint 3 — 2 条)
-- ============================================================

INSERT INTO mr_files (id, device_id, device_sn, carrier, mr_type, file_name, file_size, collect_time, minio_path, parsed, record_count, created_at) VALUES
('e2e00008-0000-0000-0000-000000000001', 'e2e00001-0000-0000-0000-000000000001', 'TEST-SN-001', 'cmcc', 'MRO',
 'MRO_TEST-SN-001_20260307.xml', 102400, NOW() - INTERVAL '1 hour',
 'mr/cmcc/TEST-SN-001/MRO_20260307.xml', true, 50, NOW() - INTERVAL '1 hour'),
('e2e00008-0000-0000-0000-000000000002', 'e2e00001-0000-0000-0000-000000000002', 'TEST-SN-002', 'cmcc', 'MRS',
 'MRS_TEST-SN-002_20260307.xml', 51200, NOW() - INTERVAL '2 hours',
 'mr/cmcc/TEST-SN-002/MRS_20260307.xml', true, 30, NOW() - INTERVAL '2 hours');

-- ============================================================
-- 11. MR 解析记录 (Sprint 3 — 4 条)
-- ============================================================

INSERT INTO mr_records (time, file_id, device_id, cell_id, mr_type, measurement_data) VALUES
(NOW() - INTERVAL '1 hour', 'e2e00008-0000-0000-0000-000000000001', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-1', 'MRO',
 '{"RSRP": -85.5, "RSRQ": -9.2, "SINR": 12.3}'::jsonb),
(NOW() - INTERVAL '1 hour', 'e2e00008-0000-0000-0000-000000000001', 'e2e00001-0000-0000-0000-000000000001', 'CELL-001-2', 'MRO',
 '{"RSRP": -92.1, "RSRQ": -11.5, "SINR": 8.7}'::jsonb),
(NOW() - INTERVAL '2 hours', 'e2e00008-0000-0000-0000-000000000002', 'e2e00001-0000-0000-0000-000000000002', 'CELL-002-1', 'MRS',
 '{"RSRP": -78.3, "RSRQ": -7.8, "SINR": 15.6}'::jsonb),
(NOW() - INTERVAL '2 hours', 'e2e00008-0000-0000-0000-000000000002', 'e2e00001-0000-0000-0000-000000000002', 'CELL-002-2', 'MRS',
 '{"RSRP": -88.9, "RSRQ": -10.1, "SINR": 10.2}'::jsonb);

-- ============================================================
-- 12. 审计日志 (Sprint 3 — 3 条，不同时间戳便于时间范围过滤测试)
-- ============================================================

INSERT INTO audit_logs (id, user_id, username, action, resource, resource_id, details, ip_address, created_at) VALUES
('e2e00009-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 'admin', 'login', 'auth', '',
 '{}'::jsonb, '10.0.0.1'::inet, NOW() - INTERVAL '3 hours'),
('e2e00009-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', 'admin', 'create', 'users', 'new-user-001',
 '{"username":"testuser"}'::jsonb, '10.0.0.1'::inet, NOW() - INTERVAL '2 hours'),
('e2e00009-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000001', 'admin', 'update', 'devices', 'e2e00001-0000-0000-0000-000000000001',
 '{"field":"firmware"}'::jsonb, '10.0.0.1'::inet, NOW() - INTERVAL '1 hour');

-- ============================================================
-- 13. Alarm Rules (Sprint 4 — 3 条)
-- ============================================================

-- Sprint 4: Alarm Rules
INSERT INTO alarm_rules (id, name, description, alarm_code, severity, condition_type, condition_config, action_type, action_config, carrier, technology, enabled, created_at, updated_at)
VALUES
  ('a0000000-0000-0000-0000-000000000001', 'High CPU Alert', 'CPU usage exceeds threshold', 'CPU_HIGH', 1, 'threshold', '{"metric":"cpu_usage","operator":"gt","value":90}', 'notification', '{"channel":"email","recipients":["admin@omc.com"]}', 'cmcc', 'LTE', true, NOW(), NOW()),
  ('a0000000-0000-0000-0000-000000000002', 'Link Down Alert', 'Network link goes down', 'LINK_DOWN', 2, 'event', '{"event_type":"link_status","value":"down"}', 'notification', '{"channel":"sms","recipients":["+8613800138000"]}', 'cmcc', 'NR', true, NOW(), NOW()),
  ('a0000000-0000-0000-0000-000000000003', 'Low Signal Quality', 'Signal quality below minimum', 'SIGNAL_LOW', 3, 'threshold', '{"metric":"sinr","operator":"lt","value":-5}', 'auto_heal', '{"action":"reboot"}', 'ctcc', 'LTE', false, NOW(), NOW());

-- ============================================================
-- 14. KPI Thresholds (Sprint 4 — 3 条)
-- ============================================================

-- Sprint 4: KPI Thresholds
INSERT INTO kpi_thresholds (id, kpi_name, carrier, technology, warning_threshold, minor_threshold, major_threshold, critical_threshold, comparison, enabled, description, created_at, updated_at)
VALUES
  ('b0000000-0000-0000-0000-000000000001', 'rrc_succ_rate', 'cmcc', 'LTE', 95.0, 90.0, 85.0, 80.0, 'lt', true, 'RRC success rate thresholds', NOW(), NOW()),
  ('b0000000-0000-0000-0000-000000000002', 'erab_succ_rate', 'cmcc', 'LTE', 98.0, 95.0, 90.0, 85.0, 'lt', true, 'E-RAB success rate thresholds', NOW(), NOW()),
  ('b0000000-0000-0000-0000-000000000003', 'handover_succ_rate', 'ctcc', 'NR', 97.0, 94.0, 90.0, 85.0, 'lt', false, 'Handover success rate thresholds', NOW(), NOW());

-- ============================================================
-- 15. System Logs (Sprint 4 — 3 条)
-- ============================================================

-- Sprint 4: System Logs
INSERT INTO system_logs (id, level, source, message, details, created_at)
VALUES
  (gen_random_uuid(), 'ERROR', 'alarm-engine', 'Failed to process alarm batch', '{"batch_size":50,"error":"timeout"}', NOW() - INTERVAL '2 hours'),
  (gen_random_uuid(), 'WARN', 'pm-collector', 'Counter collection delayed', '{"device_count":10,"delay_ms":5000}', NOW() - INTERVAL '1 hour'),
  (gen_random_uuid(), 'INFO', 'config-sync', 'Configuration push completed', '{"device_id":"d0000000-0000-0000-0000-000000000001","params":5}', NOW() - INTERVAL '30 minutes');

-- ============================================================
-- 16. NE Message Logs (Sprint 4 — 3 条)
-- ============================================================

-- Sprint 4: NE Message Logs
INSERT INTO ne_message_logs (id, device_id, device_sn, device_name, message_type, direction, content, created_at)
VALUES
  (gen_random_uuid(), 'd0000000-0000-0000-0000-000000000001', 'CMCC-ENB-001', 'eNB-BJ-001', 'Inform', 'inbound', '{"event":"2 PERIODIC"}', NOW() - INTERVAL '1 hour'),
  (gen_random_uuid(), 'd0000000-0000-0000-0000-000000000001', 'CMCC-ENB-001', 'eNB-BJ-001', 'GetParameterValuesResponse', 'inbound', '{"params":["Device.DeviceInfo.SoftwareVersion"]}', NOW() - INTERVAL '45 minutes'),
  (gen_random_uuid(), 'd0000000-0000-0000-0000-000000000002', 'CMCC-ENB-002', 'eNB-SH-001', 'SetParameterValues', 'outbound', '{"params":[{"name":"Device.ManagementServer.PeriodicInformInterval","value":"300"}]}', NOW() - INTERVAL '30 minutes');

-- ============================================================
-- 17. Alarm Trend Test Data (Sprint 7 — 3 条，不同日期用于告警趋势聚合验证)
-- ============================================================

INSERT INTO alarms_active (
    id, device_id, device_sn, carrier, severity, alarm_type, alarm_code,
    description, status, raised_at, acknowledged_at, acknowledged_by,
    additional_info, created_at, updated_at
) VALUES
-- Sprint 7 Alarm 1: today (Critical)
(
    'e2e00099-0000-0000-0000-000000000001',
    'e2e00001-0000-0000-0000-000000000001', 'TEST-SN-001', 'cmcc',
    1, 'CommunicationFailure', 'ALM-S7-001',
    'E2E Sprint 7 alarm trend test 1 — today', 'active',
    NOW(), NULL, '',
    '{"test": "sprint7-trend"}'::jsonb,
    NOW(), NOW()
),
-- Sprint 7 Alarm 2: yesterday (Minor)
(
    'e2e00099-0000-0000-0000-000000000002',
    'e2e00001-0000-0000-0000-000000000001', 'TEST-SN-001', 'cmcc',
    3, 'HighTemperature', 'ALM-S7-002',
    'E2E Sprint 7 alarm trend test 2 — yesterday', 'active',
    NOW() - INTERVAL '1 day', NULL, '',
    '{"test": "sprint7-trend"}'::jsonb,
    NOW(), NOW()
),
-- Sprint 7 Alarm 3: 2 days ago (Major)
(
    'e2e00099-0000-0000-0000-000000000003',
    'e2e00001-0000-0000-0000-000000000002', 'TEST-SN-002', 'cmcc',
    2, 'HighCPUUtilization', 'ALM-S7-003',
    'E2E Sprint 7 alarm trend test 3 — 2 days ago', 'active',
    NOW() - INTERVAL '2 days', NULL, '',
    '{"test": "sprint7-trend"}'::jsonb,
    NOW(), NOW()
);

-- ============================================================
-- 18. Backup Tasks (Sprint 8)
-- ============================================================

INSERT INTO backup_tasks (id, task_type, target_type, target_ids, status, progress, created_at, updated_at) VALUES
('e2e00010-0000-0000-0000-000000000001', 'full', 'device', '["TEST00001"]', 'completed', 100, NOW() - INTERVAL '1 day', NOW()),
('e2e00010-0000-0000-0000-000000000002', 'config_only', 'device', '["TEST00001"]', 'pending', 0, NOW(), NOW());

-- ============================================================
-- 19. Backup Schedules (Sprint 8)
-- ============================================================

INSERT INTO backup_schedules (id, name, cron_expr, enabled, task_type, created_at, updated_at) VALUES
('e2e00011-0000-0000-0000-000000000001', 'Daily Full Backup', '0 2 * * *', true, 'full', NOW(), NOW()),
('e2e00011-0000-0000-0000-000000000002', 'Weekly Config Backup', '0 3 * * 0', false, 'config_only', NOW(), NOW());

-- ============================================================
-- 20. Managed Files (Sprint 8)
-- ============================================================

INSERT INTO managed_files (id, file_name, file_type, file_size, minio_path, status, description, created_at, updated_at) VALUES
('e2e00012-0000-0000-0000-000000000001', 'router-config.xml', 'config', 1024, 'managed-files/config/test.xml', 'ready', 'E2E config file', NOW(), NOW()),
('e2e00012-0000-0000-0000-000000000002', 'device.log', 'log', 2048, 'managed-files/log/device.log', 'ready', 'E2E log file', NOW(), NOW()),
('e2e00012-0000-0000-0000-000000000003', 'firmware-v2.bin', 'firmware', 10240, 'managed-files/firmware/fw.bin', 'ready', 'E2E firmware', NOW(), NOW());

-- ============================================================
-- 21. MML Scripts (Sprint 8)
-- ============================================================

INSERT INTO mml_scripts (id, script_name, description, content, device_type, creator, created_at, updated_at) VALUES
('e2e00013-0000-0000-0000-000000000001', 'Device Health Check', 'Check device status', 'LST DEVPARAM;RST_DEV', 'router', 'admin', NOW(), NOW());

COMMIT;

-- Verify counts
SELECT 'devices' AS entity, COUNT(*) AS count FROM devices WHERE id::text LIKE 'e2e00001%'
UNION ALL
SELECT 'alarms_active', COUNT(*) FROM alarms_active WHERE id::text LIKE 'e2e00002%'
UNION ALL
SELECT 'config_templates', COUNT(*) FROM config_templates WHERE id::text LIKE 'e2e00003%'
UNION ALL
SELECT 'firmware_versions', COUNT(*) FROM firmware_versions WHERE id::text LIKE 'e2e00004%'
UNION ALL
SELECT 'upgrade_tasks', COUNT(*) FROM upgrade_tasks WHERE id::text LIKE 'e2e00005%'
UNION ALL
SELECT 'device_groups', COUNT(*) FROM device_groups WHERE id::text LIKE 'e2e00007%'
UNION ALL
SELECT 'pm_counters', COUNT(*) FROM pm_counters WHERE device_id::text LIKE 'e2e00001%'
UNION ALL
SELECT 'kpi_definitions', COUNT(*) FROM kpi_definitions WHERE name LIKE 'E2E_%'
UNION ALL
SELECT 'kpi_values', COUNT(*) FROM kpi_values WHERE device_id::text LIKE 'e2e00001%'
UNION ALL
SELECT 'mr_files', COUNT(*) FROM mr_files WHERE id::text LIKE 'e2e00008%'
UNION ALL
SELECT 'mr_records', COUNT(*) FROM mr_records WHERE device_id::text LIKE 'e2e00001%'
UNION ALL
SELECT 'audit_logs', COUNT(*) FROM audit_logs WHERE id::text LIKE 'e2e00009%'
UNION ALL
SELECT 'alarm_rules', COUNT(*) FROM alarm_rules WHERE id::text LIKE 'a0000000%'
UNION ALL
SELECT 'kpi_thresholds', COUNT(*) FROM kpi_thresholds WHERE id::text LIKE 'b0000000%'
UNION ALL
SELECT 'system_logs', COUNT(*) FROM system_logs WHERE source IN ('alarm-engine', 'pm-collector', 'config-sync')
UNION ALL
SELECT 'ne_message_logs', COUNT(*) FROM ne_message_logs WHERE device_sn LIKE 'CMCC-ENB-%'
UNION ALL
SELECT 'alarms_sprint7', COUNT(*) FROM alarms_active WHERE id::text LIKE 'e2e00099%'
UNION ALL
SELECT 'backup_tasks', COUNT(*) FROM backup_tasks WHERE id::text LIKE 'e2e00010%'
UNION ALL
SELECT 'backup_schedules', COUNT(*) FROM backup_schedules WHERE id::text LIKE 'e2e00011%'
UNION ALL
SELECT 'managed_files', COUNT(*) FROM managed_files WHERE id::text LIKE 'e2e00012%'
UNION ALL
SELECT 'mml_scripts', COUNT(*) FROM mml_scripts WHERE id::text LIKE 'e2e00013%';

-- seed_e2e_testdata.sql — E2E 联调测试数据
-- 使用固定 UUID，方便验证脚本引用
-- 运行前需已执行全部 migrations (make migrate-up)
-- 使用方法: psql "$DSN" -f scripts/seed_e2e_testdata.sql

BEGIN;

-- ============================================================
-- 0. 清理旧的 E2E 测试数据 (幂等)
-- ============================================================

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

COMMIT;

-- Verify counts
SELECT 'devices' AS entity, COUNT(*) AS count FROM devices WHERE id::text LIKE 'e2e00001%'
UNION ALL
SELECT 'alarms_active', COUNT(*) FROM alarms_active WHERE id::text LIKE 'e2e00002%';

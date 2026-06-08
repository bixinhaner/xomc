-- ============================================================
-- seed_test_data.sql — 批量测试数据
-- 使用 generate_series 高效生成，适合开发环境
--
-- 数据量级:
--   小 (10-50):  sites, firmware, templates, rules, ops, reports
--   中 (200-500): devices, device_info, alarms_active, mr_files, audit_logs
--   大 (5K-50K):  device_parameters, pm_metrics (counter+kpi), alarms_history, mr_records
--
-- 前置条件: migrations 000001-000011 已全部执行（含种子数据）
-- 使用方法:  psql "$DSN" -f scripts/seed_test_data.sql
-- ============================================================

BEGIN;

-- ============================================================
-- 0. 幂等清理（按 FK 逆序）
-- ============================================================
DELETE FROM mr_records              WHERE device_id::text LIKE 'a0000000%';
DELETE FROM mr_files                WHERE device_id::text LIKE 'a0000000%';
-- T-0164-P3 / G3：pm_counters + kpi_values 合入 pm_metrics
-- T-0164-P3 fix: 设备唯一标识 (device_oui, device_sn) 双键
DELETE FROM pm_metrics              WHERE device_sn LIKE 'TD-%';
DELETE FROM pm_files                WHERE device_id::text LIKE 'a0000000%';
DELETE FROM alarms_history          WHERE device_id::text LIKE 'a0000000%';
DELETE FROM alarms_active           WHERE device_id::text LIKE 'a0000000%';
DELETE FROM ne_message_logs         WHERE device_id::text LIKE 'a0000000%';
DELETE FROM audit_logs              WHERE id::text LIKE 'a0000000%';
DELETE FROM system_logs             WHERE id::text LIKE 'a0000000%';
DELETE FROM device_parameters       WHERE device_id::text LIKE 'a0000000%';
DELETE FROM device_info             WHERE device_id::text LIKE 'a0000000%';
DELETE FROM device_group_members    WHERE device_id::text LIKE 'a0000000%';
DELETE FROM devices                 WHERE id::text LIKE 'a0000000%';
DELETE FROM device_tasks            WHERE device_sn LIKE 'TD-%';
DELETE FROM upgrade_tasks           WHERE id::text LIKE 'a0000000%';
DELETE FROM topo_edges              WHERE source_id::text LIKE 'a0000000%' OR target_id::text LIKE 'a0000000%';
DELETE FROM topo_nodes              WHERE id::text LIKE 'a0000000%';
DELETE FROM sites                   WHERE id::text LIKE 'a0000000%';
DELETE FROM ops_tasks               WHERE id::text LIKE 'a0000000%';
DELETE FROM ops_templates           WHERE id::text LIKE 'a0000000%';
DELETE FROM ops_command_records     WHERE id::text LIKE 'a0000000%';
DELETE FROM report_records          WHERE id::text LIKE 'a0000000%';
DELETE FROM backup_tasks            WHERE id::text LIKE 'a0000000%';
DELETE FROM managed_files           WHERE id::text LIKE 'a0000000%';
DELETE FROM firmware_versions       WHERE id::text LIKE 'a0000000%';
DELETE FROM alarm_rules             WHERE id::text LIKE 'a0000000%';
DELETE FROM config_baselines        WHERE id::text LIKE 'a0000000%';
DELETE FROM config_templates        WHERE id::text LIKE 'a0000000%';
-- device_rules / licenses tables removed from schema — DELETEs dropped

-- ============================================================
-- 1. 设备 (500台: cmcc=200, ctcc=150, cucc=150, lte/nr混合)
-- ============================================================
-- 使用确定性 UUID: a0000000-{i:04d}-4000-8000-000000000001
-- 其中 i 的千位编码 carrier+tech:
--   0xxx=cmcc-lte, 1xxx=cmcc-nr, 2xxx=ctcc-lte, 3xxx=ctcc-nr, 4xxx=cucc-lte, 5xxx=cucc-nr

INSERT INTO devices (id, serial_number, oui, product_class, manufacturer, model_name,
    carrier, technology, lifecycle_state, firmware_version, ip_address,
    connection_request_url, inform_interval, site_name, site_id,
    latitude, longitude, last_inform_at, created_at, updated_at)
SELECT
    format('a0000000-%s-4000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    'TD-' || c.carrier || '-' || lpad(i::text, 4, '0'),
    c.oui,
    c.product_class,
    c.manufacturer,
    c.model_name,
    c.carrier,
    c.tech,
    (ARRAY['commissioned','commissioned','commissioned','commissioned','registered','discovered'])[1 + floor(random()*6)::int],
    c.fw_ver,
    format('10.%s.%s.%s', (10 + floor(random()*240))::int, (1 + floor(random()*254))::int, (1 + floor(random()*254))::int)::inet,
    NULL,  -- connection_request_url
    300,
    'Site-' || c.carrier || '-' || lpad((1 + floor(random()*20))::text, 2, '0'),
    c.carrier || '-S' || lpad((1 + floor(random()*20))::text, 2, '0'),
    22.0 + random()*16,   -- latitude 22~38 覆盖中国主要区域
    100.0 + random()*25,  -- longitude 100~125
    NOW() - (random() * INTERVAL '7 days')::interval,
    NOW() - (random() * INTERVAL '30 days')::interval,
    NOW() - (random() * INTERVAL '1 hour')::interval
FROM generate_series(1, 500) AS i
CROSS JOIN LATERAL (
    SELECT
        CASE
            WHEN i <= 200 THEN 'cmcc'
            WHEN i <= 350 THEN 'ctcc'
            ELSE 'cucc'
        END AS carrier,
        CASE
            WHEN i <= 100 THEN 'lte'
            WHEN i <= 200 THEN 'nr'
            WHEN i <= 275 THEN 'lte'
            WHEN i <= 350 THEN 'nr'
            WHEN i <= 425 THEN 'lte'
            ELSE 'nr'
        END AS tech,
        CASE
            WHEN i <= 100 THEN '00E0FC'
            WHEN i <= 200 THEN '001E7E'
            WHEN i <= 275 THEN '00E0FC'
            WHEN i <= 350 THEN '000DB9'
            WHEN i <= 425 THEN '001E7E'
            ELSE '58FB96'
        END AS oui,
        CASE
            WHEN (i-1)%100 < 50 THEN 'FAP-LTE-100' ELSE 'gNB-100'
        END AS product_class,
        CASE
            WHEN i <= 200 THEN 'Huawei'
            WHEN i <= 350 THEN 'ZTE'
            ELSE 'Comba'
        END AS manufacturer,
        CASE
            WHEN (i-1)%100 < 50 THEN 'eLTE-230' ELSE 'AAU5613'
        END AS model_name,
        CASE
            WHEN (i-1)%100 < 50 THEN 'V200R003C10' ELSE 'V100R018C10'
        END AS fw_ver
) c;

-- ============================================================
-- 2. device_info (每台设备一条)
-- ============================================================
INSERT INTO device_info (
    device_id, device_name, address, remark, project_status,
    height, eci, pci, cell_id, bandwidth, transmit_power,
    rf_status, cell_status, kpi_status, num_of_cells,
    first_online_time, last_online_time, run_time,
    creator, updater, created_at, updated_at)
SELECT
    format('a0000000-%s-4000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    format('%s-基站-%s',
        CASE WHEN i <= 200 THEN 'CMCC' WHEN i <= 350 THEN 'CTCC' ELSE 'CUCC' END,
        lpad(i::text, 4, '0')),
    format('中国%s市%s区某某路%s号',
        (ARRAY['北京','上海','广州','深圳','杭州','南京','成都','武汉','西安','长沙'])[1 + floor(random()*10)::int],
        (ARRAY['朝阳','浦东','天河','南山','西湖','鼓楼','武侯','洪山','雁塔','岳麓'])[1 + floor(random()*10)::int],
        (1 + floor(random()*200))::int),
    NULL,
    (ARRAY['deployed','deployed','deployed','commissioning','maintenance'])[1 + floor(random()*5)::int],
    (15 + random()*35)::decimal(10,2),  -- height 15~50m
    lpad((floor(random()*256)::int * 65536 + floor(random()*65536)::int)::text, 8, '0'),  -- ECI
    lpad((floor(random()*504))::text, 3, '0'),  -- PCI 0~503
    lpad((floor(random()*256))::text, 3, '0'),  -- cell_id
    (ARRAY[5, 10, 15, 20, 25, 40, 50, 75, 100])[1 + floor(random()*9)::int]::decimal(8,2),
    (10 + random()*190)::decimal(8,2),  -- transmit_power 10~200
    (ARRAY['on','on','on','off'])[1 + floor(random()*4)::int],
    (ARRAY['active','active','active','blocked'])[1 + floor(random()*4)::int],
    (ARRAY['normal','normal','normal','warning','critical'])[1 + floor(random()*5)::int],
    1,
    NOW() - (random() * INTERVAL '90 days')::interval,
    NOW() - (random() * INTERVAL '1 hour')::interval,
    (random() * 8640000)::bigint,  -- run_time 0~100 days in seconds
    'admin', 'admin',
    NOW() - (random() * INTERVAL '30 days')::interval,
    NOW() - (random() * INTERVAL '1 hour')::interval
FROM generate_series(1, 500) AS i;

-- ============================================================
-- 3. 设备加入默认组
-- ============================================================
INSERT INTO device_group_members (group_id, device_id, added_at)
SELECT
    '00000000-0000-0000-0000-000000000002'::uuid,
    format('a0000000-%s-4000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    NOW() - (random() * INTERVAL '30 days')::interval
FROM generate_series(1, 500) AS i
ON CONFLICT (device_id) DO NOTHING;

-- ============================================================
-- 4. device_parameters (每台设备 20 个参数 = 10,000 行)
-- ============================================================
INSERT INTO device_parameters (device_id, parameter_path, parameter_value, parameter_type, writable, last_updated_at, fap_instance, param_group)
SELECT
    d.device_id,
    p.path,
    p.val,
    p.ptype,
    p.writable,
    NOW() - (random() * INTERVAL '1 day')::interval,
    0,
    p.pgroup
FROM (
    SELECT format('a0000000-%s-4000-8000-000000000001', lpad(i::text, 4, '0'))::uuid AS device_id
    FROM generate_series(1, 500) AS i
) d
CROSS JOIN LATERAL (
    VALUES
    ('Device.DeviceInfo.SoftwareVersion', 'V200R003C10', 'string', false, 'system'),
    ('Device.DeviceInfo.HardwareVersion', 'V2.0', 'string', false, 'system'),
    ('Device.DeviceInfo.UpTime', '1234567', 'unsignedInt', false, 'system'),
    ('Device.ManagementServer.URL', 'http://acs:7547', 'string', true, 'management'),
    ('Device.ManagementServer.PeriodicInformInterval', '300', 'unsignedInt', true, 'management'),
    ('Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus', '1', 'boolean', true, 'rf'),
    ('Device.Services.FAPService.1.FAPControl.LTE.Bandwidth', '20', 'unsignedInt', true, 'rf'),
    ('Device.Services.FAPService.1.FAPControl.LTE.PCI', '100', 'unsignedInt', true, 'rf'),
    ('Device.FAP.GPS.Latitude', '39.9042', 'string', false, 'gps'),
    ('Device.FAP.GPS.Longitude', '116.4074', 'string', false, 'gps'),
    ('Device.FAP.GPS.LockedLatitude', '39.9042', 'string', false, 'gps'),
    ('Device.FAP.GPS.LockedLongitude', '116.4074', 'string', false, 'gps'),
    ('Device.IP.Interface.1.IPv4Address.1.IPAddress', '10.0.1.1', 'string', false, 'network'),
    ('Device.IP.Interface.1.IPv4Address.1.SubnetMask', '255.255.255.0', 'string', false, 'network'),
    ('Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.1.PCI', '50', 'unsignedInt', true, 'neighbor'),
    ('Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.2.PCI', '120', 'unsignedInt', true, 'neighbor'),
    ('Device.Services.FAPService.1.FAPControl.LTE.PlmnList.1.PLMNID', '46000', 'string', true, 'plmn'),
    ('Device.Services.FAPService.1.FAPControl.LTE.TAC', '10001', 'unsignedInt', true, 'rf'),
    ('Device.Services.FAPService.1.FAPControl.LTE.CellIdentity', '12345', 'unsignedInt', true, 'rf'),
    ('Device.Services.FAPService.1.FAPControl.LTE.MaxTxPower', '20', 'int', true, 'rf')
) AS p(path, val, ptype, writable, pgroup);

-- ============================================================
-- 5. 站点 (20个)
-- ============================================================
INSERT INTO sites (id, name, address, longitude, latitude, device_count, status, created_at, updated_at)
SELECT
    format('a0000000-%s-5000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    (ARRAY['北京朝阳','上海浦东','广州天河','深圳南山','杭州西湖','南京鼓楼','成都武侯','武汉洪山','西安雁塔','长沙岳麓',
           '天津和平','重庆渝中','苏州工业园','郑州金水','济南历下','合肥蜀山','福州鼓楼','昆明盘龙','沈阳沈河','大连中山'])[i] || '站点',
    (ARRAY['北京市朝阳区','上海市浦东新区','广州市天河区','深圳市南山区','杭州市西湖区','南京市鼓楼区','成都市武侯区','武汉市洪山区','西安市雁塔区','长沙市岳麓区',
           '天津市和平区','重庆市渝中区','苏州市工业园区','郑州市金水区','济南市历下区','合肥市蜀山区','福州市鼓楼区','昆明市盘龙区','沈阳市沈河区','大连市中山区'])[i],
    100.0 + random()*25,
    22.0 + random()*16,
    10 + floor(random()*40)::int,
    'active',
    NOW() - (random() * INTERVAL '60 days')::interval,
    NOW()
FROM generate_series(1, 20) AS i;

-- ============================================================
-- 6. 拓扑节点 + 边 (30 nodes, 25 edges)
-- ============================================================
INSERT INTO topo_nodes (id, label, node_type, x, y, status, device_sn, created_at, updated_at)
SELECT
    format('a0000000-%s-6000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    CASE i
        WHEN 1 THEN '核心网'
        WHEN 2 THEN 'OMC-Server'
        ELSE '基站-' || (i-2)::text
    END,
    CASE i
        WHEN 1 THEN 'core'
        WHEN 2 THEN 'server'
        ELSE 'enodeb'
    END,
    CASE WHEN i <= 2 THEN 400 ELSE 100 + floor(random()*600)::int END,
    CASE WHEN i <= 2 THEN 50 + (i-1)*60 ELSE 150 + floor(random()*400)::int END,
    (ARRAY['online','online','online','offline'])[1 + floor(random()*4)::int],
    CASE WHEN i > 2 THEN 'TD-' || (ARRAY['cmcc','ctcc','cucc'])[1 + floor(random()*3)::int] || '-' || lpad((i-2)::text, 4, '0') ELSE NULL END,
    NOW(), NOW()
FROM generate_series(1, 30) AS i;

INSERT INTO topo_edges (id, source_id, target_id, label, status, created_at)
SELECT
    format('a0000000-%s-6001-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    CASE
        WHEN i <= 10 THEN format('a0000000-%s-6000-8000-000000000001', lpad('1', 4, '0'))::uuid  -- 核心网→基站
        ELSE format('a0000000-%s-6000-8000-000000000001', lpad('2', 4, '0'))::uuid  -- OMC→基站
    END,
    format('a0000000-%s-6000-8000-000000000001', lpad((3 + i - 1)::text, 4, '0'))::uuid,
    'connected',
    'active',
    NOW()
FROM generate_series(1, 25) AS i;

-- ============================================================
-- 7. 固件版本 (6条: 3运营商 x 2制式)
-- ============================================================
INSERT INTO firmware_versions (id, product_class, version, file_name, file_size, minio_path, release_notes, status, created_at, updated_at)
VALUES
    ('a0000000-0001-7000-8000-000000000001', 'FAP-LTE-100', 'V200R003C10', 'cmcc_lte_v200r003c10.bin', 52428800, 'firmware/cmcc/FAP-LTE-100/V200R003C10/firmware.bin', 'Bug fixes and performance improvements', 'active', NOW(), NOW()),
    ('a0000000-0002-7000-8000-000000000001', 'gNB-100', 'V100R018C10', 'cmcc_nr_v100r018c10.bin', 104857600, 'firmware/cmcc/gNB-100/V100R018C10/firmware.bin', 'NR protocol stack update', 'active', NOW(), NOW()),
    ('a0000000-0003-7000-8000-000000000001', 'FAP-LTE-200', 'V4.16.30P4', 'ctcc_lte_v4.16.30p4.bin', 45056000, 'firmware/ctcc/FAP-LTE-200/V4.16.30P4/firmware.bin', 'Security patch', 'active', NOW(), NOW()),
    ('a0000000-0004-7000-8000-000000000001', 'gNB-200', 'V5.20.10', 'ctcc_nr_v5.20.10.bin', 98304000, 'firmware/ctcc/gNB-200/V5.20.10/firmware.bin', 'Initial NR release', 'active', NOW(), NOW()),
    ('a0000000-0005-7000-8000-000000000001', 'FAP-LTE-300', 'V3.12.50', 'cucc_lte_v3.12.50.bin', 48000000, 'firmware/cucc/FAP-LTE-300/V3.12.50/firmware.bin', 'Stability improvements', 'active', NOW(), NOW()),
    ('a0000000-0006-7000-8000-000000000001', 'gNB-300', 'V2.8.20', 'cucc_nr_v2.8.20.bin', 90000000, 'firmware/cucc/gNB-300/V2.8.20/firmware.bin', 'Feature update', 'active', NOW(), NOW());

-- ============================================================
-- 8. 升级任务 (100条)
-- ============================================================
INSERT INTO upgrade_tasks (id, firmware_id, status, retry_count, started_at, completed_at, created_at, updated_at)
SELECT
    format('a0000000-%s-8000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    (ARRAY[
        'a0000000-0001-7000-8000-000000000001'::uuid,
        'a0000000-0002-7000-8000-000000000001'::uuid,
        'a0000000-0003-7000-8000-000000000001'::uuid,
        'a0000000-0004-7000-8000-000000000001'::uuid,
        'a0000000-0005-7000-8000-000000000001'::uuid,
        'a0000000-0006-7000-8000-000000000001'::uuid
    ])[1 + floor(random()*6)::int],
    (ARRAY['ended','ended','ended','ended','suspended','pending'])[1 + floor(random()*6)::int],
    floor(random()*3)::int,
    CASE WHEN random() > 0.2 THEN NOW() - (random() * INTERVAL '14 days')::interval ELSE NULL END,
    CASE WHEN random() > 0.3 THEN NOW() - (random() * INTERVAL '7 days')::interval ELSE NULL END,
    NOW() - (random() * INTERVAL '14 days')::interval,
    NOW()
FROM generate_series(1, 100) AS i;

-- ============================================================
-- 9. 告警规则 (10条)
-- ============================================================
INSERT INTO alarm_rules (id, name, description, alarm_identifier, severity, condition_type, condition_config, action_type, carrier, enabled, created_at, updated_at)
SELECT
    format('a0000000-%s-a000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    r.name, r.desc, r.code, r.severity,
    'threshold', '{"operator": "gt", "value": 100}'::jsonb,
    'notify',
    r.carrier, true, NOW(), NOW()
FROM generate_series(1, 10) AS i
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['设备离线告警','RF发射异常','CPU使用率过高','内存不足','光功率异常','同步丢失','温度过高','风扇故障','电源异常','传输中断'])[i] AS name,
        (ARRAY['设备连续5分钟无心跳','RF发射功率超出范围','CPU使用率超过阈值','内存使用率超过阈值','光功率低于阈值','GPS同步丢失','设备温度超过阈值','风扇转速异常','电源电压异常','传输链路中断'])[i] AS desc,
        (ARRAY['DEV_OFFLINE','RF_ABNORMAL','CPU_HIGH','MEM_HIGH','OPTICAL_LOW','SYNC_LOST','TEMP_HIGH','FAN_FAULT','POWER_ABNORMAL','LINK_DOWN'])[i] AS code,
        (ARRAY[4,3,3,3,4,4,3,4,4,4])[i] AS severity,
        (ARRAY['cmcc',NULL,'ctcc',NULL,'cucc',NULL,NULL,NULL,NULL,NULL])[i] AS carrier
) r;

-- ============================================================
-- 10. 活动告警 (100条，分布在不同设备)
-- ============================================================
INSERT INTO alarms_active (id, device_id, device_sn, carrier, severity, alarm_type, alarm_identifier, description, status, raised_at, created_at, updated_at)
SELECT
    format('a0000000-%s-b000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    format('a0000000-%s-4000-8000-000000000001', lpad(((i-1)*5 + 1)::text, 4, '0'))::uuid,
    'TD-' || c.carrier || '-' || lpad(((i-1)*5 + 1)::text, 4, '0'),
    c.carrier,
    (ARRAY[1,2,2,3,3,3,4,4,4,4])[1 + floor(random()*10)::int],
    (ARRAY['communication','equipment','processing','environmental'])[1 + floor(random()*4)::int],
    (ARRAY['DEV_OFFLINE','RF_ABNORMAL','CPU_HIGH','MEM_HIGH','TEMP_HIGH','SYNC_LOST','FAN_FAULT','POWER_ABNORMAL','LINK_DOWN','OPTICAL_LOW'])[1 + floor(random()*10)::int],
    (ARRAY['设备离线','RF发射异常','CPU过高','内存不足','温度过高','同步丢失','风扇故障','电源异常','链路中断','光功率低'])[1 + floor(random()*10)::int],
    (ARRAY['active','active','active','acknowledged'])[1 + floor(random()*4)::int],
    NOW() - (random() * INTERVAL '7 days')::interval,
    NOW() - (random() * INTERVAL '7 days')::interval,
    NOW()
FROM generate_series(1, 100) AS i
CROSS JOIN LATERAL (
    SELECT CASE
        WHEN (i-1)*5 + 1 <= 200 THEN 'cmcc'
        WHEN (i-1)*5 + 1 <= 350 THEN 'ctcc'
        ELSE 'cucc'
    END AS carrier
) c;

-- ============================================================
-- 11. 历史告警 (10000条，TimescaleDB 超表)
-- ============================================================
INSERT INTO alarms_history (time, alarm_id, device_id, device_sn, carrier, severity, alarm_type, alarm_identifier, description, status, raised_at, cleared_at)
SELECT
    ts,
    format('a0000000-%s-b100-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    format('a0000000-%s-4000-8000-000000000001', lpad(((i-1) % 500 + 1)::text, 4, '0'))::uuid,
    'TD-' || c.carrier || '-' || lpad(((i-1) % 500 + 1)::text, 4, '0'),
    c.carrier,
    (ARRAY[1,2,2,3,3,3,4,4,4,4])[1 + floor(random()*10)::int],
    (ARRAY['communication','equipment','processing'])[1 + floor(random()*3)::int],
    (ARRAY['DEV_OFFLINE','RF_ABNORMAL','CPU_HIGH','MEM_HIGH','TEMP_HIGH'])[1 + floor(random()*5)::int],
    (ARRAY['设备离线','RF异常','CPU过高','内存不足','温度过高'])[1 + floor(random()*5)::int],
    (ARRAY['cleared','cleared','cleared','acknowledged'])[1 + floor(random()*4)::int],
    ts - (random() * INTERVAL '1 hour')::interval,
    ts + (random() * INTERVAL '4 hours')::interval
FROM generate_series(1, 10000) AS i
CROSS JOIN LATERAL (
    SELECT NOW() - (random() * INTERVAL '30 days')::interval AS ts
) t
CROSS JOIN LATERAL (
    SELECT CASE
        WHEN (i-1) % 500 + 1 <= 200 THEN 'cmcc'
        WHEN (i-1) % 500 + 1 <= 350 THEN 'ctcc'
        ELSE 'cucc'
    END AS carrier
) c;

-- ============================================================
-- 12. PM 文件 (200条)
-- ============================================================
INSERT INTO pm_files (id, device_id, device_sn, carrier, technology, file_name, file_size, collect_time, minio_path, parsed, parsed_at, counter_count, created_at)
SELECT
    format('a0000000-%s-c000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    format('a0000000-%s-4000-8000-000000000001', lpad(((i-1) % 500 + 1)::text, 4, '0'))::uuid,
    'TD-' || c.carrier || '-' || lpad(((i-1) % 500 + 1)::text, 4, '0'),
    c.carrier,
    c.tech,
    'pm_' || c.carrier || '_' || to_char(t.ts, 'YYYYMMDD_HH24MISS') || '.xml',
    (50000 + floor(random()*200000)::int),
    t.ts,
    'pm-files/' || c.carrier || '/' || to_char(t.ts, 'YYYY/MM/DD') || '/TD-' || c.carrier || '-' || lpad(((i-1) % 500 + 1)::text, 4, '0') || '/pm.xml',
    true,
    t.ts + INTERVAL '30 seconds',
    50 + floor(random()*200)::int,
    t.ts
FROM generate_series(1, 200) AS i
CROSS JOIN LATERAL (
    SELECT NOW() - (random() * INTERVAL '14 days')::interval AS ts
) t
CROSS JOIN LATERAL (
    SELECT CASE
        WHEN (i-1) % 500 + 1 <= 200 THEN 'cmcc'
        WHEN (i-1) % 500 + 1 <= 350 THEN 'ctcc'
        ELSE 'cucc'
    END AS carrier,
    CASE WHEN ((i-1) % 500 + 1) % 2 = 0 THEN 'nr' ELSE 'lte' END AS tech
) c;

-- ============================================================
-- 13. PM 计数器 → pm_metrics（metric_type='counter'，20000 条，TimescaleDB hypertable）
-- T-0164-P3 / G3：旧 pm_counters 表已合入 pm_metrics（详见 docs/design/pm-kpi-pipeline-improvements.md §4.3）
-- ============================================================
-- T-0164-P3 fix: 设备唯一标识 (device_oui, device_sn) 双键。
-- OUI 从 3 个常见厂商池循环选（48BF74 / 00A0C6 / 00E0FC），与 devices 表真实数据风格一致。
INSERT INTO pm_metrics (device_oui, device_sn, metric_path, metric_type, metric_value, granularity, time, start_time, end_time, extra)
SELECT
    (ARRAY['48BF74', '00A0C6', '00E0FC'])[1 + ((i-1) % 500) % 3],
    'TD-' || c.carrier || '-' || lpad(((i-1) % 500 + 1)::text, 4, '0'),
    cn.name,
    'counter',
    floor(random() * cn.max_val)::double precision,
    '15min',
    ts,
    ts - INTERVAL '15 minutes',
    ts,
    jsonb_build_object('carrier', c.carrier, 'technology', c.tech)
FROM generate_series(1, 20000) AS i
CROSS JOIN LATERAL (
    SELECT NOW() - (random() * INTERVAL '7 days')::interval AS ts
) t
CROSS JOIN LATERAL (
    SELECT CASE
        WHEN (i-1) % 500 + 1 <= 200 THEN 'cmcc'
        WHEN (i-1) % 500 + 1 <= 350 THEN 'ctcc'
        ELSE 'cucc'
    END AS carrier,
    CASE
        WHEN (i-1) % 500 + 1 <= 100 THEN 'lte'
        WHEN (i-1) % 500 + 1 <= 200 THEN 'nr'
        WHEN (i-1) % 500 + 1 <= 275 THEN 'lte'
        WHEN (i-1) % 500 + 1 <= 350 THEN 'nr'
        WHEN (i-1) % 500 + 1 <= 425 THEN 'lte'
        ELSE 'nr'
    END AS tech
) c
CROSS JOIN LATERAL (
    SELECT * FROM (VALUES
        ('rrc_conn_setup_att', 10000),
        ('rrc_conn_setup_succ', 9800),
        ('erab_setup_att', 5000),
        ('erab_setup_succ', 4900),
        ('dl_prb_used_avg', 50),
        ('dl_prb_available', 100),
        ('ul_prb_used_avg', 30),
        ('erab_abnormal_release', 10),
        ('erab_release_total', 5000),
        ('intra_freq_ho_att', 2000),
        ('intra_freq_ho_succ', 1900)
    ) AS n(name, max_val)
    OFFSET floor(random()*11)::int
    LIMIT 1
) cn
ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn) DO NOTHING;

-- ============================================================
-- 14. KPI 值 → pm_metrics（metric_type='kpi'，20000 条，TimescaleDB hypertable）
-- T-0164-P3 / G3：旧 kpi_values 表已合入 pm_metrics
-- ============================================================
INSERT INTO pm_metrics (device_oui, device_sn, metric_path, metric_type, metric_value, granularity, time, start_time, end_time)
SELECT
    (ARRAY['48BF74', '00A0C6', '00E0FC'])[1 + ((i-1) % 500) % 3],
    'TD-LOAD-' || lpad(((i-1) % 500 + 1)::text, 4, '0'),
    (ARRAY[
        'RRC_CONN_SETUP_SR', 'ERAB_SETUP_SR', 'INTRA_FREQ_HO_SR',
        'CALL_DROP_RATE', 'DL_PRB_UTIL',
        'NR_RRC_SETUP_SR', 'NR_PDCP_RATE_DL', 'NR_SA_HO_SR'
    ])[1 + floor(random()*8)::int],
    'kpi',
    CASE
        WHEN random() < 0.3 THEN 85 + random()*15      -- 85~100 好的性能
        WHEN random() < 0.7 THEN 70 + random()*15       -- 70~85 一般
        ELSE 40 + random()*30                            -- 40~70 差
    END,
    '15min',
    ts,
    ts,
    ts
FROM generate_series(1, 20000) AS i
CROSS JOIN LATERAL (
    SELECT NOW() - (random() * INTERVAL '7 days')::interval AS ts
) t
ON CONFLICT (device_oui, device_sn, metric_path, granularity, end_time, time, object_ldn) DO NOTHING;

-- ============================================================
-- 15. MR 文件 (100条)
-- ============================================================
INSERT INTO mr_files (id, device_id, device_sn, carrier, mr_type, file_name, file_size, collect_time, minio_path, parsed, record_count, created_at)
SELECT
    format('a0000000-%s-d000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    format('a0000000-%s-4000-8000-000000000001', lpad(((i-1) % 500 + 1)::text, 4, '0'))::uuid,
    'TD-' || c.carrier || '-' || lpad(((i-1) % 500 + 1)::text, 4, '0'),
    c.carrier,
    (ARRAY['MRO','MRS','MRE'])[1 + floor(random()*3)::int],
    'mr_' || (ARRAY['MRO','MRS','MRE'])[1 + floor(random()*3)::int] || '_' || to_char(t.ts, 'YYYYMMDD_HH24MISS') || '.xml',
    (100000 + floor(random()*900000)::int),
    t.ts,
    'mr-files/' || c.carrier || '/' || to_char(t.ts, 'YYYY/MM/DD') || '/mr.xml',
    true,
    100 + floor(random()*1000)::int,
    t.ts
FROM generate_series(1, 100) AS i
CROSS JOIN LATERAL (
    SELECT NOW() - (random() * INTERVAL '14 days')::interval AS ts
) t
CROSS JOIN LATERAL (
    SELECT CASE
        WHEN (i-1) % 500 + 1 <= 200 THEN 'cmcc'
        WHEN (i-1) % 500 + 1 <= 350 THEN 'ctcc'
        ELSE 'cucc'
    END AS carrier
) c;

-- ============================================================
-- 16. MR 记录 (5000条，TimescaleDB 超表)
-- ============================================================
INSERT INTO mr_records (time, file_id, device_id, cell_id, mr_type, measurement_data)
SELECT
    ts,
    format('a0000000-%s-d000-8000-000000000001', lpad((1 + floor(random()*100)::int)::text, 4, '0'))::uuid,
    format('a0000000-%s-4000-8000-000000000001', lpad((1 + floor(random()*500)::int)::text, 4, '0'))::uuid,
    lpad(floor(random()*256)::text, 3, '0'),
    (ARRAY['MRO','MRS','MRE'])[1 + floor(random()*3)::int],
    jsonb_build_object(
        'rsrp', -80 - floor(random()*60)::int,
        'rsrq', -10 - floor(random()*10)::int,
        'sinr', -5 + floor(random()*35)::int,
        'ta', floor(random()*200)::int
    )
FROM generate_series(1, 5000) AS i
CROSS JOIN LATERAL (
    SELECT NOW() - (random() * INTERVAL '14 days')::interval AS ts
) t;

-- ============================================================
-- 17. 审计日志 (1000条)
-- ============================================================
INSERT INTO audit_logs (id, user_id, username, action, resource, resource_id, details, ip_address, created_at)
SELECT
    format('a0000000-%s-e000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    '20000000-0000-0000-0000-000000000001'::uuid,
    'admin',
    (ARRAY['login','create','update','delete','query','export','import'])[1 + floor(random()*7)::int],
    (ARRAY['devices','alarms','pm','config','users','roles','firmware'])[1 + floor(random()*7)::int],
    format('a0000000-%s-4000-8000-000000000001', lpad((1 + floor(random()*500)::int)::text, 4, '0')),
    jsonb_build_object('ip', format('192.168.1.%s', floor(random()*254 + 1)::int)),
    format('192.168.%s.%s', floor(random()*10)::int, floor(random()*254 + 1)::int)::inet,
    NOW() - (random() * INTERVAL '30 days')::interval
FROM generate_series(1, 1000) AS i;

-- ============================================================
-- 18. 系统日志 (500条)
-- ============================================================
INSERT INTO system_logs (id, level, source, message, details, created_at)
SELECT
    format('a0000000-%s-f000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    (ARRAY['info','info','info','warn','error'])[1 + floor(random()*5)::int],
    (ARRAY['device-service','alarm-engine','pm-collector','config-sync','acs-session','auth-service'])[1 + floor(random()*6)::int],
    (ARRAY[
        '设备状态更新完成',
        '告警规则匹配成功',
        'PM文件解析完成',
        '配置同步超时',
        '数据库连接池使用率过高',
        'Redis连接超时',
        'NATS消息发布失败',
        '会话清理完成'
    ])[1 + floor(random()*8)::int],
    NULL,
    NOW() - (random() * INTERVAL '7 days')::interval
FROM generate_series(1, 500) AS i;

-- ============================================================
-- 19. 网元消息日志 (2000条)
-- ============================================================
INSERT INTO ne_message_logs (id, device_sn, device_id, message_type, direction, content, created_at)
SELECT
    format('a0000000-%s-1000-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    'TD-' || c.carrier || '-' || lpad((1 + floor(random()*500)::int)::text, 4, '0'),
    format('a0000000-%s-4000-8000-000000000001', lpad((1 + floor(random()*500)::int)::text, 4, '0'))::uuid,
    (ARRAY['Inform','GetParameterValues','SetParameterValues','Reboot','Download','TransferComplete'])[1 + floor(random()*6)::int],
    (ARRAY['inbound','outbound'])[1 + floor(random()*2)::int],
    '<soap:Envelope>...TR-069 SOAP content...</soap:Envelope>',
    NOW() - (random() * INTERVAL '7 days')::interval
FROM generate_series(1, 2000) AS i
CROSS JOIN LATERAL (
    SELECT (ARRAY['cmcc','ctcc','cucc'])[1 + floor(random()*3)::int] AS carrier
) c;

-- ============================================================
-- 20. 设备任务 (200条)
-- ============================================================
INSERT INTO device_tasks (id, device_sn, method, params, status, created_at, completed_at, source, creator_id)
SELECT
    format('a0000000-%s-1100-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    'TD-' || c.carrier || '-' || lpad((1 + floor(random()*500)::int)::text, 4, '0'),
    (ARRAY['GetParameterValues','SetParameterValues','Reboot','Download','GetParameterNames'])[1 + floor(random()*5)::int],
    jsonb_build_object('parameter_path', 'Device.'),
    (ARRAY['completed','completed','completed','completed','pending','failed'])[1 + floor(random()*6)::int],
    NOW() - (random() * INTERVAL '7 days')::interval,
    CASE WHEN random() > 0.3 THEN NOW() - (random() * INTERVAL '3 days')::interval ELSE NULL END,
    'api',
    '20000000-0000-0000-0000-000000000001'
FROM generate_series(1, 200) AS i
CROSS JOIN LATERAL (
    SELECT (ARRAY['cmcc','ctcc','cucc'])[1 + floor(random()*3)::int] AS carrier
) c;

-- ============================================================
-- 21. 配置模板 (10条)
-- ============================================================
INSERT INTO config_templates (id, name, description, carrier, technology, product_class, template_type, parameters, active, version, created_at, updated_at)
SELECT
    format('a0000000-%s-1200-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    t.name, '自动生成的配置模板', t.carrier, t.tech, t.product_class, 'batch_config',
    jsonb_build_object(
        'Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus', '1',
        'Device.Services.FAPService.1.FAPControl.LTE.Bandwidth', t.bw,
        'Device.Services.FAPService.1.FAPControl.LTE.MaxTxPower', t.power
    ),
    true, 1, NOW(), NOW()
FROM (
    VALUES
        (1, 'CMCC-LTE-标准配置', 'cmcc', 'lte', 'FAP-LTE-100', '20', '20'),
        (2, 'CMCC-LTE-高功率配置', 'cmcc', 'lte', 'FAP-LTE-100', '20', '40'),
        (3, 'CMCC-NR-标准配置', 'cmcc', 'nr', 'gNB-100', '100', '50'),
        (4, 'CTCC-LTE-标准配置', 'ctcc', 'lte', 'FAP-LTE-200', '15', '20'),
        (5, 'CTCC-LTE-高功率配置', 'ctcc', 'lte', 'FAP-LTE-200', '15', '40'),
        (6, 'CTCC-NR-标准配置', 'ctcc', 'nr', 'gNB-200', '80', '50'),
        (7, 'CUCC-LTE-标准配置', 'cucc', 'lte', 'FAP-LTE-300', '10', '20'),
        (8, 'CUCC-NR-标准配置', 'cucc', 'nr', 'gNB-300', '60', '50'),
        (9, '通用-LTE-最小配置', 'cmcc', 'lte', NULL, '10', '10'),
        (10, '通用-NR-最小配置', 'cmcc', 'nr', NULL, '40', '10')
) AS t(i, name, carrier, tech, product_class, bw, power);

-- ============================================================
-- 22. 备份任务 (30条)
-- ============================================================
INSERT INTO backup_tasks (id, task_type, target_type, target_ids, status, progress, file_path, started_at, completed_at, created_at, updated_at)
SELECT
    format('a0000000-%s-1300-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    'full', 'device',
    jsonb_build_array(
        format('a0000000-%s-4000-8000-000000000001', lpad((1 + floor(random()*500)::int)::text, 4, '0'))
    ),
    (ARRAY['completed','completed','completed','failed','pending'])[1 + floor(random()*5)::int],
    CASE WHEN random() > 0.2 THEN 100 ELSE floor(random()*100)::int END,
    CASE WHEN random() > 0.2 THEN 'config-backup/' || to_char(NOW(), 'YYYY/MM/DD') || '/backup_' || i || '.xml' ELSE NULL END,
    NOW() - (random() * INTERVAL '14 days')::interval,
    CASE WHEN random() > 0.3 THEN NOW() - (random() * INTERVAL '7 days')::interval ELSE NULL END,
    NOW() - (random() * INTERVAL '14 days')::interval,
    NOW()
FROM generate_series(1, 30) AS i;

-- ============================================================
-- 23. 许可证 (3条) — REMOVED: licenses table dropped (now system_license / device_licenses)
-- ============================================================

-- ============================================================
-- 24. Ops 模板 + 任务 (5 + 20)
-- ============================================================
INSERT INTO ops_templates (id, template_name, description, category, target_device_types, steps, estimated_duration, creator, use_count, tags, created_at, updated_at)
VALUES
    ('a0000000-0001-1500-8000-000000000001', '批量重启', '批量重启选中设备', 'maintenance', '["smallcell"]'::jsonb, '[{"name":"重启设备","method":"Reboot"}]'::jsonb, 60, 'admin', 15, '["重启","维护"]'::jsonb, NOW(), NOW()),
    ('a0000000-0002-1500-8000-000000000001', '参数同步', '批量读取设备参数', 'config', '["smallcell"]'::jsonb, '[{"name":"读取参数","method":"GetParameterValues"}]'::jsonb, 120, 'admin', 42, '["参数","同步"]'::jsonb, NOW(), NOW()),
    ('a0000000-0003-1500-8000-000000000001', '固件升级', '批量升级固件版本', 'upgrade', '["smallcell"]'::jsonb, '[{"name":"下载固件","method":"Download"},{"name":"重启应用","method":"Reboot"}]'::jsonb, 300, 'admin', 8, '["固件","升级"]'::jsonb, NOW(), NOW()),
    ('a0000000-0004-1500-8000-000000000001', '配置备份', '批量备份设备配置', 'backup', '["smallcell"]'::jsonb, '[{"name":"上传配置","method":"Upload"}]'::jsonb, 180, 'admin', 23, '["备份","配置"]'::jsonb, NOW(), NOW()),
    ('a0000000-0005-1500-8000-000000000001', 'RF校准', 'RF参数校准', 'calibration', '["smallcell"]'::jsonb, '[{"name":"设置RF参数","method":"SetParameterValues"}]'::jsonb, 90, 'admin', 5, '["RF","校准"]'::jsonb, NOW(), NOW());

INSERT INTO ops_tasks (id, task_name, template_id, device_sns, status, progress, success_count, fail_count, total_count, creator, started_at, completed_at, created_at, updated_at)
SELECT
    format('a0000000-%s-1600-8000-000000000001', lpad(i::text, 4, '0'))::uuid,
    (ARRAY['批量重启','参数同步','固件升级','配置备份','RF校准'])[1 + floor(random()*5)::int],
    (ARRAY[
        'a0000000-0001-1500-8000-000000000001'::uuid,
        'a0000000-0002-1500-8000-000000000001'::uuid,
        'a0000000-0003-1500-8000-000000000001'::uuid,
        'a0000000-0004-1500-8000-000000000001'::uuid,
        'a0000000-0005-1500-8000-000000000001'::uuid
    ])[1 + floor(random()*5)::int],
    jsonb_build_array(
        'TD-' || (ARRAY['cmcc','ctcc','cucc'])[1 + floor(random()*3)::int] || '-' || lpad((1 + floor(random()*500)::int)::text, 4, '0'),
        'TD-' || (ARRAY['cmcc','ctcc','cucc'])[1 + floor(random()*3)::int] || '-' || lpad((1 + floor(random()*500)::int)::text, 4, '0')
    ),
    (ARRAY['completed','completed','completed','failed','running'])[1 + floor(random()*5)::int],
    CASE WHEN random() > 0.3 THEN 100 ELSE floor(random()*100)::int END,
    CASE WHEN random() > 0.3 THEN 2 ELSE floor(random()*2)::int END,
    CASE WHEN random() > 0.7 THEN floor(random()*2)::int ELSE 0 END,
    2,
    'admin',
    NOW() - (random() * INTERVAL '14 days')::interval,
    CASE WHEN random() > 0.3 THEN NOW() - (random() * INTERVAL '7 days')::interval ELSE NULL END,
    NOW() - (random() * INTERVAL '14 days')::interval,
    NOW()
FROM generate_series(1, 20) AS i;

-- ============================================================
-- 25. 设备规则 (5条) — REMOVED: device_rules table dropped from schema
-- ============================================================

COMMIT;

-- ============================================================
-- 统计输出
-- ============================================================
SELECT '===== 测试数据生成完成 =====' AS info;
SELECT 'devices' AS tbl, count(*) FROM devices WHERE id::text LIKE 'a0000000%'
UNION ALL SELECT 'device_info', count(*) FROM device_info WHERE device_id::text LIKE 'a0000000%'
UNION ALL SELECT 'device_parameters', count(*) FROM device_parameters WHERE device_id::text LIKE 'a0000000%'
UNION ALL SELECT 'device_group_members', count(*) FROM device_group_members WHERE device_id::text LIKE 'a0000000%'
UNION ALL SELECT 'sites', count(*) FROM sites WHERE id::text LIKE 'a0000000%'
UNION ALL SELECT 'topo_nodes', count(*) FROM topo_nodes WHERE id::text LIKE 'a0000000%'
UNION ALL SELECT 'alarms_active', count(*) FROM alarms_active WHERE device_id::text LIKE 'a0000000%'
UNION ALL SELECT 'alarms_history', count(*) FROM alarms_history WHERE device_id::text LIKE 'a0000000%'
UNION ALL SELECT 'pm_metrics_counter', count(*) FROM pm_metrics WHERE device_sn LIKE 'TD-%' AND metric_type = 'counter'
UNION ALL SELECT 'pm_metrics_kpi',     count(*) FROM pm_metrics WHERE device_sn LIKE 'TD-LOAD-%' AND metric_type = 'kpi'
UNION ALL SELECT 'mr_files', count(*) FROM mr_files WHERE device_id::text LIKE 'a0000000%'
UNION ALL SELECT 'mr_records', count(*) FROM mr_records WHERE device_id::text LIKE 'a0000000%'
UNION ALL SELECT 'audit_logs', count(*) FROM audit_logs WHERE id::text LIKE 'a0000000%'
UNION ALL SELECT 'system_logs', count(*) FROM system_logs WHERE id::text LIKE 'a0000000%'
UNION ALL SELECT 'ne_message_logs', count(*) FROM ne_message_logs WHERE device_id::text LIKE 'a0000000%'
UNION ALL SELECT 'device_tasks', count(*) FROM device_tasks WHERE device_sn LIKE 'TD-%'
UNION ALL SELECT 'upgrade_tasks', count(*) FROM upgrade_tasks WHERE id::text LIKE 'a0000000%'
ORDER BY 2 DESC;

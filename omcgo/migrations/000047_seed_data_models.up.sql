-- ============================================================================
-- 000047: 初始化数据模型定义
--    为系统提供开箱即用的 TR069 数据模型
--    支持三级回退：product -> oui -> carrier_default
--
--    采用先删除后插入策略，避免部分唯一索引冲突
-- ============================================================================

-- ============================================================================
-- 1. 清理可能冲突的数据
--    删除所有可能触发唯一约束冲突的记录
-- ============================================================================

-- 删除 carrier_default 级别可能冲突的记录
DELETE FROM data_model_definitions
WHERE scope = 'carrier_default'
AND is_active = true
AND (carrier, technology) IN (
    ('cmcc', 'lte'),
    ('cmcc', 'nr'),
    ('ctcc', 'lte'),
    ('cucc', 'lte')
);

-- 删除 oui 级别可能冲突的记录
DELETE FROM data_model_definitions
WHERE scope = 'oui'
AND is_active = true
AND (carrier, technology, oui) IN (
    ('cmcc', 'lte', '001A2B')
);

-- 删除 product 级别可能冲突的记录
DELETE FROM data_model_definitions
WHERE scope = 'product'
AND is_active = true
AND (carrier, technology, oui, product_class) IN (
    ('cmcc', 'lte', '001A2B', 'SmallCell-LTE')
);

-- 删除相同 ID 的记录（如果存在）
DELETE FROM data_model_definitions WHERE id = '30000047-0001-4000-8000-000000000001';
DELETE FROM data_model_definitions WHERE id = '30000047-0001-4000-8000-000000000002';
DELETE FROM data_model_definitions WHERE id = '30000047-0001-4000-8000-000000000003';
DELETE FROM data_model_definitions WHERE id = '30000047-0001-4000-8000-000000000004';
DELETE FROM data_model_definitions WHERE id = '30000047-0002-4000-8000-000000000001';
DELETE FROM data_model_definitions WHERE id = '30000047-0003-4000-8000-000000000001';

-- ============================================================================
-- 2. 运营商默认级数据模型 (carrier_default scope)
--    最低优先级，作为所有设备的兜底
-- ============================================================================

-- CMCC LTE 默认模型
INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active,
    parameter_tree, description
) VALUES (
    '30000047-0001-4000-8000-000000000001',
    'cmcc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true,
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"},
        "Device.ManagementServer.ConnectionRequestURL": {"access": "r", "type": "string"},
        "Device.Services.FAPService.1": {"access": "rw"},
        "Device.Services.FAPService.1.FAPControl.LTE": {"access": "rw"}
    }'::jsonb,
    '中国移动 LTE 默认数据模型，适用于所有未识别的 LTE 设备'
);

-- CMCC NR 默认模型
INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active,
    parameter_tree, description
) VALUES (
    '30000047-0001-4000-8000-000000000002',
    'cmcc', 'nr', '1.0', NULL, NULL, 'carrier_default', 'active', true,
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"}
    }'::jsonb,
    '中国移动 NR 默认数据模型'
);

-- CTCC LTE 默认模型
INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active,
    parameter_tree, description
) VALUES (
    '30000047-0001-4000-8000-000000000003',
    'ctcc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true,
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"}
    }'::jsonb,
    '中国电信 LTE 默认数据模型'
);

-- CUCC LTE 默认模型
INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active,
    parameter_tree, description
) VALUES (
    '30000047-0001-4000-8000-000000000004',
    'cucc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true,
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"}
    }'::jsonb,
    '中国联通 LTE 默认数据模型'
);

-- ============================================================================
-- 3. OUI 级数据模型 (oui scope)
--    厂商级默认，适用于特定厂商的所有产品
-- ============================================================================

-- BaiCells (OUI: 001A2B) LTE 模型
INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active,
    parameter_tree, description
) VALUES (
    '30000047-0002-4000-8000-000000000001',
    'cmcc', 'lte', '1.0', '001A2B', NULL, 'oui', 'active', true,
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"},
        "Device.DeviceInfo.X_COM_MODULE_TYPE": {"access": "r", "type": "string"},
        "Device.DeviceInfo.X_COM_STATION_RUN_Time": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"},
        "Device.ManagementServer.ConnectionRequestURL": {"access": "r", "type": "string"},
        "Device.Services.FAPService.1": {"access": "rw"},
        "Device.Services.FAPService.1.FAPControl.LTE": {"access": "rw"},
        "Device.Services.FAPService.1.FAPControl.LTE.OpState": {"access": "r", "type": "boolean"},
        "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": {"access": "rw", "type": "boolean"},
        "Device.Services.FAPService.1.FAPControl.LTE.Gateway": {"access": "rw"},
        "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList": {"access": "rw", "type": "string"},
        "Device.FAP.GPS": {"access": "r"},
        "Device.FAP.GPS.LockedLatitude": {"access": "r", "type": "string"},
        "Device.FAP.GPS.LockedLongitude": {"access": "r", "type": "string"},
        "Device.IP.Interface.1.IPv4Address.1.IPAddress": {"access": "r", "type": "string"}
    }'::jsonb,
    'BaiCells 厂商 LTE 设备默认配置'
);

-- ============================================================================
-- 4. Product 级数据模型 (product scope)
--    最具体的配置，针对特定产品型号
-- ============================================================================

-- BaiCells SmallCell-LTE 产品模型
INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active,
    parameter_tree, description
) VALUES (
    '30000047-0003-4000-8000-000000000001',
    'cmcc', 'lte', '1.0', '001A2B', 'SmallCell-LTE', 'product', 'active', true,
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"},
        "Device.DeviceInfo.X_COM_MODULE_TYPE": {"access": "r", "type": "string"},
        "Device.DeviceInfo.X_COM_STATION_RUN_Time": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"},
        "Device.ManagementServer.ConnectionRequestURL": {"access": "r", "type": "string"},
        "Device.Services.FAPService.1": {"access": "rw"},
        "Device.Services.FAPService.1.FAPControl.LTE": {"access": "rw"},
        "Device.Services.FAPService.1.FAPControl.LTE.OpState": {"access": "r", "type": "boolean"},
        "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": {"access": "rw", "type": "boolean"},
        "Device.Services.FAPService.1.FAPControl.LTE.Gateway": {"access": "rw"},
        "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList": {"access": "rw", "type": "string"},
        "Device.FAP.GPS": {"access": "r"},
        "Device.FAP.GPS.LockedLatitude": {"access": "r", "type": "string"},
        "Device.FAP.GPS.LockedLongitude": {"access": "r", "type": "string"},
        "Device.IP.Interface.1.IPv4Address.1.IPAddress": {"access": "r", "type": "string"}
    }'::jsonb,
    'BaiCells SmallCell-LTE 产品数据模型，用于自动开站'
);

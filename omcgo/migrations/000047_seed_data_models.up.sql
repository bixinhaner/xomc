-- ============================================================================
-- 000047: 初始化数据模型定义
--    为系统提供开箱即用的 TR069 数据模型
--    支持三级回退：product -> oui -> carrier_default
-- ============================================================================

-- ============================================================================
-- 1. 运营商默认级数据模型 (carrier_default scope)
--    最低优先级，作为所有设备的兜底
-- ============================================================================

-- CMCC LTE 默认模型
INSERT INTO data_model_definitions (
    id, name, carrier, technology, oui, product_class, scope, status,
    parameter_tree, rpc_methods, description
) VALUES (
    '30000047-0001-4000-8000-000000000001',
    'CMCC LTE Default',
    'cmcc',
    'lte',
    NULL,
    NULL,
    'carrier_default',
    'active',
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"},
        "Device.ManagementServer.ConnectionRequestURL": {"access": "r", "type": "string"},
        "Device.Services.FAPService.1": {"access": "rw"},
        "Device.Services.FAPService.1.FAPControl.LTE": {"access": "rw"}
    }'::jsonb,
    '["GetParameterValues", "SetParameterValues", "GetParameterNames", "Reboot", "Download"]'::jsonb,
    '中国移动 LTE 默认数据模型，适用于所有未识别的 LTE 设备'
) ON CONFLICT (carrier, technology, oui, product_class, scope) WHERE status = 'active' DO NOTHING;

-- CMCC NR 默认模型
INSERT INTO data_model_definitions (
    id, name, carrier, technology, oui, product_class, scope, status,
    parameter_tree, rpc_methods, description
) VALUES (
    '30000047-0001-4000-8000-000000000002',
    'CMCC NR Default',
    'cmcc',
    'nr',
    NULL,
    NULL,
    'carrier_default',
    'active',
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"}
    }'::jsonb,
    '["GetParameterValues", "SetParameterValues", "GetParameterNames", "Reboot", "Download"]'::jsonb,
    '中国移动 NR 默认数据模型'
) ON CONFLICT (carrier, technology, oui, product_class, scope) WHERE status = 'active' DO NOTHING;

-- CTCC LTE 默认模型
INSERT INTO data_model_definitions (
    id, name, carrier, technology, oui, product_class, scope, status,
    parameter_tree, rpc_methods, description
) VALUES (
    '30000047-0001-4000-8000-000000000003',
    'CTCC LTE Default',
    'ctcc',
    'lte',
    NULL,
    NULL,
    'carrier_default',
    'active',
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"}
    }'::jsonb,
    '["GetParameterValues", "SetParameterValues", "GetParameterNames", "Reboot"]'::jsonb,
    '中国电信 LTE 默认数据模型'
) ON CONFLICT (carrier, technology, oui, product_class, scope) WHERE status = 'active' DO NOTHING;

-- CUCC LTE 默认模型
INSERT INTO data_model_definitions (
    id, name, carrier, technology, oui, product_class, scope, status,
    parameter_tree, rpc_methods, description
) VALUES (
    '30000047-0001-4000-8000-000000000004',
    'CUCC LTE Default',
    'cucc',
    'lte',
    NULL,
    NULL,
    'carrier_default',
    'active',
    '{
        "Device.DeviceInfo": {"access": "r"},
        "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"},
        "Device.ManagementServer": {"access": "rw"}
    }'::jsonb,
    '["GetParameterValues", "SetParameterValues", "GetParameterNames", "Reboot"]'::jsonb,
    '中国联通 LTE 默认数据模型'
) ON CONFLICT (carrier, technology, oui, product_class, scope) WHERE status = 'active' DO NOTHING;

-- ============================================================================
-- 2. OUI 级数据模型 (oui scope)
--    厂商级默认，适用于特定厂商的所有产品
-- ============================================================================

-- BaiCells (OUI: 001A2B) LTE 模型
INSERT INTO data_model_definitions (
    id, name, carrier, technology, oui, product_class, scope, status,
    parameter_tree, rpc_methods, description
) VALUES (
    '30000047-0002-4000-8000-000000000001',
    'BaiCells LTE',
    'cmcc',
    'lte',
    '001A2B',
    NULL,
    'oui',
    'active',
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
    '["GetParameterValues", "SetParameterValues", "GetParameterNames", "Reboot", "Download", "Upload"]'::jsonb,
    'BaiCells 厂商 LTE 数据模型'
) ON CONFLICT (carrier, technology, oui, product_class, scope) WHERE status = 'active' DO NOTHING;

-- ============================================================================
-- 3. 产品级数据模型 (product scope)
--    最高优先级，适用于特定产品型号
-- ============================================================================

-- BaiCells SmallCell-LTE 产品模型
INSERT INTO data_model_definitions (
    id, name, carrier, technology, oui, product_class, scope, status,
    parameter_tree, rpc_methods, description
) VALUES (
    '30000047-0003-4000-8000-000000000001',
    'BaiCells SmallCell-LTE',
    'cmcc',
    'lte',
    '001A2B',
    'SmallCell-LTE',
    'product',
    'active',
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
    '["GetParameterValues", "SetParameterValues", "GetParameterNames", "Reboot", "Download", "Upload"]'::jsonb,
    'BaiCells SmallCell-LTE 产品数据模型，用于自动开站'
) ON CONFLICT (carrier, technology, oui, product_class, scope) WHERE status = 'active' DO NOTHING;

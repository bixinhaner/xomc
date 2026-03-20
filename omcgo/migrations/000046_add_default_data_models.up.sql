-- ============================================================================
-- 000046: 添加默认数据模型定义
--    为新设备注册时提供开箱即用的默认数据模型
--    当设备的 OUI/ProductClass 未配置时，回退到这些默认模型
-- ============================================================================

-- ============================================================================
-- 1. CMCC 运营商默认数据模型 (carrier_default scope)
--    最宽泛的默认配置
-- ============================================================================

INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description
) VALUES (
    '30000000-0001-4000-8000-000000000001',
    'cmcc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true,
    '{}'::jsonb,
    '中国移动 LTE 网络默认数据模型配置'
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 2. CMCC 运营商 5G NR 默认数据模型
-- ============================================================================

INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description
) VALUES (
    '30000000-0001-4000-8000-000000000002',
    'cmcc', 'nr', '1.0', NULL, NULL, 'carrier_default', 'active', true,
    '{}'::jsonb,
    '中国移动 5G NR 网络默认数据模型配置'
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 3. 中国电信默认数据模型
-- ============================================================================

INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description
) VALUES (
    '30000000-0001-4000-8000-000000000003',
    'ctcc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true,
    '{}'::jsonb,
    '中国电信 LTE 网络默认数据模型配置'
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 4. 中国联通默认数据模型
-- ============================================================================

INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description
) VALUES (
    '30000000-0001-4000-8000-000000000004',
    'cucc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true,
    '{}'::jsonb,
    '中国联通 LTE 网络默认数据模型配置'
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 5. BaiCell 厂商默认数据模型 (oui scope)
--    OUI = 001A2B 的默认配置
-- ============================================================================

INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description
) VALUES (
    '30000000-0001-4000-8000-000000000010',
    'cmcc', 'lte', '1.0', '001A2B', NULL, 'oui', 'active', true,
    '{}'::jsonb,
    'BaiCell 厂商 LTE 设备默认配置'
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 6. BaiCell SmallCell-LTE 产品模型 (product scope)
--    最具体的配置
-- ============================================================================

INSERT INTO data_model_definitions (
    id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description
) VALUES (
    '30000000-0001-4000-8000-000000000020',
    'cmcc', 'lte', '1.0', '001A2B', 'SmallCell-LTE', 'product', 'active', true,
    '{}'::jsonb,
    'BaiCell SmallCell-LTE 产品特定配置'
)
ON CONFLICT (id) DO NOTHING;

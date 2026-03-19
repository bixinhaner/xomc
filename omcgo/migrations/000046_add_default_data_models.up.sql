-- ============================================================================
-- 000046: 添加默认数据模型定义
--    为新设备注册时提供开箱即用的默认数据模型
--    当设备的 OUI/ProductClass 未配置时，回退到这些默认模型
-- ============================================================================

-- ============================================================================
-- 1. CMCC 运营商默认数据模型 (carrier_default scope)
--    最宽泛的默认配置，-- ============================================================================

INSERT INTO data_model_definitions (
    id, name, description, carrier, technology, oui, product_class, scope, status, parameter_tree
) VALUES (
    '30000000-0001-4000-8000-000000000001',
    'CMCC LTE 默认模型',
    '中国移动 LTE 网络默认数据模型配置',
    'cmcc', 'lte', '', '', 'carrier_default', 'active',
    '{
        "version": "1.0",
    }'::jsonb
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 2. CMCC 运营商 5G NR 默认数据模型
-- ============================================================================

INSERT INTO data_model_definitions (
    id, name, description, carrier, technology, oui, product_class, scope, status, parameter_tree
) VALUES (
    '30000000-0001-4000-8000-000000000002',
    'CMCC NR 默认模型',
    '中国移动 5G NR 网络默认数据模型配置',
    'cmcc', 'nr', '', '', 'carrier_default', 'active',
    '{
        "version": "1.0"
    }'::jsonb
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 3. 中国电信默认数据模型
-- ============================================================================

INSERT INTO data_model_definitions (
    id, name, description, carrier, technology, oui, product_class, scope, status, parameter_tree
) VALUES (
    '30000000-0001-4000-8000-000000000003',
    'CTCC LTE 默认模型',
    '中国电信 LTE 网络默认数据模型配置',
    'ctcc', 'lte', '', '', 'carrier_default', 'active',
    '{
        "version": "1.0"
    }'::jsonb
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 4. 中国联通默认数据模型
-- ============================================================================

INSERT INTO data_model_definitions (
    id, name, description, carrier, technology, oui, product_class, scope, status, parameter_tree
) VALUES (
    '30000000-0001-4000-8000-000000000004',
    'CUCC LTE 默认模型',
    '中国联通 LTE 网络默认数据模型配置',
    'cucc', 'lte', '', '', 'carrier_default', 'active',
    '{
        "version": "1.0"
    }'::jsonb
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 5. BaiCell 厂商默认数据模型 (oui scope)
--    OUI = 001A2B 的默认配置
-- ============================================================================

INSERT INTO data_model_definitions (
    id, name, description, carrier, technology, oui, product_class, scope, status, parameter_tree
) VALUES (
    '30000000-0001-4000-8000-000000000010',
    'BaiCell LTE 默认模型',
    'BaiCell 厂商 LTE 设备默认配置',
    'cmcc', 'lte', '001A2B', '', 'oui', 'active',
    '{
        "version": "1.0"
    }'::jsonb
)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 6. BaiCell SmallCell-LTE 产品模型 (product scope)
--    最具体的配置
-- ============================================================================

INSERT INTO data_model_definitions (
    id, name, description, carrier, technology, oui, product_class, scope, status, parameter_tree
) VALUES (
    '30000000-0001-4000-8000-000000000020',
    'BaiCell SmallCell-LTE 产品模型',
    'BaiCell SmallCell-LTE 产品特定配置',
    'cmcc', 'lte', '001A2B', 'SmallCell-LTE', 'product', 'active',
    '{
        "version": "1.0"
    }'::jsonb
)
ON CONFLICT (id) DO NOTHING;

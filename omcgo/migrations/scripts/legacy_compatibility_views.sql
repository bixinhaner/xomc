-- ============================================================
-- legacy_compatibility_views.sql
-- 兼容老系统的视图
-- 
-- 用途: 让老系统的查询代码无需修改即可运行
-- 使用场景: 数据迁移过渡期,逐步替换为直接查询新表
--
-- 执行方式:
-- psql -U omcgo -d omcgo -f legacy_compatibility_views.sql
-- ============================================================

-- ============================================================
-- 视图 1: v_param_groups_legacy
-- 兼容老 small_cell_param_group 表的查询
-- ============================================================
CREATE OR REPLACE VIEW v_param_groups_legacy AS
SELECT 
    g.id::VARCHAR,                              -- id (转字符串兼容老系统)
    g.group_code AS keyword,
    g.group_name_zh AS param_name,
    g.group_name_en AS param_name_en,
    CASE 
        WHEN g.is_listable AND g.is_modifiable AND g.is_addable AND g.is_removable THEN 'S'
        WHEN g.is_listable THEN 'Y' 
        ELSE 'N' 
    END AS v_lst,
    CASE WHEN g.is_modifiable THEN 'Y' ELSE 'N' END AS v_mod,
    CASE WHEN g.is_addable THEN 'Y' ELSE 'N' END AS v_add,
    CASE WHEN g.is_removable THEN 'Y' ELSE 'N' END AS v_rmv,
    COALESCE(g.parent_id::VARCHAR, g.id::VARCHAR) AS parent_id,  -- 根节点指向自己
    g.param_version,
    g.add_object_path AS add_path,
    CASE WHEN g.mobile_support THEN 'Y' ELSE 'N' END AS mobile_support,
    CASE WHEN g.broadband_support THEN 'Y' ELSE 'N' END AS broadband_support,
    g.platform_support[1] AS platform_support,
    g.cell_number,
    g.cell_index_location,
    CASE WHEN g.require_second_confirm THEN 'Y' ELSE NULL END AS second_confirm,
    g.confirm_message_en AS confirm_en,
    g.confirm_message_zh AS confirm_cn,
    g.display_order AS dis_order
FROM mml_param_groups g
WHERE g.is_active = true 
  AND g.deleted_at IS NULL;

-- ============================================================
-- 视图 2: v_params_legacy
-- 兼容老 small_cell_param 表的查询
-- ============================================================
CREATE OR REPLACE VIEW v_params_legacy AS
SELECT 
    ROW_NUMBER() OVER (ORDER BY p.param_version, p.tr069_path)::BIGINT AS "PARAM_ID",
    p.param_name_zh AS "PARAM_NAME",
    p.tr069_path AS "NAME_PATH",
    p.default_value AS "DFT_VALUE",
    build_v_type_string(p.value_type, p.value_constraint) AS "V_TYPE",
    CASE WHEN p.is_writable THEN 'W' ELSE '-' END AS "V_WRITABLE",
    CASE WHEN p.is_listable THEN 'Y' ELSE 'N' END AS "V_LST",
    CASE WHEN p.is_modifiable THEN 'Y' ELSE 'N' END AS "V_MOD",
    CASE WHEN p.is_addable THEN 'Y' ELSE 'N' END AS "V_ADD",
    CASE WHEN p.is_removable THEN 'Y' ELSE 'N' END AS "V_RMV",
    CASE WHEN p.is_leaf THEN 'Y' ELSE 'N' END AS "IS_LEAF",
    p.display_order AS "DISP_ORD",
    CASE WHEN p.is_active THEN '0' ELSE '1' END AS "STOP_SIGN",
    p.memo AS "MEMO",
    CASE WHEN p.is_dynamic THEN '1' ELSE '0' END AS "V_DYNAMIC",
    p.param_version AS "PARAM_VERSION",
    p.param_code AS "MIB_DN",
    p.param_name_en AS "PARAM_NAME_EN",
    CASE WHEN p.mobile_support THEN 'Y' ELSE 'N' END AS "MOBILE_SUPPORT",
    CASE WHEN p.broadband_support THEN 'Y' ELSE 'N' END AS "BROADBAND_SUPPORT",
    p.js_regex AS "JS_REGEX",
    p.title_zh AS "TITLE_CN",
    p.title_en AS "TITLE_EN",
    p.platform_support[1] AS "PLATFORM_SUPPORT",
    p.explanation_zh AS "CN_EXPLANATION",
    p.explanation_en AS "EN_EXPLANATION",
    p.software_version AS "SOFTWARE_VERSION",
    CASE WHEN p.require_second_confirm THEN 'Y' ELSE NULL END AS second_confirm,
    p.confirm_message_en AS confirm_en,
    p.confirm_message_zh AS confirm_cn
FROM mml_params p
WHERE p.is_active = true 
  AND p.deleted_at IS NULL;

-- ============================================================
-- 视图 3: v_param_tree (新增 - 树形结构查询)
-- 查询完整的参数树(分组 + 参数)
-- ============================================================
CREATE OR REPLACE VIEW v_param_tree AS
WITH RECURSIVE group_tree AS (
    -- 根节点
    SELECT 
        id,
        group_code,
        group_name_zh,
        group_name_en,
        parent_id,
        level,
        param_version,
        display_order,
        group_code::TEXT AS path
    FROM mml_param_groups
    WHERE parent_id IS NULL
      AND is_active = true
      AND deleted_at IS NULL
    
    UNION ALL
    
    -- 子节点
    SELECT 
        g.id,
        g.group_code,
        g.group_name_zh,
        g.group_name_en,
        g.parent_id,
        g.level,
        g.param_version,
        g.display_order,
        gt.path || '.' || g.group_code
    FROM mml_param_groups g
    JOIN group_tree gt ON g.parent_id = gt.id
    WHERE g.is_active = true
      AND g.deleted_at IS NULL
)
SELECT 
    gt.id AS group_id,
    gt.group_code,
    gt.group_name_zh,
    gt.group_name_en,
    gt.parent_id,
    gt.level,
    gt.param_version,
    gt.path AS group_path,
    p.id AS param_id,
    p.param_code,
    p.param_name_zh AS param_name,
    p.param_name_en,
    p.tr069_path,
    p.value_type,
    p.value_constraint,
    p.default_value,
    p.is_writable,
    p.is_leaf,
    p.display_order AS param_order
FROM group_tree gt
LEFT JOIN mml_group_param_rel r ON r.group_id = gt.id
LEFT JOIN mml_params p ON p.id = r.param_id 
    AND p.is_active = true 
    AND p.deleted_at IS NULL
ORDER BY gt.path, gt.display_order, p.display_order;

-- ============================================================
-- 视图 4: v_param_version_stats (新增 - 版本统计)
-- 查看各版本的参数统计信息
-- ============================================================
CREATE OR REPLACE VIEW v_param_version_stats AS
SELECT 
    v.version_code,
    v.version_name,
    v.description,
    v.product_models,
    v.is_active,
    v.is_deprecated,
    COUNT(DISTINCT g.id) AS group_count,
    COUNT(DISTINCT p.id) AS param_count,
    COUNT(DISTINCT r.id) AS relation_count,
    v.group_count AS stored_group_count,
    v.param_count AS stored_param_count
FROM mml_param_versions v
LEFT JOIN mml_param_groups g ON g.param_version = v.version_code 
    AND g.is_active = true 
    AND g.deleted_at IS NULL
LEFT JOIN mml_group_param_rel r ON r.group_id = g.id
LEFT JOIN mml_params p ON p.id = r.param_id 
    AND p.is_active = true 
    AND p.deleted_at IS NULL
GROUP BY v.version_code, v.version_name, v.description, v.product_models, 
         v.is_active, v.is_deprecated, v.group_count, v.param_count;

-- ============================================================
-- 使用示例
-- ============================================================

-- 示例 1: 查询老格式的分组数据
-- SELECT * FROM v_param_groups_legacy WHERE param_version = 'QB1.0' ORDER BY dis_order;

-- 示例 2: 查询老格式的参数数据
-- SELECT * FROM v_params_legacy WHERE "PARAM_VERSION" = 'QB1.0' LIMIT 10;

-- 示例 3: 查询完整的参数树
-- SELECT * FROM v_param_tree WHERE param_version = 'QB1.0' AND level <= 2;

-- 示例 4: 查看版本统计
-- SELECT * FROM v_param_version_stats;

-- 示例 5: 查询某个分组下的所有参数
-- SELECT * FROM v_param_tree WHERE group_code = 'NTP' AND param_id IS NOT NULL;

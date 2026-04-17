-- ============================================================
-- migrate_old_mml_data.sql
-- 老 MML 数据迁移脚本
-- 
-- 使用说明:
-- 1. 确保已执行 000022_mml_param_library.sql 创建新表
-- 2. 在测试环境验证后再在生产环境执行
-- 3. 建议分批执行,每步验证数据完整性
--
-- 执行方式:
-- psql -U omcgo -d omcgo -f migrate_old_mml_data.sql
-- ============================================================

BEGIN;

-- ============================================================
-- 步骤 1: 创建临时映射表(保留老 ID 到新 UUID 的映射)
-- ============================================================
CREATE TEMP TABLE temp_group_id_mapping (
    old_id INT,
    new_id UUID
);

CREATE TEMP TABLE temp_param_id_mapping (
    old_id BIGINT,
    new_id UUID
);

-- ============================================================
-- 步骤 2: 迁移分组数据
-- ============================================================
INSERT INTO mml_param_groups (
    id, group_code, group_name_zh, group_name_en, 
    parent_id, level,
    is_listable, is_modifiable, is_addable, is_removable,
    add_object_path, delete_object_path,
    param_version, platform_support, mobile_support, broadband_support,
    cell_number, cell_index_location,
    require_second_confirm, confirm_message_zh, confirm_message_en,
    display_order, is_active, created_at
)
SELECT 
    gen_random_uuid(),                          -- 新 UUID
    COALESCE(keyword, ''),                      -- group_code
    COALESCE(param_name, ''),                   -- group_name_zh
    param_name_en,                              -- group_name_en
    NULL,                                       -- parent_id (后续更新)
    0,                                          -- level
    CASE WHEN v_lst = 'Y' THEN true ELSE false END,
    CASE WHEN v_mod = 'Y' THEN true ELSE false END,
    CASE WHEN v_add LIKE 'Y%' THEN true ELSE false END,
    CASE WHEN v_rmv LIKE 'Y%' THEN true ELSE false END,
    add_path,
    add_path,
    COALESCE(param_version, 'QB1.0'),
    NULLIF(ARRAY[platform_support], ARRAY[NULL]),
    CASE WHEN COALESCE(mobile_support, 'Y') = 'Y' THEN true ELSE false END,
    CASE WHEN COALESCE(broadband_support, 'Y') = 'Y' THEN true ELSE false END,
    COALESCE(cell_number, 1),
    COALESCE(cell_index_location, 0),
    CASE WHEN second_confirm IS NOT NULL THEN true ELSE false END,
    confirm_cn,
    confirm_en,
    COALESCE(dis_order, 0),
    true,                                       -- 默认激活
    NOW()
FROM small_cell_param_group;

-- 保存映射关系
INSERT INTO temp_group_id_mapping (old_id, new_id)
SELECT id, id FROM mml_param_groups;

-- 统计
DO $$
DECLARE
    count INT;
BEGIN
    SELECT COUNT(*) INTO count FROM mml_param_groups;
    RAISE NOTICE '分组数据迁移完成: % 条', count;
END $$;

-- ============================================================
-- 步骤 3: 更新 parent_id 树形结构
-- ============================================================
UPDATE mml_param_groups g
SET parent_id = m.new_id
FROM small_cell_param_group s
JOIN temp_group_id_mapping m ON s.parent_id = m.old_id
WHERE g.group_code = s.keyword 
  AND g.param_version = s.param_version
  AND s.parent_id != s.id;  -- 排除根节点

-- 更新 level 层级
WITH RECURSIVE tree AS (
    -- 根节点
    SELECT id, 0 AS level
    FROM mml_param_groups
    WHERE parent_id IS NULL
    
    UNION ALL
    
    -- 子节点
    SELECT g.id, t.level + 1
    FROM mml_param_groups g
    JOIN tree t ON g.parent_id = t.id
)
UPDATE mml_param_groups g
SET level = t.level
FROM tree t
WHERE g.id = t.id;

DO $$
DECLARE
    count INT;
BEGIN
    SELECT COUNT(*) INTO count FROM mml_param_groups WHERE parent_id IS NOT NULL;
    RAISE NOTICE '树形结构更新完成: % 条有父节点', count;
END $$;

-- ============================================================
-- 步骤 4: 迁移参数数据
-- ============================================================
INSERT INTO mml_params (
    id, param_code, param_name_zh, param_name_en, tr069_path,
    value_type, value_constraint, default_value, js_regex,
    is_writable, is_listable, is_modifiable, is_addable, is_removable,
    is_leaf, is_dynamic, display_order,
    param_version, software_version, platform_support,
    mobile_support, broadband_support,
    memo, explanation_zh, explanation_en, title_zh, title_en,
    require_second_confirm, confirm_message_zh, confirm_message_en,
    is_active, created_at
)
SELECT
    gen_random_uuid(),
    COALESCE(MIB_DN, ''),
    COALESCE(PARAM_NAME, ''),
    PARAM_NAME_EN,
    NAME_PATH,
    
    -- 使用 parse_v_type 函数解析
    (parse_v_type(V_TYPE)).value_type,
    (parse_v_type(V_TYPE)).value_constraint,
    
    DFT_VALUE,
    JS_REGEX,
    
    CASE WHEN V_WRITABLE = 'W' THEN true ELSE false END,
    CASE WHEN V_LST = 'Y' THEN true ELSE false END,
    CASE WHEN V_MOD = 'Y' THEN true ELSE false END,
    CASE WHEN V_ADD LIKE 'Y%' THEN true ELSE false END,
    CASE WHEN V_RMV LIKE 'Y%' THEN true ELSE false END,
    
    CASE WHEN COALESCE(IS_LEAF, 'Y') = 'Y' THEN true ELSE false END,
    CASE WHEN V_DYNAMIC = '1' THEN true ELSE false END,
    COALESCE(DISP_ORD, 0)::INT,
    
    COALESCE(PARAM_VERSION, 'QB1.0'),
    SOFTWARE_VERSION,
    NULLIF(ARRAY[PLATFORM_SUPPORT], ARRAY[NULL]),
    CASE WHEN COALESCE(MOBILE_SUPPORT, 'Y') = 'Y' THEN true ELSE false END,
    CASE WHEN COALESCE(BROADBAND_SUPPORT, 'Y') = 'Y' THEN true ELSE false END,
    
    MEMO,
    CN_EXPLANATION,
    EN_EXPLANATION,
    TITLE_CN,
    TITLE_EN,
    CASE WHEN second_confirm IS NOT NULL THEN true ELSE false END,
    confirm_cn,
    confirm_en,
    CASE WHEN COALESCE(STOP_SIGN, '0') = '0' THEN true ELSE false END,
    NOW()
FROM small_cell_param;

-- 保存映射关系
INSERT INTO temp_param_id_mapping (old_id, new_id)
SELECT "PARAM_ID", id FROM mml_params;

-- 统计
DO $$
DECLARE
    count INT;
BEGIN
    SELECT COUNT(*) INTO count FROM mml_params;
    RAISE NOTICE '参数数据迁移完成: % 条', count;
END $$;

-- ============================================================
-- 步骤 5: 建立关联关系 - 方式 1: add_path 前缀匹配
-- ============================================================
INSERT INTO mml_group_param_rel (group_id, param_id, sort_order, matched_by, match_rule)
SELECT 
    g.id, p.id, p.display_order, 'path_prefix',
    'tr069_path LIKE ' || quote_literal(g.add_object_path || '%')
FROM mml_param_groups g
JOIN mml_params p ON p.param_version = g.param_version
WHERE g.add_object_path IS NOT NULL
  AND p.tr069_path LIKE (g.add_object_path || '%')
  AND p.is_active = true;

DO $$
DECLARE
    count INT;
BEGIN
    SELECT COUNT(*) INTO count FROM mml_group_param_rel WHERE matched_by = 'path_prefix';
    RAISE NOTICE '关联关系建立(path_prefix): % 条', count;
END $$;

-- ============================================================
-- 步骤 6: 建立关联关系 - 方式 2: keyword 模糊匹配
-- ============================================================
INSERT INTO mml_group_param_rel (group_id, param_id, sort_order, matched_by, match_rule)
SELECT 
    g.id, p.id, p.display_order, 'keyword',
    'tr069_path ILIKE ' || quote_literal('%' || g.group_code || '%')
FROM mml_param_groups g
JOIN mml_params p ON p.param_version = g.param_version
WHERE g.group_code IS NOT NULL
  AND g.group_code != ''
  AND g.add_object_path IS NULL  -- 排除已处理的动态对象
  AND p.tr069_path ILIKE ('%' || g.group_code || '%')
  AND p.is_active = true
  AND NOT EXISTS (
      SELECT 1 FROM mml_group_param_rel r WHERE r.group_id = g.id AND r.param_id = p.id
  );

DO $$
DECLARE
    count INT;
BEGIN
    SELECT COUNT(*) INTO count FROM mml_group_param_rel WHERE matched_by = 'keyword';
    RAISE NOTICE '关联关系建立(keyword): % 条', count;
END $$;

-- ============================================================
-- 步骤 7: 更新版本统计表
-- ============================================================
UPDATE mml_param_versions v
SET 
    group_count = (SELECT COUNT(*) FROM mml_param_groups g WHERE g.param_version = v.version_code),
    param_count = (SELECT COUNT(*) FROM mml_params p WHERE p.param_version = v.version_code);

DO $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN SELECT version_code, group_count, param_count FROM mml_param_versions
    LOOP
        RAISE NOTICE '版本 %: % 个分组, % 个参数', rec.version_code, rec.group_count, rec.param_count;
    END LOOP;
END $$;

-- ============================================================
-- 步骤 8: 数据验证
-- ============================================================
DO $$
DECLARE
    v_old_group_count INT;
    v_new_group_count INT;
    v_old_param_count INT;
    v_new_param_count INT;
    v_rel_count INT;
    v_unmatched_params INT;
BEGIN
    -- 验证分组数量
    SELECT COUNT(*) INTO v_old_group_count FROM small_cell_param_group;
    SELECT COUNT(*) INTO v_new_group_count FROM mml_param_groups;
    
    IF v_old_group_count != v_new_group_count THEN
        RAISE WARNING '分组数量不一致: 老=%, 新=%', v_old_group_count, v_new_group_count;
    ELSE
        RAISE NOTICE '✓ 分组数量一致: %', v_new_group_count;
    END IF;
    
    -- 验证参数数量
    SELECT COUNT(*) INTO v_old_param_count FROM small_cell_param;
    SELECT COUNT(*) INTO v_new_param_count FROM mml_params;
    
    IF v_old_param_count != v_new_param_count THEN
        RAISE WARNING '参数数量不一致: 老=%, 新=%', v_old_param_count, v_new_param_count;
    ELSE
        RAISE NOTICE '✓ 参数数量一致: %', v_new_param_count;
    END IF;
    
    -- 统计关联关系
    SELECT COUNT(*) INTO v_rel_count FROM mml_group_param_rel;
    RAISE NOTICE '✓ 建立关联关系: % 条', v_rel_count;
    
    -- 检查未关联的参数
    SELECT COUNT(*) INTO v_unmatched_params
    FROM mml_params p
    WHERE NOT EXISTS (
        SELECT 1 FROM mml_group_param_rel r WHERE r.param_id = p.id
    )
    AND p.is_active = true;
    
    IF v_unmatched_params > 0 THEN
        RAISE WARNING '未关联的参数: % 条(需要手动处理)', v_unmatched_params;
    ELSE
        RAISE NOTICE '✓ 所有参数都已关联';
    END IF;
END $$;

COMMIT;

-- ============================================================
-- 临时表清理(会话结束时自动删除)
-- ============================================================
-- DROP TABLE temp_group_id_mapping;
-- DROP TABLE temp_param_id_mapping;

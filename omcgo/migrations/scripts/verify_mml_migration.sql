-- ============================================================
-- verify_mml_migration.sql
-- 验证 MML 参数库迁移的完整性
-- 
-- 执行方式:
-- psql -U omcgo -d omcgo -f verify_mml_migration.sql
-- ============================================================

\echo '=========================================='
\echo 'MML 参数库迁移验证'
\echo '=========================================='

-- ============================================================
-- 1. 检查表是否存在
-- ============================================================
\echo ''
\echo '1. 检查表结构...'

SELECT 
    tablename,
    CASE 
        WHEN tablename = 'mml_param_versions' THEN '版本管理表'
        WHEN tablename = 'mml_param_groups' THEN '参数分组表'
        WHEN tablename = 'mml_params' THEN '参数定义表'
        WHEN tablename = 'mml_group_param_rel' THEN '关联表'
    END AS description
FROM pg_tables
WHERE schemaname = 'public'
  AND tablename LIKE 'mml_%'
ORDER BY tablename;

-- ============================================================
-- 2. 检查索引
-- ============================================================
\echo ''
\echo '2. 检查索引...'

SELECT 
    t.tablename,
    indexname,
    indexdef
FROM pg_indexes t
WHERE t.schemaname = 'public'
  AND t.tablename LIKE 'mml_%'
ORDER BY t.tablename, indexname;

-- ============================================================
-- 3. 检查函数
-- ============================================================
\echo ''
\echo '3. 检查辅助函数...'

SELECT 
    routine_name,
    data_type AS return_type
FROM information_schema.routines
WHERE routine_schema = 'public'
  AND routine_name IN ('parse_v_type', 'build_v_type_string');

-- ============================================================
-- 4. 检查版本数据
-- ============================================================
\echo ''
\echo '4. 检查版本数据...'

SELECT 
    version_code,
    version_name,
    product_models,
    is_active
FROM mml_param_versions
ORDER BY version_code;

-- ============================================================
-- 5. 检查约束
-- ============================================================
\echo ''
\echo '5. 检查约束...'

SELECT
    tc.table_name,
    tc.constraint_name,
    tc.constraint_type
FROM information_schema.table_constraints tc
WHERE tc.table_schema = 'public'
  AND tc.table_name LIKE 'mml_%'
  AND tc.constraint_type IN ('PRIMARY KEY', 'FOREIGN KEY', 'UNIQUE', 'CHECK')
ORDER BY tc.table_name, tc.constraint_type;

-- ============================================================
-- 6. 如果已迁移数据,验证数据完整性
-- ============================================================
\echo ''
\echo '6. 数据统计...'

DO $$
DECLARE
    v_version_count INT;
    v_group_count INT;
    v_param_count INT;
    v_rel_count INT;
BEGIN
    SELECT COUNT(*) INTO v_version_count FROM mml_param_versions;
    SELECT COUNT(*) INTO v_group_count FROM mml_param_groups;
    SELECT COUNT(*) INTO v_param_count FROM mml_params;
    SELECT COUNT(*) INTO v_rel_count FROM mml_group_param_rel;
    
    RAISE NOTICE '版本数量: %', v_version_count;
    RAISE NOTICE '分组数量: %', v_group_count;
    RAISE NOTICE '参数数量: %', v_param_count;
    RAISE NOTICE '关联关系数量: %', v_rel_count;
    
    IF v_group_count = 0 THEN
        RAISE NOTICE '提示: 数据为空,需要执行 migrate_old_mml_data.sql 迁移数据';
    END IF;
END $$;

-- ============================================================
-- 7. 检查外键关系
-- ============================================================
\echo ''
\echo '7. 检查外键关系...'

SELECT
    kcu.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    tc.constraint_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu 
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage ccu 
    ON tc.constraint_name = ccu.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = 'public'
  AND tc.table_name LIKE 'mml_%'
ORDER BY kcu.table_name;

-- ============================================================
-- 8. 测试辅助函数
-- ============================================================
\echo ''
\echo '8. 测试 parse_v_type 函数...'

SELECT 
    'enum-{true,false}-{1,0}' AS input,
    parse_v_type('enum-{true,false}-{1,0}') AS result;

SELECT 
    'string-[0:256]' AS input,
    parse_v_type('string-[0:256]') AS result;

SELECT 
    'unsignedInt-[1:65535]' AS input,
    parse_v_type('unsignedInt-[1:65535]') AS result;

\echo ''
\echo '9. 测试 build_v_type_string 函数...'

SELECT 
    'enum' AS value_type,
    '{"type": "enum", "labels": ["true", "false"], "values": [1, 0]}'::jsonb AS constraint,
    build_v_type_string('enum', '{"type": "enum", "labels": ["true", "false"], "values": [1, 0]}'::jsonb) AS result;

SELECT 
    'string' AS value_type,
    '{"type": "string", "min_length": 0, "max_length": 256}'::jsonb AS constraint,
    build_v_type_string('string', '{"type": "string", "min_length": 0, "max_length": 256}'::jsonb) AS result;

-- ============================================================
-- 9. 如果已创建视图,检查视图
-- ============================================================
\echo ''
\echo '10. 检查视图...'

SELECT 
    viewname,
    definition
FROM pg_views
WHERE schemaname = 'public'
  AND viewname LIKE 'v_param%'
ORDER BY viewname;

-- ============================================================
-- 总结
-- ============================================================
\echo ''
\echo '=========================================='
\echo '验证完成!'
\echo '=========================================='
\echo ''
\echo '下一步:'
\echo '1. 如果表结构正常,执行 migrate_old_mml_data.sql 迁移数据'
\echo '2. 如果需要兼容老系统,执行 legacy_compatibility_views.sql 创建视图'
\echo '3. 详细使用说明请查看: scripts/README_MML_MIGRATION.md'
\echo ''

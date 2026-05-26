-- ============================================================
-- audit_subfield_to_parammapping_promotion.sql
--
-- Pre/post 审计脚本：MML is_supported 单一真值源重构 PR-A。
--
-- 用途：迁移 000195 前/后跑此脚本，得到以下统计（PR 描述贴上来）：
--   A. mml_command_sub_fields.is_supported=false 涉及的 unique standard_path 数
--   B. 这些 path 中已在 BLQ param_mappings（迁移将 UPDATE）的数量
--   C. 不在 BLQ param_mappings（迁移将 INSERT 占位行）的数量
--   D. 迁移前基线：BLQ param_mappings is_supported=false 行数
--   E. 抽样 10 条会被 INSERT 的 standard_path（验证合理性）
--
-- 用法：
--   psql ... -f omcgo/scripts/audit_subfield_to_parammapping_promotion.sql
-- ============================================================

\echo '--- A. sub_field 标 false 涉及的 unique standard_path 总数 ---'
SELECT COUNT(DISTINCT sp.standard_path) AS unsupported_unique_paths
  FROM mml_command_sub_fields csf
  JOIN standard_params sp ON sp.id = csf.standard_path_id
 WHERE csf.is_supported = false;

\echo '--- B. 这些 path 中已在 BLQ param_mappings 的数量（迁移 UPDATE 目标） ---'
SELECT COUNT(*) AS will_update_count
  FROM (
      SELECT DISTINCT sp.standard_path
        FROM mml_command_sub_fields csf
        JOIN standard_params sp ON sp.id = csf.standard_path_id
       WHERE csf.is_supported = false
  ) s
 WHERE EXISTS (
     SELECT 1
       FROM param_mappings pm
       JOIN param_models pmd ON pmd.id = pm.param_model_id
      WHERE pmd.name = 'BLQ'
        AND pm.standard_path = s.standard_path
 );

\echo '--- C. 不在 BLQ param_mappings 的数量（迁移 INSERT 占位行目标） ---'
SELECT COUNT(*) AS will_insert_count
  FROM (
      SELECT DISTINCT sp.standard_path
        FROM mml_command_sub_fields csf
        JOIN standard_params sp ON sp.id = csf.standard_path_id
       WHERE csf.is_supported = false
  ) s
 WHERE NOT EXISTS (
     SELECT 1
       FROM param_mappings pm
       JOIN param_models pmd ON pmd.id = pm.param_model_id
      WHERE pmd.name = 'BLQ'
        AND pm.standard_path = s.standard_path
 );

\echo '--- D. 迁移前基线：BLQ param_mappings is_supported=false 行数 ---'
SELECT COUNT(*) AS blq_unsupported_baseline
  FROM param_mappings pm
  JOIN param_models pmd ON pmd.id = pm.param_model_id
 WHERE pmd.name = 'BLQ'
   AND pm.is_supported = false;

\echo '--- E. 抽样 10 条将被 INSERT 占位的 standard_path（合理性验证） ---'
SELECT s.standard_path
  FROM (
      SELECT DISTINCT sp.standard_path
        FROM mml_command_sub_fields csf
        JOIN standard_params sp ON sp.id = csf.standard_path_id
       WHERE csf.is_supported = false
  ) s
 WHERE NOT EXISTS (
     SELECT 1
       FROM param_mappings pm
       JOIN param_models pmd ON pmd.id = pm.param_model_id
      WHERE pmd.name = 'BLQ'
        AND pm.standard_path = s.standard_path
 )
 ORDER BY s.standard_path
 LIMIT 10;

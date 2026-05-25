-- +goose Up
-- ============================================================
-- 000180_mml_mod_subfield_backfill.sql
-- 57 MOD standard 命令补 mml_command_sub_fields 关联 — 让它们在 UI 上可写
--
-- 背景：000177 去重后剩 905 mml_commands 行，其中 57 条 source='standard' /
-- operation_type='MOD' 命令的 target_paths 字段有 path（平均 7 条 / 总计 420 条），
-- 但 mml_command_sub_fields 表里 0 关联 → MML 控制台 MOD 面板渲染"无字段可选"
-- → 用户无法填值 → 这 57 条命令在 UI 上完全不可用。
--
-- 这批命令的命名特征：command_code 形态 `MOD_<full_path_underscored>`，与
-- 传统 `MOD <chapter_code>` 形态不同 — 由 spec parser 新代码路径派生，
-- T-0123/T-0170 修复 sub_field 链路时遗漏覆盖。
--
-- 修复策略（"反查回填"）：
-- - 对每条命令的每条 target_path：
--     1. 取 standard_params 中的 id 做 standard_path_id（FK）
--     2. mml_code = path 末段（'.' 后最后一段）
--     3. label_i18n.en = 末段 humanize（驼峰拆词 + snake → 空格，同 000176）
--     4. label_i18n.zh = COALESCE(standard_params.description, en_name)
--     5. access_type = sp.access 映射：READ_WRITE→RW / READ_ONLY→RO
--     6. sort_order = path 在 target_paths 数组中的位置 (1-based)
--
-- 预检确认：
-- - 420 path / 57 命令，distinct=420（无 path 重复）
-- - standard_params 覆盖率 100%（0 缺失）
-- - 末段 leaf 最长 34 字符（VARCHAR(100) 充裕），无命令内 mml_code 冲突
-- ============================================================

-- Step 1: 批量 INSERT
INSERT INTO mml_command_sub_fields (command_id, mml_code, standard_path_id, label_i18n, access_type, sort_order, default_selected, is_required)
SELECT
    c.id AS command_id,
    leaf AS mml_code,
    sp.id AS standard_path_id,
    jsonb_build_object(
        'en', en_name,
        'zh', COALESCE(NULLIF(sp.description, ''), en_name)
    ) AS label_i18n,
    CASE sp.access
        WHEN 'READ_WRITE' THEN 'RW'
        WHEN 'READ_ONLY'  THEN 'RO'
        ELSE NULL
    END AS access_type,
    elem.idx::int AS sort_order,
    true  AS default_selected,
    false AS is_required
FROM mml_commands c,
     LATERAL jsonb_array_elements_text(c.target_paths) WITH ORDINALITY elem(path, idx),
     LATERAL (
         SELECT REGEXP_REPLACE(elem.path, '^.*\.', '') AS leaf
     ) leaf_ext,
     LATERAL (
         SELECT TRIM(REGEXP_REPLACE(
                    REGEXP_REPLACE(
                        REGEXP_REPLACE(leaf_ext.leaf,
                            '([a-z])([A-Z])', '\1 \2', 'g'),       -- camelCase split
                        '([A-Z])([A-Z][a-z])', '\1 \2', 'g'),       -- upper-cluster boundary
                    '_+', ' ', 'g')                                  -- snake → space
                ) AS en_name
     ) hum,
     standard_params sp
WHERE c.operation_type = 'MOD'
  AND c.source = 'standard'
  AND sp.standard_path = elem.path
  AND NOT EXISTS (
      SELECT 1 FROM mml_command_sub_fields sf2
      WHERE sf2.command_id = c.id
  );

-- Step 2: 自检
-- +goose StatementBegin
DO $$
DECLARE
    cmds_zero_sf  INT;
    cmds_covered  INT;
    new_sf_total  INT;
BEGIN
    -- 期望：0 sub_field 的 MOD standard 命令应清零
    SELECT COUNT(*) INTO cmds_zero_sf
      FROM mml_commands c
     WHERE c.operation_type = 'MOD'
       AND c.source = 'standard'
       AND NOT EXISTS (SELECT 1 FROM mml_command_sub_fields WHERE command_id = c.id);

    -- 期望：57 条命令各有 sub_field
    SELECT COUNT(DISTINCT c.id) INTO cmds_covered
      FROM mml_commands c
     WHERE c.operation_type = 'MOD'
       AND c.source = 'standard'
       AND EXISTS (SELECT 1 FROM mml_command_sub_fields WHERE command_id = c.id);

    SELECT COUNT(*) INTO new_sf_total FROM mml_command_sub_fields;

    RAISE NOTICE '000180 post-check: MOD/standard zero-sf=%, covered=%, total sub_fields=%',
                 cmds_zero_sf, cmds_covered, new_sf_total;

    IF cmds_zero_sf > 0 THEN
        RAISE EXCEPTION '000180: % MOD/standard cmds still have 0 sub_field (backfill incomplete)', cmds_zero_sf;
    END IF;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 回滚：删除本次 INSERT 的 sub_field 行。识别条件：
--   - command 属于 57 条 MOD/standard 命令
--   - sub_field 是本 migration 唯一新增（即该 command 的所有 sub_field 都在
--     本 migration 之后 created_at；其他 command 不在 57 集合内）
-- 简化：直接 DELETE 这 57 条命令的所有 sub_field（前提：这些 command 在
-- 000180 之前 0 sub_field，本 Down 仅清理本 migration 新建的）
-- ============================================================
DELETE FROM mml_command_sub_fields sf
 WHERE sf.command_id IN (
     SELECT c.id FROM mml_commands c
      WHERE c.operation_type = 'MOD'
        AND c.source = 'standard'
        AND c.command_code LIKE 'MOD_Device_%'  -- 仅本批新派生命令（区别于老 `MOD <chapter>` 形态）
 );

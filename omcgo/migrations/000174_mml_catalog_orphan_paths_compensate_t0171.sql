-- +goose Up
-- ============================================================
-- 000174_mml_catalog_orphan_paths_compensate_t0171.sql
-- T-0171 Phase 2 —— 孤儿 path 补偿（不放过任何 path，按规则派生 catalog）
--
-- 背景：
--   000173 审计发现 1544 条孤儿 path（standard_params 中无 sub_field 引用）。
--   本迁移按 cmcc-tdlte-v2.3.md §R-2.4 / §R-3 同款规则**全量补齐**：
--     - 不放过任何 path（按 path 规则归集）
--     - 按 path 中"."的层级聚合分组与命令
--     - LST/MOD 都要派生（依 path access 决定 MOD 是否生）
--
-- 派生规则:
--   1. chapter 切分: 按 path 顶层 1-2 段，例
--      Device.DeviceInfo.AntennaInfo.Azimuth → 顶层 'Device.DeviceInfo'
--      → chapter:SX_DEVICE_DEVICEINFO_EXT
--   2. command 切分: 按 object_prefix（path 去末段加 .*），例
--      Device.DeviceInfo.AntennaInfo.Azimuth → object_prefix
--      'Device.DeviceInfo.AntennaInfo.*' → 一个命令
--   3. op 派生（与 spec §R-3 一致）:
--      - LST 恒生（path 集非空）
--      - MOD 当 object_prefix 下 ≥1 条 RW path
--      - ADD/RMV 暂不派生（保守，待业务方拍板后单独补）
--   4. sub_field 关联:
--      - LST 命令 ← 关联该 object_prefix 下所有 path
--      - MOD 命令 ← 仅关联 RW path
--
-- 隔离设计:
--   - mml_command_groups.source = 'extension'
--   - mml_commands.source       = 'extension'
--   - 与 spec 'standard' 和用户 'admin' 三方互不污染
--   - admin UI 后续可清理（catalog_protected=false）
--
-- 重跑安全:
--   - 全 INSERT ... ON CONFLICT DO NOTHING
--   - 不动 standard_params / param_mappings 任何行
--   - Down 段精确 DELETE source='extension' 行（不动 admin/standard）
--
-- 配套独立脚本（用户要求）:
--   - omcgo/scripts/mml_catalog_extension_reset.sql
--     用于不走 goose 的快速删除/重建（删 extension 行后重跑 000174）
-- ============================================================

-- 前置依赖：000173 必须先建审计表
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables
                    WHERE table_name = 'mml_catalog_orphan_paths_audit_t0171') THEN
        RAISE EXCEPTION 'T-0171 Phase 2 requires 000173 audit table; please apply 000173 first';
    END IF;
END $$;
-- +goose StatementEnd

-- ============================================================
-- Step 1: 派生并 INSERT mml_command_groups（新增 chapter 分组）
--
-- 顶层 1-2 段聚合（DISTINCT seg1+seg2）：
--   Device.DeviceInfo.*     → chapter:SX_DEVICE_DEVICEINFO_EXT
--   Device.Services.*       → chapter:SX_DEVICE_SERVICES_EXT
--   Device.FAP.*            → chapter:SX_DEVICE_FAP_EXT
--   DeviceGSM.Bts.*         → chapter:SX_DEVICEGSM_BTS_EXT
--   ... 等约 19 个新 chapter
-- ============================================================
INSERT INTO mml_command_groups (
    id, group_code, group_name_zh, group_name_en,
    path, param_version, display_order,
    is_active, name_i18n, source, catalog_protected,
    family_code, family_name_zh,
    created_at, updated_at
)
SELECT
    gen_random_uuid(),
    'chapter:SX_' || UPPER(seg1) || '_' || UPPER(seg2) || '_EXT',
    seg1 || '.' || seg2 || ' 扩展',
    seg1 || '.' || seg2 || ' Extension',
    ('SX_' || UPPER(seg1) || '_' || UPPER(seg2) || '_EXT')::ltree,
    'cmcc-td-lte-v2.3',
    1000,
    true,
    jsonb_build_object('zh-CN', seg1 || '.' || seg2 || ' 扩展',
                       'en-US', seg1 || '.' || seg2 || ' Extension'),
    'extension',
    false,
    '',  -- family_code 留空（扩展分组无家族归属）
    '',  -- family_name_zh 留空
    NOW(), NOW()
  FROM (
      SELECT DISTINCT
             split_part(standard_path, '.', 1) AS seg1,
             split_part(standard_path, '.', 2) AS seg2
        FROM mml_catalog_orphan_paths_audit_t0171
       WHERE split_part(standard_path, '.', 2) <> ''
  ) tops
ON CONFLICT (param_version, group_code) DO NOTHING;


-- ============================================================
-- Step 2: 派生并 INSERT mml_commands（每个 object_prefix × {LST, MOD?}）
--
-- object_prefix → logical_code 派生（简化版，logical_code 是内部 ID 不强求 camel 分词）:
--   Device.DeviceInfo.AntennaInfo.*     → DEVICEINFO_ANTENNAINFO
--   Device.FAP.GPS.*                    → FAP_GPS
--   DeviceGSM.Bts.{i}.*                 → DEVICEGSM_BTS_I
--   Device.Services.FAPService.{i}.NR.* → SERVICES_FAPSERVICE_I_NR (跳过 Device. 前缀)
--
-- command_code = OP + " EXT_" + logical_code
-- ============================================================

-- 2.1 派生表（按 object_prefix 聚合：是否含 RW、是否含 {i}）
-- 用 CTE 一次性派生供后续两个 INSERT 复用
-- +goose StatementBegin
DO $$
DECLARE
    rec RECORD;
    lst_count INT := 0;
    mod_count INT := 0;
    sub_count INT := 0;
BEGIN
    -- 遍历每个 object_prefix，派生命令
    FOR rec IN
        SELECT
            o.object_prefix,
            -- chapter 反查 group_code
            'chapter:SX_' ||
                UPPER(split_part(MIN(o.standard_path), '.', 1)) || '_' ||
                UPPER(split_part(MIN(o.standard_path), '.', 2)) || '_EXT' AS chapter_code,
            -- logical_code 派生:
            --   1) 去顶层 'Device.' 或 'DeviceGSM.' 前缀（但 Device.X 仍保留 X）— 用 UPPER 后段
            --   2) 去 .*
            --   3) {i} 替换为 _I
            --   4) . 替换为 _
            --   5) 全大写
            -- logical_code VARCHAR(100)：派生后 LEFT(_, 90) 截断 + 留 EXT_ 4 字符空间
            LEFT(UPPER(REGEXP_REPLACE(REPLACE(REPLACE(
                  REGEXP_REPLACE(o.object_prefix, '^Device\.', ''),
                  '.*', ''),
                  '.{i}', '_I'
              ), '\.', '_', 'g')), 90) AS logical_base,
            -- 是否含 RW（决定 MOD 是否生）
            bool_or(o.access = 'READ_WRITE' OR o.access = 'WRITE_ONLY') AS has_rw,
            -- 命令显示名（用 object_prefix 截短后 + 后缀）
            REPLACE(REPLACE(o.object_prefix, '.', ' '), '*', '') AS display_name,
            -- 收集该 object_prefix 下所有 standardPath 进 target_paths
            jsonb_agg(o.standard_path ORDER BY o.standard_path) AS all_paths,
            jsonb_agg(o.standard_path ORDER BY o.standard_path) FILTER (
                WHERE o.access = 'READ_WRITE' OR o.access = 'WRITE_ONLY'
            ) AS rw_paths
          FROM mml_catalog_orphan_paths_audit_t0171 o
         GROUP BY o.object_prefix
    LOOP
        -- 2.2 LST 恒生
        INSERT INTO mml_commands (
            id, command_name, command_code, category, description,
            rpc_method, operation_type, target_paths, target_object,
            group_id, command_name_i18n, logical_code, logical_name_i18n,
            source, catalog_protected, help_doc, created_at
        )
        SELECT
            gen_random_uuid(),
            '列出 ' || rec.display_name AS command_name,
            'LST EXT_' || rec.logical_base AS command_code,
            '99' AS category,  -- 扩展类别
            '列出 ' || rec.display_name AS description,
            'GetParameterValues',
            'LST',
            rec.all_paths,
            REGEXP_REPLACE(rec.object_prefix, '\.\*$', '.') AS target_object,
            g.id,
            jsonb_build_object('en', 'List ' || rec.display_name, 'zh', '列出 ' || rec.display_name),
            'EXT_' || rec.logical_base,
            jsonb_build_object('en', rec.display_name, 'zh', rec.display_name),
            'extension', false, '', NOW()
          FROM mml_command_groups g
         WHERE g.group_code = rec.chapter_code
           AND g.param_version = 'cmcc-td-lte-v2.3'
        ON CONFLICT (command_code) DO NOTHING;

        GET DIAGNOSTICS lst_count = ROW_COUNT;

        -- 2.3 MOD 仅当含 RW path 时派生
        IF rec.has_rw THEN
            INSERT INTO mml_commands (
                id, command_name, command_code, category, description,
                rpc_method, operation_type, target_paths, target_object,
                group_id, command_name_i18n, logical_code, logical_name_i18n,
                source, catalog_protected, help_doc, created_at
            )
            SELECT
                gen_random_uuid(),
                '修改 ' || rec.display_name,
                'MOD EXT_' || rec.logical_base,
                '99',
                '修改 ' || rec.display_name,
                'SetParameterValues',
                'MOD',
                rec.rw_paths,
                REGEXP_REPLACE(rec.object_prefix, '\.\*$', '.'),
                g.id,
                jsonb_build_object('en', 'Modify ' || rec.display_name, 'zh', '修改 ' || rec.display_name),
                'EXT_' || rec.logical_base,
                jsonb_build_object('en', rec.display_name, 'zh', rec.display_name),
                'extension', false, '', NOW()
              FROM mml_command_groups g
             WHERE g.group_code = rec.chapter_code
               AND g.param_version = 'cmcc-td-lte-v2.3'
            ON CONFLICT (command_code) DO NOTHING;

            GET DIAGNOSTICS mod_count = ROW_COUNT;
        END IF;
    END LOOP;

    RAISE NOTICE 'T-0171 Phase 2: 命令派生完成（按 object_prefix 聚合）';
END $$;
-- +goose StatementEnd


-- ============================================================
-- Step 3: 派生并 INSERT mml_command_sub_fields（每个 path 关联到对应 LST/MOD 命令）
-- ============================================================

-- 3.1 LST 命令关联（全部 path）
INSERT INTO mml_command_sub_fields (
    id, command_id, standard_path_id, mml_code, label_i18n,
    default_selected, is_required, sort_order, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    c.id,
    o.standard_param_id,
    -- mml_code 派生: path 最后一段（regexp_replace 比 SUBSTRING+regex 更稳，path 即使以 . 结尾也兜底为空字符串）
    -- 不可为 NULL（NOT NULL 约束），所以加 COALESCE
    COALESCE(NULLIF(REGEXP_REPLACE(REGEXP_REPLACE(o.standard_path, '^.*\.', ''), '[^A-Za-z0-9_]', '_', 'g'), ''),
             'PATH_' || SUBSTRING(o.standard_param_id::text FROM 1 FOR 8)) AS mml_code,
    jsonb_build_object('zh-CN', o.standard_path, 'en-US', o.standard_path),
    true,  -- LST 默认勾选
    false,
    ROW_NUMBER() OVER (PARTITION BY o.object_prefix ORDER BY o.standard_path) AS sort_order,
    NOW(), NOW()
  FROM mml_catalog_orphan_paths_audit_t0171 o
  JOIN mml_commands c
    ON c.source = 'extension'
   AND c.operation_type = 'LST'
   -- T-0171 修复：与命令派生时的 LEFT(_, 90) 一致，否则超长 5G NR path 的 logical_code
   -- 截断后与命令表存值不一致 → JOIN 失败 → sub_field 漏关联
   AND c.logical_code = 'EXT_' ||
       LEFT(UPPER(REGEXP_REPLACE(REPLACE(REPLACE(
             REGEXP_REPLACE(o.object_prefix, '^Device\.', ''),
             '.*', ''),
             '.{i}', '_I'
         ), '\.', '_', 'g')), 90)
ON CONFLICT (command_id, standard_path_id) DO NOTHING;


-- 3.2 MOD 命令关联（仅 RW path）
INSERT INTO mml_command_sub_fields (
    id, command_id, standard_path_id, mml_code, label_i18n,
    default_selected, is_required, sort_order, created_at, updated_at
)
SELECT
    gen_random_uuid(),
    c.id,
    o.standard_param_id,
    COALESCE(NULLIF(REGEXP_REPLACE(REGEXP_REPLACE(o.standard_path, '^.*\.', ''), '[^A-Za-z0-9_]', '_', 'g'), ''),
             'PATH_' || SUBSTRING(o.standard_param_id::text FROM 1 FOR 8)),
    jsonb_build_object('zh-CN', o.standard_path, 'en-US', o.standard_path),
    false,  -- MOD 默认不勾选（用户主动选要改的字段）
    true,   -- 改时必填
    ROW_NUMBER() OVER (PARTITION BY o.object_prefix ORDER BY o.standard_path),
    NOW(), NOW()
  FROM mml_catalog_orphan_paths_audit_t0171 o
  JOIN mml_commands c
    ON c.source = 'extension'
   AND c.operation_type = 'MOD'
   -- T-0171 修复：与命令派生时的 LEFT(_, 90) 一致，否则超长 5G NR path 的 logical_code
   -- 截断后与命令表存值不一致 → JOIN 失败 → sub_field 漏关联
   AND c.logical_code = 'EXT_' ||
       LEFT(UPPER(REGEXP_REPLACE(REPLACE(REPLACE(
             REGEXP_REPLACE(o.object_prefix, '^Device\.', ''),
             '.*', ''),
             '.{i}', '_I'
         ), '\.', '_', 'g')), 90)
 WHERE o.access IN ('READ_WRITE', 'WRITE_ONLY')
ON CONFLICT (command_id, standard_path_id) DO NOTHING;


-- ============================================================
-- Step 4: 统计输出
-- ============================================================
-- +goose StatementBegin
DO $$
DECLARE
    chapter_count INT;
    lst_total     INT;
    mod_total     INT;
    sub_total     INT;
BEGIN
    SELECT COUNT(*) INTO chapter_count
      FROM mml_command_groups
     WHERE source = 'extension' AND param_version = 'cmcc-td-lte-v2.3';

    SELECT COUNT(*) FILTER (WHERE operation_type = 'LST'),
           COUNT(*) FILTER (WHERE operation_type = 'MOD')
      INTO lst_total, mod_total
      FROM mml_commands
     WHERE source = 'extension';

    SELECT COUNT(*) INTO sub_total
      FROM mml_command_sub_fields csf
      JOIN mml_commands c ON c.id = csf.command_id
     WHERE c.source = 'extension';

    RAISE NOTICE 'T-0171 Phase 2 compensation complete:';
    RAISE NOTICE '  Extension chapters created: %', chapter_count;
    RAISE NOTICE '  LST commands generated:     %', lst_total;
    RAISE NOTICE '  MOD commands generated:     %', mod_total;
    RAISE NOTICE '  sub_field links generated:  %', sub_total;
    RAISE NOTICE '  All marked source=extension, catalog_protected=false (allows admin cleanup)';
END $$;
-- +goose StatementEnd


-- ============================================================
-- Step 5: 自检规则（防退化）— 用户硬约束：父子命令不得有 path 重叠
--
-- 规则 1: 同 op 类型内任意两个命令的 target_paths 不应有交集
--         （例: LST EXT_DEVICEINFO ∩ LST EXT_DEVICEINFO_EU.target_paths = ∅）
-- 规则 2: 每条孤儿 path 至少出现于一个 LST 命令的 target_paths（无遗漏）
-- 规则 3: 每条孤儿 path 至少有一个 sub_field 关联（前端可展示）
-- ============================================================
-- +goose StatementBegin
DO $$
DECLARE
    overlap_count   INT;
    not_in_target   INT;
    not_in_subfield INT;
BEGIN
    -- 规则 1: LST 命令两两间 target_paths 不应重叠
    WITH all_paths AS (
        SELECT c.command_code, p.value AS path
          FROM mml_commands c, jsonb_array_elements_text(c.target_paths) p
         WHERE c.source = 'extension' AND c.operation_type = 'LST'
    )
    SELECT COUNT(*) INTO overlap_count
      FROM (
          SELECT a.command_code AS cmd_a, b.command_code AS cmd_b
            FROM all_paths a JOIN all_paths b
              ON a.path = b.path AND a.command_code < b.command_code
           GROUP BY a.command_code, b.command_code
      ) ovr;

    -- 规则 2: 每条孤儿 path 至少出现于一个 LST 命令的 target_paths
    SELECT COUNT(*) INTO not_in_target
      FROM mml_catalog_orphan_paths_audit_t0171 o
     WHERE NOT EXISTS (
         SELECT 1 FROM mml_commands c, jsonb_array_elements_text(c.target_paths) p
          WHERE c.source = 'extension'
            AND c.operation_type = 'LST'
            AND p.value = o.standard_path
     );

    -- 规则 3: 每条孤儿 path 至少有一个 sub_field 关联
    SELECT COUNT(*) INTO not_in_subfield
      FROM mml_catalog_orphan_paths_audit_t0171 o
     WHERE NOT EXISTS (
         SELECT 1 FROM mml_command_sub_fields csf
           JOIN mml_commands c ON c.id = csf.command_id
          WHERE c.source = 'extension'
            AND c.operation_type = 'LST'
            AND csf.standard_path_id = o.standard_param_id
     );

    RAISE NOTICE 'T-0171 Phase 2 self-check:';
    RAISE NOTICE '  Rule 1 (no path overlap between LST cmds): % offending pairs (must be 0)', overlap_count;
    RAISE NOTICE '  Rule 2 (all orphan paths in some target_paths): % missing (must be 0)', not_in_target;
    RAISE NOTICE '  Rule 3 (all orphan paths in some sub_field): % missing (must be 0)', not_in_subfield;

    IF overlap_count > 0 THEN
        RAISE EXCEPTION 'T-0171 self-check FAILED: % LST command pairs share path (规则违反)', overlap_count;
    END IF;
    IF not_in_target > 0 THEN
        RAISE EXCEPTION 'T-0171 self-check FAILED: % orphan paths not covered by any LST command', not_in_target;
    END IF;
    IF not_in_subfield > 0 THEN
        RAISE EXCEPTION 'T-0171 self-check FAILED: % orphan paths have no sub_field link', not_in_subfield;
    END IF;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 安全回滚：精确 DELETE source='extension' 行，不动 standard / admin
-- ============================================================
-- 1. 先删 sub_field（依赖 command_id）
DELETE FROM mml_command_sub_fields
 WHERE command_id IN (
   SELECT id FROM mml_commands WHERE source = 'extension'
 );

-- 2. 删命令
DELETE FROM mml_commands WHERE source = 'extension';

-- 3. 删分组（仅 source='extension' 的 chapter:SX_*_EXT）
DELETE FROM mml_command_groups
 WHERE source = 'extension'
   AND param_version = 'cmcc-td-lte-v2.3'
   AND group_code LIKE 'chapter:SX_%_EXT';

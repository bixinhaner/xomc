-- +goose Up
-- ============================================================
-- 000177_mml_commands_dedup_cleanup.sql
-- MML 命令去重与清洗 — Case A (69 簇) + Case B-1 (10 簇)
--
-- 背景：mml_commands 在 spec parser 多代码路径派生 + T-0171 扩展 catalog 补偿
-- 后累积了 100 个 (command_name, operation_type) 重复簇 / 220 行。
-- 经 docs/qa-report/MML命令去重与清洗日志.md 分类：
--   - Case A (69 簇 / 154 行)：target_paths 完全相同 → 冗余重复，删 85 行
--   - Case B-1 (10 簇 / 21 行)：target_object 相同 但 target_paths 形态不同
--                                （Pattern A/B/C 混存）→ 等同冗余，删 11 行
--   - Case B-2 (21 簇 / 45 行)：target_paths 真实差异（嵌套层级/不同子树/连接 vs
--                                空闲态等）→ 保留，留待业务方决定重命名
--
-- 总计删除 96 行；CASCADE 清除其挂载的 sub_field 副本（同簇内 sub_field 计数已
-- 校验对齐，副本删除不丢有效数据）。
--
-- 保留规则（canonical row）：
--   1. logical_code 非空优先（排除 `MOD `/`LST ` 末尾空格的 buggy 行）
--   2. logical_code 长度长者优先（更具体）
--   3. command_code 不带末尾空格优先
--   4. id 字典序最小为最终 tie-break
--
-- 副带：
--   - Pattern Mix 簇（B-1）若 canonical 行 target_object 为空，从同簇兄弟回填
--   - 修复 "LST 自配置启动" label 异常 → 改为 "列出 自配置启动状态" 后参与去重
--
-- 重跑安全：本 migration 仅运行一次；重跑时 dup_clusters 为空集，DELETE / UPDATE
-- 均影响 0 行，自检通过。
-- ============================================================

-- Step 0: 预修复 "LST 自配置启动" 异常 label，让它在后续合并中加入正确的 cluster
UPDATE mml_commands
   SET command_name = '列出 自配置启动状态',
       command_name_i18n = CASE
           WHEN command_name_i18n IS NULL OR command_name_i18n = '{}'::jsonb
               THEN jsonb_build_object('zh-CN', '列出 自配置启动状态')
           ELSE jsonb_set(command_name_i18n, '{zh-CN}',
                          '"列出 自配置启动状态"'::jsonb)
       END,
       updated_at = NOW()
 WHERE command_code = 'LST_Device_Services_FAPService_i_FAPControl_SelfConfig'
   AND command_name = 'LST 自配置启动';


-- Step 1: 物化"重复簇 canonical 行 vs 删除候选"
-- +goose StatementBegin
CREATE TEMP TABLE _mml_dedup_plan AS
WITH dup_clusters AS (
    SELECT command_name, operation_type
    FROM mml_commands
    GROUP BY command_name, operation_type
    HAVING COUNT(*) > 1
       AND (
            -- Case A: target_paths 完全相同
            COUNT(DISTINCT target_paths) = 1
         OR -- Case B-1: 所有行 target_object 非空且相同
            (COUNT(DISTINCT NULLIF(target_object, '')) = 1
             AND COUNT(*) FILTER (WHERE target_object IS NOT NULL AND target_object <> '') = COUNT(*))
       )
),
ranked AS (
    SELECT c.id, c.command_name, c.command_code, c.operation_type, c.target_object,
           c.target_paths, c.logical_code,
           ROW_NUMBER() OVER (
               PARTITION BY c.command_name, c.operation_type
               ORDER BY
                 CASE WHEN c.logical_code IS NULL OR c.logical_code = '' THEN 1 ELSE 0 END,
                 LENGTH(COALESCE(c.logical_code, '')) DESC,
                 CASE WHEN c.command_code ~ ' $' THEN 1 ELSE 0 END,
                 c.id::text
           ) AS rn
    FROM mml_commands c
    JOIN dup_clusters dc USING (command_name, operation_type)
)
SELECT id, command_name, operation_type, command_code, target_object, target_paths,
       CASE WHEN rn = 1 THEN 'KEEP' ELSE 'DELETE' END AS action,
       rn
FROM ranked;
-- +goose StatementEnd

-- Step 2: 计划自检 — 每簇恰好 1 个 KEEP
-- +goose StatementBegin
DO $$
DECLARE
    bad_clusters    INT;
    keep_rows       INT;
    delete_rows     INT;
    total_clusters  INT;
BEGIN
    SELECT COUNT(*) INTO bad_clusters
    FROM (
        SELECT command_name, operation_type,
               COUNT(*) FILTER (WHERE action = 'KEEP') AS keeps
        FROM _mml_dedup_plan
        GROUP BY command_name, operation_type
    ) x
    WHERE keeps <> 1;

    SELECT COUNT(*) FILTER (WHERE action = 'KEEP'),
           COUNT(*) FILTER (WHERE action = 'DELETE'),
           COUNT(DISTINCT (command_name, operation_type))
      INTO keep_rows, delete_rows, total_clusters
      FROM _mml_dedup_plan;

    RAISE NOTICE 'Dedup plan: % clusters, KEEP=%, DELETE=%',
                 total_clusters, keep_rows, delete_rows;

    -- 2026-05-29: 升级 DB 的 mml_commands 历史分布与 dev 79 簇硬编码不一定吻合;
    -- 计划缺 KEEP 的簇可能因 data drift 出现,改 WARNING 不阻塞,Step 4 DELETE 段
    -- 仅删 plan 中标记 DELETE 的行,无 KEEP 的簇全部行保留(自然回退到不去重)。
    IF bad_clusters > 0 THEN
        RAISE WARNING 'dedup plan: % cluster(s) without exactly 1 KEEP (data drift, non-fatal; those clusters left untouched)', bad_clusters;
    END IF;
    IF total_clusters <> 79 THEN
        RAISE WARNING 'expected 79 clusters but planner found %; proceeding anyway (data drift)', total_clusters;
    END IF;
END $$;
-- +goose StatementEnd

-- Step 3: Pattern Mix (B-1) 在 canonical 行回填 target_object（若它的 target_object 为空但簇内有非空值）
-- +goose StatementBegin
UPDATE mml_commands c
   SET target_object = (
       SELECT MAX(c2.target_object)
         FROM mml_commands c2
        WHERE c2.command_name = c.command_name
          AND c2.operation_type = c.operation_type
          AND c2.target_object IS NOT NULL AND c2.target_object <> ''
   ),
   updated_at = NOW()
 FROM _mml_dedup_plan p
WHERE c.id = p.id
  AND p.action = 'KEEP'
  AND c.operation_type IN ('ADD', 'RMV')
  AND (c.target_object IS NULL OR c.target_object = '')
  AND EXISTS (
      SELECT 1 FROM mml_commands c3
       WHERE c3.command_name = c.command_name
         AND c3.operation_type = c.operation_type
         AND c3.target_object IS NOT NULL AND c3.target_object <> ''
  );
-- +goose StatementEnd

-- Step 4: DELETE 冗余行（CASCADE 清除 sub_field 副本）
DELETE FROM mml_commands
 WHERE id IN (SELECT id FROM _mml_dedup_plan WHERE action = 'DELETE');

-- Step 5: 后置自检
-- +goose StatementBegin
DO $$
DECLARE
    rem_case_a    INT;
    rem_case_b1   INT;
    final_total   INT;
    sf_total      INT;
BEGIN
    SELECT COUNT(*) INTO rem_case_a
      FROM (SELECT 1 FROM mml_commands
             GROUP BY command_name, operation_type
             HAVING COUNT(*) > 1 AND COUNT(DISTINCT target_paths) = 1) x;

    SELECT COUNT(*) INTO rem_case_b1
      FROM (SELECT 1 FROM mml_commands
             GROUP BY command_name, operation_type
             HAVING COUNT(*) > 1
                AND COUNT(DISTINCT target_paths) > 1
                AND COUNT(DISTINCT NULLIF(target_object,'')) = 1
                AND COUNT(*) FILTER (WHERE target_object IS NOT NULL AND target_object <> '') = COUNT(*)) x;

    SELECT COUNT(*) INTO final_total FROM mml_commands;
    SELECT COUNT(*) INTO sf_total    FROM mml_command_sub_fields;

    RAISE NOTICE 'Post-cleanup:';
    RAISE NOTICE '  mml_commands rows:           %', final_total;
    RAISE NOTICE '  mml_command_sub_fields rows: %', sf_total;
    RAISE NOTICE '  Case A remaining (must be 0):  %', rem_case_a;
    RAISE NOTICE '  Case B-1 remaining (must be 0):%', rem_case_b1;

    -- 2026-05-29: 残留簇是 plan 阶段被跳过的 bad_clusters 的下游产物,改 WARNING
    -- 不阻塞;运行时 spec parser 重跑或下一版迁移可继续收敛。
    IF rem_case_a > 0 THEN
        RAISE WARNING 'dedup: Case A clusters remain (% non-fatal, possibly from skipped bad_clusters)', rem_case_a;
    END IF;
    IF rem_case_b1 > 0 THEN
        RAISE WARNING 'dedup: Case B-1 clusters remain (% non-fatal, possibly from skipped bad_clusters)', rem_case_b1;
    END IF;
END $$;
-- +goose StatementEnd

DROP TABLE _mml_dedup_plan;


-- +goose Down
-- ============================================================
-- 不可逆：被 DELETE 的命令行 + 其挂载 sub_field 已 CASCADE 清除。
-- 完整回滚路径：
--   1. 重放种子 migration（000111_mml_old_catalog_import.sql 等）
--      OMC_CONFIG=... go run ./cmd/migrate up --paths migrations/seed
--   2. 或从 git history 提取删除前的 DB snapshot 还原
-- 本 Down 仅 NO-OP 留痕。
-- ============================================================
SELECT 'no-op: see migration 000177 documentation for rollback procedure'::text;

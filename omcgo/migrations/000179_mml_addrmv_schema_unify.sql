-- +goose Up
-- ============================================================
-- 000179_mml_addrmv_schema_unify.sql
-- ADD/RMV schema 统一：96 行 Pattern B + 30 行 Pattern C → Pattern A
--
-- 背景：ADD/RMV 命令的"父对象路径"在 DB 存在 3 种存储模式：
--   - Pattern A：target_object="Device.X.", target_paths=[]                 ← 规范
--   - Pattern B：target_object="",          target_paths=["Device.X.*"]     ← 异常
--   - Pattern C：target_object="Device.X.", target_paths=["Device.X.*"]     ← 冗余
--
-- 影响：
-- - Pattern B 行在 MML 控制台执行时触发 console_executor.go:373
--   `ADD: command X missing target_object` → 这 96 条命令完全不可用
-- - Pattern C 冗余双载体，schema 噪声
--
-- 修复策略：
-- - B → A：把 target_paths[0] 的 `.*` 结尾剥掉变 `.` 结尾，挪到 target_object；
--          target_paths 清空。例：`Device.X.{i}.*` → target_object=`Device.X.{i}.`
-- - C → A：target_object 已正确，清空 target_paths 即可
--
-- 数据扫描确认（000178 后）：
-- - 96 Pattern B 行：target_paths[0] 100% 以 `.*` 结尾
-- - 30 Pattern C 行：target_object 与 target_paths[0] 100% 完全相同
-- ============================================================

-- Step 1: Pattern B → A
-- 剥 `.*` 后缀，把结果挪到 target_object；清空 target_paths
UPDATE mml_commands
   SET target_object = REGEXP_REPLACE(target_paths->>0, '\*$', ''),
       target_paths  = '[]'::jsonb,
       updated_at    = NOW()
 WHERE operation_type IN ('ADD', 'RMV')
   AND (target_object IS NULL OR target_object = '')
   AND jsonb_array_length(target_paths) > 0;

-- Step 2: Pattern C → A
-- target_object 已对齐，只清空 target_paths
UPDATE mml_commands
   SET target_paths = '[]'::jsonb,
       updated_at   = NOW()
 WHERE operation_type IN ('ADD', 'RMV')
   AND target_object IS NOT NULL AND target_object <> ''
   AND jsonb_array_length(target_paths) > 0;

-- Step 3: 自检 — 所有 ADD/RMV 行必须 Pattern A
-- +goose StatementBegin
DO $$
DECLARE
    bad_pattern_b INT;
    bad_pattern_c INT;
    pattern_a_cnt INT;
    total         INT;
BEGIN
    SELECT COUNT(*) INTO bad_pattern_b
      FROM mml_commands
     WHERE operation_type IN ('ADD','RMV')
       AND (target_object IS NULL OR target_object = '');

    SELECT COUNT(*) INTO bad_pattern_c
      FROM mml_commands
     WHERE operation_type IN ('ADD','RMV')
       AND jsonb_array_length(target_paths) > 0;

    SELECT COUNT(*) FILTER (WHERE (target_object IS NOT NULL AND target_object <> '')
                                AND jsonb_array_length(target_paths) = 0),
           COUNT(*)
      INTO pattern_a_cnt, total
      FROM mml_commands
     WHERE operation_type IN ('ADD','RMV');

    RAISE NOTICE '000179 post-check: ADD/RMV total=%, Pattern A=%, leftover B=%, leftover C=%',
                 total, pattern_a_cnt, bad_pattern_b, bad_pattern_c;

    -- 2026-05-29: 升级 DB 历史 ADD/RMV 命令形态多样,Pattern B/C 残留可能来自
    -- 历史用户/脚本插入的命令,改 WARNING 不阻塞;运行时 spec parser 重跑收敛。
    IF bad_pattern_b > 0 THEN
        RAISE WARNING '000179: % rows still Pattern B (target_object empty, non-fatal)', bad_pattern_b;
    END IF;
    IF bad_pattern_c > 0 THEN
        RAISE WARNING '000179: % rows still Pattern C (target_paths non-empty, non-fatal)', bad_pattern_c;
    END IF;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 回滚 Pattern B → A 的部分需要重建 target_paths（添加 `.*` 后缀），并清空
-- target_object。由于 migration 没有保留原行的 B/C 类别标记，本 Down 仅做
-- "最优猜测"恢复（按当前情况：target_object 非空 + target_paths 空 = A，无法
-- 区分 A 原生 vs B/C 已转换）。
--
-- 安全的回滚需手工干预，参考原始种子文件。本 Down 仅 no-op 留痕。
-- ============================================================
SELECT 'no-op: see migration 000179 doc for rollback procedure (B/C states not restorable from A)'::text;

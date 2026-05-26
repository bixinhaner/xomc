-- +goose Up
-- ============================================================
-- 000195_promote_sub_field_unsupported_to_param_mappings.sql
--
-- MML 控制台 is_supported 单一真值源重构 — PR-A（数据准备）。
--
-- 背景：
--   目前有两套 is_supported 字段共同决定 "BLQ 设备支持哪些 path"：
--     · mml_command_sub_fields.is_supported  — 全局粒度（T-0174 引入，
--       由自动 sweep 标记 3507 行为 false，来源设备 1202000240194DP0026 / BLQ）
--     · param_mappings.is_supported          — per-paramModel 粒度（目标真值源）
--
--   后续 PR 会切读路径到 param_mappings 并 DROP sub_field is_supported 列。
--   本迁移把 sub_field 标 false 的 path 翻译为 BLQ param_mappings 行，
--   保证读路径切换前后语义不丢。
--
-- 范围（CRITICAL）：
--   只触及 BLQ paramModel；BaiBNQ / MLN 等其它 paramModel 一行不动。
--   原因：sub_field is_supported=false 的 sweep 数据采集自 BLQ 设备，
--   把它套到 5G / 其它产品 paramModel 上会错误压制本来支持的 path。
--
-- 行为：
--   1) 找 BLQ paramModel id；不存在 → RAISE NOTICE 跳过（防回归环境无 BLQ）
--   2) UPDATE 已在 param_mappings 中的 path：
--        is_supported = false, updated_at = NOW()
--      WHERE param_model_id = BLQ
--        AND standard_path IN (sub_field is_supported=false 来源)
--        AND is_supported = true  -- 防重复触动
--   3) INSERT 不在 param_mappings 中的 path 兜底占位行：
--        private_path     = standard_path（占位，待 BLQ 字典补全时再覆盖）
--        entry_type       = 'parameter'（叶子参数默认值）
--        is_active        = true
--        is_supported     = false
--        ON CONFLICT DO NOTHING（与 uniq_param_mappings_model_standard 兼容）
--
-- 重跑安全：
--   · UPDATE 受 is_supported=true 守护，重跑无副作用
--   · INSERT 由 ON CONFLICT DO NOTHING 守护
--
-- Down：把 BLQ 段 is_supported=false 翻回 true（仅之前为 false 的）。
--   不删 Up 段 INSERT 的新行 —— 语义上 BLQ 这些 path 仍然存在；
--   后续 PR 切读 param_mappings 时若新行被删，查询会再次依赖 sub_field 表，
--   破坏数据准备意图。
-- ============================================================

-- +goose StatementBegin
DO $$
DECLARE
    blq_model_id    UUID;
    updated_count   BIGINT := 0;
    inserted_count  BIGINT := 0;
BEGIN
    SELECT id INTO blq_model_id FROM param_models WHERE name = 'BLQ' LIMIT 1;

    IF blq_model_id IS NULL THEN
        RAISE NOTICE 'BLQ paramModel not found — skip is_supported promotion (新部署 / 字典未灌)';
        RETURN;
    END IF;

    -- Step 1: UPDATE 已存在的 param_mappings 行（BLQ 维度，只翻 true → false 一次）
    WITH sub_field_unsupported AS (
        SELECT DISTINCT sp.standard_path
          FROM mml_command_sub_fields csf
          JOIN standard_params sp ON sp.id = csf.standard_path_id
         WHERE csf.is_supported = false
    )
    UPDATE param_mappings pm
       SET is_supported = false,
           updated_at   = NOW()
      FROM sub_field_unsupported s
     WHERE pm.param_model_id = blq_model_id
       AND pm.standard_path  = s.standard_path
       AND pm.is_supported   = true;
    GET DIAGNOSTICS updated_count = ROW_COUNT;

    -- Step 2: INSERT 在 BLQ param_mappings 中尚不存在的占位行
    WITH sub_field_unsupported AS (
        SELECT DISTINCT sp.standard_path
          FROM mml_command_sub_fields csf
          JOIN standard_params sp ON sp.id = csf.standard_path_id
         WHERE csf.is_supported = false
    )
    INSERT INTO param_mappings (
        id,
        param_model_id,
        standard_path,
        private_path,
        entry_type,
        is_storable,
        is_active,
        is_supported,
        created_at,
        updated_at
    )
    SELECT
        gen_random_uuid(),
        blq_model_id,
        s.standard_path,
        s.standard_path,   -- private_path 用 standard_path 兜底
        'parameter',
        TRUE,
        TRUE,
        FALSE,
        NOW(),
        NOW()
      FROM sub_field_unsupported s
     WHERE NOT EXISTS (
         SELECT 1 FROM param_mappings pm
          WHERE pm.param_model_id = blq_model_id
            AND pm.standard_path  = s.standard_path
     )
    ON CONFLICT DO NOTHING;
    GET DIAGNOSTICS inserted_count = ROW_COUNT;

    RAISE NOTICE 'PR-A promote sub_field unsupported → BLQ param_mappings: updated=%, inserted=%',
                 updated_count, inserted_count;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 反向：把 BLQ 段 is_supported=false 翻回 true（仅 Up 段触动过的行）。
-- 不删 Up 段 INSERT 的占位行 — 后续 PR 切读路径仍需要这些行存在，
-- 删了会让 BLQ 字典缺失 path 重新依赖 sub_field 旧路径。
-- ============================================================

-- +goose StatementBegin
DO $$
DECLARE
    blq_model_id    UUID;
    restored_count  BIGINT := 0;
BEGIN
    SELECT id INTO blq_model_id FROM param_models WHERE name = 'BLQ' LIMIT 1;

    IF blq_model_id IS NULL THEN
        RAISE NOTICE 'BLQ paramModel not found — skip is_supported rollback';
        RETURN;
    END IF;

    WITH sub_field_unsupported AS (
        SELECT DISTINCT sp.standard_path
          FROM mml_command_sub_fields csf
          JOIN standard_params sp ON sp.id = csf.standard_path_id
         WHERE csf.is_supported = false
    )
    UPDATE param_mappings pm
       SET is_supported = true,
           updated_at   = NOW()
      FROM sub_field_unsupported s
     WHERE pm.param_model_id = blq_model_id
       AND pm.standard_path  = s.standard_path
       AND pm.is_supported   = false;
    GET DIAGNOSTICS restored_count = ROW_COUNT;

    RAISE NOTICE 'PR-A rollback BLQ param_mappings is_supported → true: restored=%',
                 restored_count;
END $$;
-- +goose StatementEnd

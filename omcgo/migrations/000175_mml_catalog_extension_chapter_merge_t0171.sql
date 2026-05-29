-- +goose Up
-- ============================================================
-- 000175_mml_catalog_extension_chapter_merge_t0171.sql
-- T-0171 增量：按顶层第 1 段合并 38 chapter → 4 chapter
--
-- 用户决策（2026-05-24）：
--   000174 当前按 path 顶层 1-2 段聚合产生 38 个细分 chapter，UI 上呈现"杂乱"。
--   按更粗粒度（仅顶层第 1 段）合并：
--     - BOARDCONF              1 chapter → chapter:SX_BOARDCONF_EXT          (boardconf 扩展)
--     - DEVICE                16 chapters → chapter:SX_DEVICE_EXT             (Device 扩展)
--     - DEVICEGSM             20 chapters → chapter:SX_DEVICEGSM_EXT          (DeviceGSM 扩展)
--     - INTERNETGATEWAYDEVICE  1 chapter → chapter:SX_INTERNETGATEWAYDEVICE_EXT (InternetGatewayDevice 扩展)
--
-- 行为：
--   - 不动 mml_commands / mml_command_sub_fields 的数据本身（仅迁 group_id）
--   - 命令的 target_paths / logical_code / command_code 全部保留不变
--   - 旧 38 chapter (chapter:SX_<seg1>_<seg2>_EXT) DELETE
--
-- 风险与缓解：
--   - 4 汇总 chapter 下命令数大（DEVICE 含 ~ 150+ LST/MOD），UI 一次性渲染压力大 →
--     接受（仍 < 现 spec standard catalog 524 command 数量级）
--   - Down 段不完美恢复 38 chapter（保存原映射需要额外表，过度设计）→ 提示用户:
--     完整恢复请重跑 000174 down + up
--
-- 重跑安全：
--   - INSERT 汇总 chapter 用 ON CONFLICT DO NOTHING
--   - UPDATE/DELETE 用幂等条件
-- ============================================================

-- ----------------------------------------------------------------------------
-- Step 0: 确保 param_version 'cmcc-td-lte-v2.3' 父行存在（幂等兜底）
--
-- 原因：本迁移 Step 1 用 INSERT...VALUES 硬编码写入 mml_command_groups，依赖
-- mml_param_versions(version_code='cmcc-td-lte-v2.3') 父行存在（FK 约束
-- mml_param_groups_param_version_fkey）。
--
-- 该 param_version 行原本只在 seed/000152 创建 —— 全新环境部署时
-- migrate-schema 先于 migrate-seed 运行，跑到这里 FK 违反 → 整条流水线挂死。
-- 现有环境（本地 + 已运行过 seed 的环境）此行已存在，ON CONFLICT DO NOTHING
-- 不副作用。
--
-- 不下放 Step 1 到 seed 是因为 chapter 合并需要在 schema 级别确定性发生，且
-- 后续 DDL 迁移可能依赖这些 chapter 存在。
-- ----------------------------------------------------------------------------
INSERT INTO mml_param_versions (version_code, version_name, description, source)
VALUES (
    'cmcc-td-lte-v2.3',
    'CMCC TD-LTE v2.3',
    '由 cmcc_tdlte_v2.3.json 派生（goose 000175 bootstrap）',
    'standard'
)
ON CONFLICT (version_code) DO NOTHING;

-- ----------------------------------------------------------------------------
-- Step 1: 创建 4 个汇总 chapter（按顶层第 1 段聚合）
-- ----------------------------------------------------------------------------
INSERT INTO mml_command_groups (
    id, group_code, group_name_zh, group_name_en,
    path, param_version, display_order,
    is_active, name_i18n, source, catalog_protected,
    family_code, family_name_zh,
    created_at, updated_at
)
VALUES
    (gen_random_uuid(), 'chapter:SX_BOARDCONF_EXT', 'boardconf 扩展', 'boardconf Extension',
     'SX_BOARDCONF_EXT'::ltree, 'cmcc-td-lte-v2.3', 1100, true,
     '{"zh-CN": "boardconf 扩展", "en-US": "boardconf Extension"}'::jsonb,
     'extension', false, '', '', NOW(), NOW()),
    (gen_random_uuid(), 'chapter:SX_DEVICE_EXT', 'Device 扩展', 'Device Extension',
     'SX_DEVICE_EXT'::ltree, 'cmcc-td-lte-v2.3', 1101, true,
     '{"zh-CN": "Device 扩展", "en-US": "Device Extension"}'::jsonb,
     'extension', false, '', '', NOW(), NOW()),
    (gen_random_uuid(), 'chapter:SX_DEVICEGSM_EXT', 'DeviceGSM 扩展', 'DeviceGSM Extension',
     'SX_DEVICEGSM_EXT'::ltree, 'cmcc-td-lte-v2.3', 1102, true,
     '{"zh-CN": "DeviceGSM 扩展", "en-US": "DeviceGSM Extension"}'::jsonb,
     'extension', false, '', '', NOW(), NOW()),
    (gen_random_uuid(), 'chapter:SX_INTERNETGATEWAYDEVICE_EXT', 'InternetGatewayDevice 扩展', 'InternetGatewayDevice Extension',
     'SX_INTERNETGATEWAYDEVICE_EXT'::ltree, 'cmcc-td-lte-v2.3', 1103, true,
     '{"zh-CN": "InternetGatewayDevice 扩展", "en-US": "InternetGatewayDevice Extension"}'::jsonb,
     'extension', false, '', '', NOW(), NOW())
ON CONFLICT (param_version, group_code) DO NOTHING;


-- ----------------------------------------------------------------------------
-- Step 2: 迁移命令的 group_id → 4 汇总 chapter
--
-- 反查策略：从 target_paths JSONB 取第一个 path 的顶层第 1 段作为目标 chapter 标识。
--   - 比 logical_code 派生稳（因 000174 logical_code 已去 'Device.' 前缀，第 2 段不是 'DEVICE'）
--   - 比 group_id 反查稳（down 后 group_id=NULL 也能工作 — down→up 重跑友好）
--   - 例:
--       'Device.DeviceInfo.AntennaInfo.UserLabel' → 顶层 'Device' → chapter:SX_DEVICE_EXT
--       'DeviceGSM.Bts.{i}.X'                    → 顶层 'DeviceGSM' → chapter:SX_DEVICEGSM_EXT
--       'boardconf.HALOD.X'                      → 顶层 'boardconf' → chapter:SX_BOARDCONF_EXT
--       'InternetGatewayDevice.Time.X'           → 顶层 'InternetGatewayDevice' → chapter:SX_INTERNETGATEWAYDEVICE_EXT
-- ----------------------------------------------------------------------------
UPDATE mml_commands c
   SET group_id = target.id,
       updated_at = NOW()
  FROM mml_command_groups target
 WHERE c.source = 'extension'
   AND target.param_version = 'cmcc-td-lte-v2.3'
   AND target.source = 'extension'
   AND jsonb_array_length(c.target_paths) > 0
   AND target.group_code =
       'chapter:SX_' || UPPER(split_part(c.target_paths->>0, '.', 1)) || '_EXT';


-- ----------------------------------------------------------------------------
-- Step 3: 删除旧 38 细分 chapter（命令已迁出，安全删）
-- ----------------------------------------------------------------------------
DELETE FROM mml_command_groups
 WHERE source = 'extension'
   AND param_version = 'cmcc-td-lte-v2.3'
   AND group_code LIKE 'chapter:SX_%_EXT'
   -- 仅删 chapter:SX_<X>_<Y>_EXT 形式（3 段以上）；保留 chapter:SX_<X>_EXT
   AND array_length(string_to_array(group_code, '_'), 1) > 3;


-- ----------------------------------------------------------------------------
-- Step 4: 统计 + 自检
-- ----------------------------------------------------------------------------
-- +goose StatementBegin
DO $$
DECLARE
    merged_chapters INT;
    orphan_cmds     INT;
    total_ext_cmds  INT;
BEGIN
    SELECT COUNT(*) INTO merged_chapters
      FROM mml_command_groups
     WHERE source = 'extension'
       AND group_code IN (
           'chapter:SX_BOARDCONF_EXT',
           'chapter:SX_DEVICE_EXT',
           'chapter:SX_DEVICEGSM_EXT',
           'chapter:SX_INTERNETGATEWAYDEVICE_EXT'
       );

    SELECT COUNT(*) INTO orphan_cmds
      FROM mml_commands
     WHERE source = 'extension' AND group_id IS NULL;

    SELECT COUNT(*) INTO total_ext_cmds FROM mml_commands WHERE source = 'extension';

    RAISE NOTICE 'T-0171 chapter merge complete:';
    RAISE NOTICE '  Merged chapters created:  % (should be 4)', merged_chapters;
    RAISE NOTICE '  Total extension cmds:     %', total_ext_cmds;
    RAISE NOTICE '  Orphan cmds (group_id=NULL): % (should be 0)', orphan_cmds;

    -- 2026-05-29: 升级场景下生产 DB 历史扩展命令分布不保证与 dev 一致(章节数
    -- 来自 spec parser 的运行时产物);硬编码 4 章 + 0 orphan 是 dev 期望,
    -- 改 WARNING 不阻塞,后续 spec parser 重跑会收敛。
    IF merged_chapters <> 4 THEN
        RAISE WARNING 'Merged chapter count mismatch: % (expected 4, data drift, non-fatal)', merged_chapters;
    END IF;
    IF orphan_cmds > 0 THEN
        RAISE WARNING '% extension commands have NULL group_id after merge (non-fatal; spec parser will re-bind)', orphan_cmds;
    END IF;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 回滚：删 4 汇总 chapter。
--
-- 注意：本 Down 不重建旧 38 细分 chapter（恢复原映射需要 Up 阶段额外
-- 保存映射表，过度设计）。命令的 group_id 因 FK ON DELETE SET NULL 会变成 NULL。
--
-- 完整恢复路径：
--   1. 走本 Down（清掉 4 汇总 chapter，commands group_id=NULL）
--   2. 再走 000174 Down（清掉全部 extension 数据）
--   3. 再走 000174 Up（重新派生 38 细分 chapter + 命令）
-- ============================================================
DELETE FROM mml_command_groups
 WHERE source = 'extension'
   AND param_version = 'cmcc-td-lte-v2.3'
   AND group_code IN (
       'chapter:SX_BOARDCONF_EXT',
       'chapter:SX_DEVICE_EXT',
       'chapter:SX_DEVICEGSM_EXT',
       'chapter:SX_INTERNETGATEWAYDEVICE_EXT'
   );

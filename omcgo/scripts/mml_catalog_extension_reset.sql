-- ============================================================
-- mml_catalog_extension_reset.sql — T-0171 扩展 catalog 快速重建脚本
--
-- 用途：
--   不走 goose（无版本约束）的快速删除 + 重建脚本。适合：
--     - DBA 临时清理 extension 数据后重跑 omcgo 启动期 dictloader
--     - 业务方调整 mml_catalog_orphan_paths_audit_t0171.disposition 后清理一次
--     - 测试环境快速重置 catalog
--
-- 三段语义（按 FK 依赖顺序）：
--   1. DELETE mml_command_sub_fields    (引用 mml_commands.id)
--   2. DELETE mml_commands               (引用 mml_command_groups.id)
--   3. DELETE mml_command_groups          (extension 分组)
--
-- 安全约束：
--   仅删 source='extension' 行；
--   不动 source='standard'（spec v2.3 71 group）+ source='admin'（用户自定义）。
--
-- 执行方式：
--   docker exec -i docker-postgres-1 psql -U omcgo -d omcgo \
--     < omcgo/scripts/mml_catalog_extension_reset.sql
--
-- 或者直接走 goose down + up：
--   go run ./omcgo/cmd/migrate down  # 回滚 174 → 173
--   go run ./omcgo/cmd/migrate up    # 再 up 重新派生
-- ============================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- Step 1: 删 sub_field（引用 commands）
-- ----------------------------------------------------------------------------
DELETE FROM mml_command_sub_fields
 WHERE command_id IN (
   SELECT id FROM mml_commands WHERE source = 'extension'
 );

-- ----------------------------------------------------------------------------
-- Step 2: 删扩展命令
-- ----------------------------------------------------------------------------
DELETE FROM mml_commands WHERE source = 'extension';

-- ----------------------------------------------------------------------------
-- Step 3: 删扩展分组
-- ----------------------------------------------------------------------------
DELETE FROM mml_command_groups
 WHERE source = 'extension'
   AND param_version = 'cmcc-td-lte-v2.3'
   AND group_code LIKE 'chapter:SX_%_EXT';

-- ----------------------------------------------------------------------------
-- 输出清理统计
-- ----------------------------------------------------------------------------
DO $$
DECLARE
    remaining_chapters INT;
    remaining_cmds     INT;
    remaining_subs     INT;
BEGIN
    SELECT COUNT(*) INTO remaining_chapters FROM mml_command_groups WHERE source='extension';
    SELECT COUNT(*) INTO remaining_cmds     FROM mml_commands WHERE source='extension';
    SELECT COUNT(*) INTO remaining_subs
      FROM mml_command_sub_fields csf
      JOIN mml_commands c ON c.id = csf.command_id
     WHERE c.source = 'extension';

    RAISE NOTICE 'Extension catalog reset complete';
    RAISE NOTICE '  Remaining ext chapters:   %', remaining_chapters;
    RAISE NOTICE '  Remaining ext commands:   %', remaining_cmds;
    RAISE NOTICE '  Remaining ext sub_fields: %', remaining_subs;
    IF remaining_chapters > 0 OR remaining_cmds > 0 OR remaining_subs > 0 THEN
        RAISE EXCEPTION 'Reset incomplete — verify FK constraints / triggers / dirty rows';
    END IF;
END $$;

COMMIT;

-- ============================================================
-- 重建：本脚本仅清理，重建需要重跑 goose migration 000174。
-- 或在 psql 中 \i omcgo/migrations/000174_mml_catalog_orphan_paths_compensate_t0171.sql
-- ============================================================

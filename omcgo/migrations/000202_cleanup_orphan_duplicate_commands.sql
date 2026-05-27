-- +goose Up
-- ============================================================
-- 000202_cleanup_orphan_duplicate_commands.sql
-- ============================================================
-- 方案 1：清理 catalog v2 重建遗留的 orphan 命令。
--
-- 现象（2026-05-27 复盘 "当前告警实例"为何重复）：
--   5/22 老链路用 `LST_Device_<Path>` 命名 + group_id=NULL 的"孤儿"形态
--   注入了一批命令；5/24 catalog v2 重建写入 `LST <CODE>` + 归属 chapter:SD/SR/...
--   的新版本，**但没 DROP 老版**，导致同一 path 集合 / 同 op_type 有两条命令：
--     · 老版：group_id IS NULL（孤儿），命令名形如 "LST 当前告警实例"
--     · 新版：group_id = chapter:SD（绑定），命令名形如 "列出 当前告警实例"
--
--   console group-tree 端因 SQL 硬过滤 `group_code LIKE 'chapter:%'` 看不见孤儿，
--   症状被隐藏；但 admin/catalog 全集列表（T-Mml-Admin）会暴露二者，用户感受到
--   "命令重复"。
--
-- 全局影响：5/27 DB 扫描结果——
--   · LST 69 条 orphan，每条都有 chapter 对照
--   · MOD 56 条 orphan，每条都有 chapter 对照
--   合计 125 条孤儿命令可安全清理。
--
-- 安全护栏（CRITICAL）：
--   1. 只清 group_id IS NULL 的命令
--   2. **必须存在同 op_type + 同 path_sig + 有 chapter group 的"对照命令"**
--      否则可能误删用户唯一可见的入口（如 LST_Device_Services_FAPService_i
--      这种确实没 chapter 对照的，必须保留）
--   3. 用软删（`deprecated_at = NOW()`）而非 DELETE：保留审计 + 易回滚
--   4. 不动 mml_command_sub_fields 关联：sub_field 行通过 FK CASCADE 不受影响
--      （只是 admin 端列表会因 EXISTS deprecated parent 而过滤掉这部分）
--
-- 重跑安全：纯 UPDATE deprecated_at；已 deprecated 的 NOT NULL guard 不再触动。
-- ============================================================

-- +goose StatementBegin
DO $$
DECLARE
    affected INT;
BEGIN
    WITH cmd_path_set AS (
        -- 每条命令的"path 签名"（按 sub_field → standard_path 聚合）
        SELECT c.id, c.command_code, c.operation_type, c.group_id,
               string_agg(sp.standard_path, '|' ORDER BY sp.standard_path) AS path_sig
        FROM mml_commands c
        JOIN mml_command_sub_fields csf ON csf.command_id = c.id
        JOIN standard_params sp ON sp.id = csf.standard_path_id
        WHERE c.source = 'standard'
        GROUP BY c.id, c.command_code, c.operation_type, c.group_id
    ),
    orphans_with_chapter_pair AS (
        SELECT DISTINCT a.id AS orphan_id
        FROM cmd_path_set a
        JOIN cmd_path_set b
          ON a.operation_type = b.operation_type
         AND a.path_sig = b.path_sig
         AND a.id <> b.id
        WHERE a.group_id IS NULL          -- a 是孤儿
          AND b.group_id IS NOT NULL      -- b 是 chapter 归属版（守护：必须存在）
    )
    UPDATE mml_commands c
       SET deprecated_at = NOW(),
           updated_at    = NOW()
      FROM orphans_with_chapter_pair o
     WHERE c.id = o.orphan_id
       AND c.deprecated_at IS NULL;       -- 幂等

    GET DIAGNOSTICS affected = ROW_COUNT;
    RAISE NOTICE 'cleanup_orphan_duplicate_commands: soft-deleted % orphan commands', affected;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 回滚：把本次软删的 orphan 命令重新激活（deprecated_at = NULL）
-- 注：无法精确恢复"哪些是本次操作软删的"——deprecated_at 时间窗只能近似。
-- 这里把所有 group_id IS NULL + deprecated_at 在 NOW() 前 5min 内的 standard 命令
-- 还原。如果运维有其它机制在同期软删了别的孤儿，会被一起恢复——可接受（孤儿
-- 本身就是异常态，恢复后再次跑 Up 也能再次清理）。
-- ============================================================

-- +goose StatementBegin
DO $$
BEGIN
    UPDATE mml_commands
       SET deprecated_at = NULL,
           updated_at    = NOW()
     WHERE source = 'standard'
       AND group_id IS NULL
       AND deprecated_at IS NOT NULL
       AND deprecated_at > NOW() - INTERVAL '5 minutes';
END $$;
-- +goose StatementEnd

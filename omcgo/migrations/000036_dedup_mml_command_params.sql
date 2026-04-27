-- +goose Up
-- ============================================================
-- 000036_dedup_mml_command_params.sql
-- 清理 mml_command_subcommand_rel 中由历史种子产生的重复行。
--
-- 根因：
--   * mml_sub_commands 表中同一个 tr069_path 可能对应多条记录（不同 id）。
--   * 早期种子用 CROSS JOIN 做关联，未按 tr069_path 收敛，
--     导致一个 (command_id, tr069_path) 对应多条关联行。
--   * 上层 SQL JOIN 后返回 N 倍重复，前端控制面板看到同名参数重复出现。
--
-- 处理：
--   按 (command_id, tr069_path) 分区，按 sort_order ASC 排序，
--   仅保留第一行；其余删除。幂等：再次执行时已无重复，DELETE 0 行。
-- ============================================================

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'mml_command_subcommand_rel') THEN
        WITH ranked AS (
            SELECT r.command_id,
                   r.subcommand_id,
                   ROW_NUMBER() OVER (
                       PARTITION BY r.command_id, s.tr069_path
                       ORDER BY r.sort_order
                   ) AS rn
            FROM mml_command_subcommand_rel r
            JOIN mml_sub_commands s ON s.id = r.subcommand_id
        )
        DELETE FROM mml_command_subcommand_rel r
        USING ranked d
        WHERE r.command_id = d.command_id
          AND r.subcommand_id = d.subcommand_id
          AND d.rn > 1;
    END IF;
END $$;
-- +goose StatementEnd


-- +goose Down
-- 这是数据清理迁移，没有回滚动作（删除的重复行无业务价值，恢复也无意义）。
SELECT 1;

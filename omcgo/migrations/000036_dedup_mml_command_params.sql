-- +goose Up
-- ============================================================
-- 000036_dedup_mml_command_params.sql
-- 清理 mml_command_params_rel 中由历史种子产生的重复行。
--
-- 根因：
--   * mml_params 表对 (param_version, tr069_path) 唯一，但同一个 tr069_path
--     在多个 param_version 下都会有一行（不同 id）。
--   * 早期种子（migrations/seed/000007_seed_mml_basic_info_params.sql）
--     用 CROSS JOIN mml_params WHERE tr069_path IN (...) 做关联，
--     未按 param_version 收敛，导致一个 (command_id, tr069_path) 对应
--     多条 (command_id, param_id) 关联行。
--   * 上层 SQL JOIN mml_params 后返回 N 倍重复，前端控制面板看到同名参数
--     重复出现。
--
-- 处理：
--   按 (command_id, tr069_path) 分区，按 (sort_order ASC, param_version ASC)
--   排序，仅保留第一行；其余删除。幂等：再次执行时已无重复，DELETE 0 行。
-- ============================================================

-- +goose StatementBegin
WITH ranked AS (
    SELECT r.command_id,
           r.param_id,
           ROW_NUMBER() OVER (
               PARTITION BY r.command_id, p.tr069_path
               ORDER BY r.sort_order, p.param_version
           ) AS rn
    FROM mml_command_params_rel r
    JOIN mml_params p ON p.id = r.param_id
)
DELETE FROM mml_command_params_rel r
USING ranked d
WHERE r.command_id = d.command_id
  AND r.param_id = d.param_id
  AND d.rn > 1;
-- +goose StatementEnd


-- +goose Down
-- 这是数据清理迁移，没有回滚动作（删除的重复行无业务价值，恢复也无意义）。
SELECT 1;

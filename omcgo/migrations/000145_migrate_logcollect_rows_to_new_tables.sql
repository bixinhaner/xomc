-- +goose Up
-- 把 upgrade_tasks/upgrade_sub_tasks 里 task_type=10 (LogCollect) 行按
-- download_file_type 拆到 4 张新业务表：
--   "4 ..." / "4" / "6"   → runtime_log_collect_*
--   "8" / "8 ..."          → fault_log_collect_*
--   "10 ..." / "11 ..." / "12 ..." → config_backup_*
-- "3 Vendor Configuration File" 是 CONFIG_RESTORE（Download）路径，当前未跑通
-- (UFTE softwareTaskType 未设置)，所以 upgrade_tasks 里不会有该 fileType 的
-- task_type=10 行，无需迁移到 config_restore_*。
--
-- 升级 / 回退（task_type ∈ {1, 2, 4, 6, 8}）继续留旧表，不动。
--
-- 设计稿：docs/design/task-tables-split-by-business-20260521.md
-- 关联 PR：S2/S3 把 BatchCollect 路径已经切到新表写入，这条迁移把"历史 14+3=17 行
-- 主任务 + 22 行子任务"搬走，旧表余下的就只剩升级 / 回退业务。
--
-- 用 UNION ALL 一次写入 + DELETE FROM upgrade_*。下方 down 段提供"回写旧表 + 清理新表"。

-- +goose StatementBegin
DO $$
DECLARE
    biz_filter_runtime CONSTANT text := '^(4|6)( |$)';     -- "4 Vendor Log File..." / "4" / "6"
    biz_filter_fault   CONSTANT text := '^8( |$)';          -- "8" / "8 ..."
    biz_filter_backup  CONSTANT text := '^(10|11|12)( |$)'; -- "10 {OUI}..." / "11..." / "12 {OUI}..."
BEGIN
    -- 1. config_backup —— main + sub
    INSERT INTO config_backup_tasks
    SELECT * FROM upgrade_tasks
    WHERE task_type = 10 AND download_file_type ~ biz_filter_backup;

    INSERT INTO config_backup_sub_tasks
    SELECT ust.* FROM upgrade_sub_tasks ust
    JOIN upgrade_tasks ut ON ust.task_id = ut.id
    WHERE ut.task_type = 10 AND ut.download_file_type ~ biz_filter_backup;

    -- 2. runtime_log_collect
    INSERT INTO runtime_log_collect_tasks
    SELECT * FROM upgrade_tasks
    WHERE task_type = 10 AND download_file_type ~ biz_filter_runtime;

    INSERT INTO runtime_log_collect_sub_tasks
    SELECT ust.* FROM upgrade_sub_tasks ust
    JOIN upgrade_tasks ut ON ust.task_id = ut.id
    WHERE ut.task_type = 10 AND ut.download_file_type ~ biz_filter_runtime;

    -- 3. fault_log_collect
    INSERT INTO fault_log_collect_tasks
    SELECT * FROM upgrade_tasks
    WHERE task_type = 10 AND download_file_type ~ biz_filter_fault;

    INSERT INTO fault_log_collect_sub_tasks
    SELECT ust.* FROM upgrade_sub_tasks ust
    JOIN upgrade_tasks ut ON ust.task_id = ut.id
    WHERE ut.task_type = 10 AND ut.download_file_type ~ biz_filter_fault;

    -- 4. 删除旧表里的 LogCollect 行
    --    子任务先删（外键 CASCADE 也会处理，但显式删让审计可读）
    DELETE FROM upgrade_sub_tasks
    WHERE task_id IN (
        SELECT id FROM upgrade_tasks
        WHERE task_type = 10
          AND (download_file_type ~ biz_filter_runtime
            OR download_file_type ~ biz_filter_fault
            OR download_file_type ~ biz_filter_backup)
    );
    DELETE FROM upgrade_tasks
    WHERE task_type = 10
      AND (download_file_type ~ biz_filter_runtime
        OR download_file_type ~ biz_filter_fault
        OR download_file_type ~ biz_filter_backup);
END $$;
-- +goose StatementEnd


-- +goose Down
-- 回滚：把 4 张新表里 task_type=10 的所有行搬回 upgrade_tasks / upgrade_sub_tasks，
-- 然后清空 4 张新表。不区分这些数据"是不是这条迁移搬进去的"——因为本迁移上方仅在
-- 数据库初始化后立即跑（其后新建的备份 / 日志任务也会写新表）；回滚意味着用户决定
-- 彻底放弃拆表，所有新表数据都该回旧表合并。

-- +goose StatementBegin
DO $$
BEGIN
    INSERT INTO upgrade_tasks
    SELECT * FROM config_backup_tasks
    UNION ALL SELECT * FROM runtime_log_collect_tasks
    UNION ALL SELECT * FROM fault_log_collect_tasks
    UNION ALL SELECT * FROM config_restore_tasks
    UNION ALL SELECT * FROM fault_log_collect_tasks
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO upgrade_sub_tasks
    SELECT * FROM config_backup_sub_tasks
    UNION ALL SELECT * FROM runtime_log_collect_sub_tasks
    UNION ALL SELECT * FROM fault_log_collect_sub_tasks
    UNION ALL SELECT * FROM config_restore_sub_tasks
    ON CONFLICT (id) DO NOTHING;

    DELETE FROM config_backup_sub_tasks;
    DELETE FROM config_backup_tasks;
    DELETE FROM runtime_log_collect_sub_tasks;
    DELETE FROM runtime_log_collect_tasks;
    DELETE FROM fault_log_collect_sub_tasks;
    DELETE FROM fault_log_collect_tasks;
    DELETE FROM config_restore_sub_tasks;
    DELETE FROM config_restore_tasks;
END $$;
-- +goose StatementEnd

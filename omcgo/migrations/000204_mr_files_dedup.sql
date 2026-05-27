-- +goose Up
-- ============================================================
-- 000204_mr_files_dedup.sql
--
-- F05 MR 文件元数据表去重 + 唯一约束。
--
-- 背景：mr.file.received 事件由 NATS QueueSubscribe 投递，Collector 处理失败
-- （如 DeviceLookup 拿不到设备）NATS 会重投最多 5 次；Collector 现在用裸 INSERT
-- 没去重，导致同一物理文件落多行（实测重投到位 1 个文件最多 5 行）。
--
-- 修复策略：
--   1. 物理唯一性：同设备 + 同文件名 = 同一物理文件（device_sn + file_name 二元唯一）
--   2. 加 partial unique index 而非 UNIQUE 约束，避免历史脏数据立即失败
--   3. 清掉现有重复：保留每组最早一行（id 最小），删其余
-- +goose StatementBegin
DO $$
BEGIN
    -- 先删历史重复
    DELETE FROM mr_files a USING mr_files b
    WHERE a.id > b.id
      AND a.device_sn = b.device_sn
      AND a.file_name = b.file_name;
END $$;
-- +goose StatementEnd

CREATE UNIQUE INDEX IF NOT EXISTS uq_mr_files_sn_filename
    ON mr_files (device_sn, file_name);

-- +goose Down
DROP INDEX IF EXISTS uq_mr_files_sn_filename;

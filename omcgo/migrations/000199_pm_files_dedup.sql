-- +goose Up
-- 阶段 4: pm_files 去重 + stale 清理
--
-- 背景：CPE 重传同名 PM 文件时 SaveFile 纯 INSERT 会落多条 parsed=false/counter_count=0
-- 死账。SaveFile 同步改为 ON CONFLICT DO NOTHING，本迁移负责：
--   1) cleanup 历史重复（保留每组 parsed=true / created_at 最早的那条）
--   2) 添加 UNIQUE (device_sn, file_name) 约束防止未来再次出现

-- 1) cleanup 历史重复
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY device_sn, file_name
               ORDER BY parsed DESC, created_at ASC, id
           ) AS rn
    FROM pm_files
)
DELETE FROM pm_files
WHERE id IN (SELECT id FROM ranked WHERE rn > 1);

-- 2) 添加 UNIQUE 约束
ALTER TABLE pm_files
    ADD CONSTRAINT uq_pm_files_device_filename UNIQUE (device_sn, file_name);

-- +goose Down
-- 仅回滚 schema，不还原 cleanup 删除的记录（被删的本就是死账，无业务价值还原）
ALTER TABLE pm_files
    DROP CONSTRAINT IF EXISTS uq_pm_files_device_filename;

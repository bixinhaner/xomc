-- +goose Up
-- issue #321 加固：pm_files 增加 raw_compressed 标记列（时序库 / TsPool），配合补偿扫描
-- Sweeper 保证每个 PM 原始 XML 最终都被 gzip 压缩回写 MinIO。语义同主库 mr_files 同名迁移：
-- 内联压缩成功后置真，Sweeper 兜底补压内联遗漏（dropped_busy / 失败 / 崩溃前未压）的残量。
-- 注意：pm_files 是普通表（非超表，超表是 pm_metrics），ALTER ADD COLUMN 直接生效。
ALTER TABLE pm_files ADD COLUMN IF NOT EXISTS raw_compressed boolean NOT NULL DEFAULT false;

-- 部分索引：只索引"尚未确认压缩"的行（稳态下接近空集），让 Sweeper 的
-- WHERE raw_compressed = false AND created_at < cutoff 查询代价 O(待压缩残量) 而非全表扫描。
CREATE INDEX IF NOT EXISTS idx_pm_files_uncompressed
    ON pm_files (created_at)
    WHERE raw_compressed = false;

-- +goose Down
DROP INDEX IF EXISTS idx_pm_files_uncompressed;
ALTER TABLE pm_files DROP COLUMN IF EXISTS raw_compressed;

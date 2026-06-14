-- +goose Up
-- issue #321 加固：mr_files 增加 raw_compressed 标记列，配合补偿扫描 Sweeper 保证
-- 每个 MR 原始 XML 最终都被 gzip 压缩回写 MinIO —— 即使入库后的内联异步压缩因并发满
-- 被丢弃（dropped_busy）、瞬时失败、或在压缩前进程崩溃，Sweeper 也会兜底重扫直至该行
-- raw_compressed=true。内联压缩成功后亦置真，使待扫描集稳态接近空集。
ALTER TABLE mr_files ADD COLUMN IF NOT EXISTS raw_compressed boolean NOT NULL DEFAULT false;

-- 部分索引：只索引"尚未确认压缩"的行（稳态下接近空集），让 Sweeper 的
-- WHERE raw_compressed = false AND created_at < cutoff 查询代价 O(待压缩残量) 而非全表扫描。
CREATE INDEX IF NOT EXISTS idx_mr_files_uncompressed
    ON mr_files (created_at)
    WHERE raw_compressed = false;

-- +goose Down
DROP INDEX IF EXISTS idx_mr_files_uncompressed;
ALTER TABLE mr_files DROP COLUMN IF EXISTS raw_compressed;

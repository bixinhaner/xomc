-- +goose Up
-- PM/MR 原始对象改由 worker 按数据库精确路径持续分批清理。

ALTER TABLE public.pm_files
    ADD COLUMN IF NOT EXISTS raw_deleted_at timestamptz,
    ADD COLUMN IF NOT EXISTS raw_delete_attempts integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS raw_delete_next_attempt_at timestamptz,
    ADD COLUMN IF NOT EXISTS raw_delete_last_error varchar(512);

ALTER TABLE public.mr_files
    ADD COLUMN IF NOT EXISTS raw_deleted_at timestamptz,
    ADD COLUMN IF NOT EXISTS raw_delete_attempts integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS raw_delete_next_attempt_at timestamptz,
    ADD COLUMN IF NOT EXISTS raw_delete_last_error varchar(512);

CREATE INDEX IF NOT EXISTS idx_pm_files_raw_cleanup
    ON public.pm_files (collect_time, id)
    WHERE raw_deleted_at IS NULL AND raw_delete_next_attempt_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pm_files_raw_cleanup_retry
    ON public.pm_files (raw_delete_next_attempt_at, collect_time, id)
    WHERE raw_deleted_at IS NULL AND raw_delete_next_attempt_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_mr_files_created
    ON public.mr_files (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mr_files_raw_cleanup
    ON public.mr_files (collect_time, id)
    WHERE raw_deleted_at IS NULL AND raw_delete_next_attempt_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mr_files_raw_cleanup_retry
    ON public.mr_files (raw_delete_next_attempt_at, collect_time, id)
    WHERE raw_deleted_at IS NULL AND raw_delete_next_attempt_at IS NOT NULL;

-- 依赖安全元数据回收的反查索引；避免在大超表上反复全表扫描。
CREATE INDEX IF NOT EXISTS idx_mr_records_file_time
    ON public.mr_records (file_id, "time" DESC);
CREATE INDEX IF NOT EXISTS idx_pm_anchors_source_file
    ON public.pm_measurement_anchors (source_file_id)
    WHERE source_file_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS public.idx_pm_anchors_source_file;
DROP INDEX IF EXISTS public.idx_mr_records_file_time;
DROP INDEX IF EXISTS public.idx_mr_files_raw_cleanup_retry;
DROP INDEX IF EXISTS public.idx_mr_files_raw_cleanup;
DROP INDEX IF EXISTS public.idx_mr_files_created;
DROP INDEX IF EXISTS public.idx_pm_files_raw_cleanup_retry;
DROP INDEX IF EXISTS public.idx_pm_files_raw_cleanup;

ALTER TABLE public.mr_files
    DROP COLUMN IF EXISTS raw_delete_last_error,
    DROP COLUMN IF EXISTS raw_delete_next_attempt_at,
    DROP COLUMN IF EXISTS raw_delete_attempts,
    DROP COLUMN IF EXISTS raw_deleted_at;

ALTER TABLE public.pm_files
    DROP COLUMN IF EXISTS raw_delete_last_error,
    DROP COLUMN IF EXISTS raw_delete_next_attempt_at,
    DROP COLUMN IF EXISTS raw_delete_attempts,
    DROP COLUMN IF EXISTS raw_deleted_at;

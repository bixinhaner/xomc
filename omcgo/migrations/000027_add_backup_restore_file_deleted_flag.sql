-- +goose Up
ALTER TABLE public.backup_restore_file
    ADD COLUMN IF NOT EXISTS is_deleted boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS deleted_at timestamp with time zone;

CREATE INDEX IF NOT EXISTS idx_backup_restore_file_active_fault_logs
    ON public.backup_restore_file USING btree (serial_number, update_time ASC, id ASC)
    WHERE is_deleted = false
      AND (object_path LIKE '%/fault/%' OR object_path LIKE 'fault/%');

CREATE INDEX IF NOT EXISTS idx_backup_restore_file_active_station_logs_retention
    ON public.backup_restore_file USING btree (update_time ASC, id ASC)
    WHERE is_deleted = false
      AND (object_path LIKE '%/running/%' OR object_path LIKE 'running/%'
        OR object_path LIKE '%/fault/%' OR object_path LIKE 'fault/%');

-- +goose Down
DROP INDEX IF EXISTS public.idx_backup_restore_file_active_station_logs_retention;
DROP INDEX IF EXISTS public.idx_backup_restore_file_active_fault_logs;
DROP INDEX IF EXISTS public.idx_backup_restore_file_active_logs;

ALTER TABLE public.backup_restore_file
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS is_deleted;

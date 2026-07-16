-- +goose Up
ALTER TABLE public.devices
    ADD COLUMN recycle_type varchar(16) NOT NULL DEFAULT '',
    ADD COLUMN recycle_executor varchar(128) NOT NULL DEFAULT '',
    ADD CONSTRAINT devices_recycle_type_check
        CHECK (recycle_type IN ('', 'manual', 'auto'));

UPDATE public.devices
SET recycle_type = CASE
        WHEN deleted_by = 'system' OR deleted_by LIKE 'system:%' THEN 'auto'
        ELSE 'manual'
    END,
    recycle_executor = COALESCE(deleted_by, '')
WHERE deleted_at IS NOT NULL;

COMMENT ON COLUMN public.devices.recycle_type IS '移入回收站方式：manual 或 auto';
COMMENT ON COLUMN public.devices.recycle_executor IS '实际执行软删除的用户或系统任务标识';

-- +goose Down
ALTER TABLE public.devices
    DROP CONSTRAINT IF EXISTS devices_recycle_type_check,
    DROP COLUMN IF EXISTS recycle_executor,
    DROP COLUMN IF EXISTS recycle_type;

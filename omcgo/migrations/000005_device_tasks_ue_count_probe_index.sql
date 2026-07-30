-- +goose Up

CREATE INDEX IF NOT EXISTS idx_device_tasks_open_method_description
    ON public.device_tasks (device_sn, method, description, created_at DESC)
    WHERE status IN ('pending', 'sent');

-- +goose Down

DROP INDEX IF EXISTS public.idx_device_tasks_open_method_description;

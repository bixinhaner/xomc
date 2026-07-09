-- +goose Up
-- +goose StatementBegin
ALTER TABLE public.mml_tasks
    ADD COLUMN IF NOT EXISTS execute_mode text DEFAULT 'common' NOT NULL,
    ADD COLUMN IF NOT EXISTS plan_items jsonb DEFAULT '[]'::jsonb NOT NULL;

COMMENT ON COLUMN public.mml_tasks.execute_mode IS 'MML task execution mode: common broadcasts commands to selected devices; device_bound uses plan_items as the execution source.';
COMMENT ON COLUMN public.mml_tasks.plan_items IS 'Device-bound execution plan rows. Each row binds source line, device SN, per-device order, raw line, and normalized command JSON.';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'mml_tasks_execute_mode_check'
    ) THEN
        ALTER TABLE public.mml_tasks
            ADD CONSTRAINT mml_tasks_execute_mode_check
            CHECK (execute_mode IN ('common', 'device_bound'));
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_mml_tasks_plan_items_gin
    ON public.mml_tasks USING gin (plan_items jsonb_path_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS public.idx_mml_tasks_plan_items_gin;

ALTER TABLE public.mml_tasks
    DROP CONSTRAINT IF EXISTS mml_tasks_execute_mode_check,
    DROP COLUMN IF EXISTS plan_items,
    DROP COLUMN IF EXISTS execute_mode;
-- +goose StatementEnd

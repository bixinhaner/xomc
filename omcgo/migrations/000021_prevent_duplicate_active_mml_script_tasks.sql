-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS uq_mml_tasks_active_root_script
    ON public.mml_tasks (script_id)
    WHERE script_id IS NOT NULL
      AND parent_task_id IS NULL
      AND status IN ('pending', 'running', 'paused');

COMMENT ON INDEX public.uq_mml_tasks_active_root_script IS
    'At most one unfinished top-level MML script task per script. Periodic child runs keep parent_task_id and are not constrained here.';

-- +goose Down
DROP INDEX IF EXISTS public.uq_mml_tasks_active_root_script;

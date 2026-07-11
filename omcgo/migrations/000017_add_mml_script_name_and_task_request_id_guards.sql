-- +goose Up
-- Keep the earliest script name and rename historical duplicates before adding
-- the invariant. New writes are guarded by service pre-checks plus this index.
WITH ranked AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY COALESCE(creator, ''), lower(btrim(script_name))
               ORDER BY created_at, id
           ) AS rn
    FROM public.mml_scripts
    WHERE btrim(script_name) <> ''
)
UPDATE public.mml_scripts AS s
SET script_name = left(btrim(s.script_name), 180) || '_重复_' || substr(s.id::text, 1, 8),
    updated_at = now()
FROM ranked AS r
WHERE s.id = r.id
  AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_mml_scripts_creator_name_ci
    ON public.mml_scripts (COALESCE(creator, ''), lower(btrim(script_name)));

ALTER TABLE public.mml_tasks
    ADD COLUMN IF NOT EXISTS request_id text;

CREATE UNIQUE INDEX IF NOT EXISTS uq_mml_tasks_creator_request_id
    ON public.mml_tasks (COALESCE(creator, ''), request_id)
    WHERE request_id IS NOT NULL AND btrim(request_id) <> '';

COMMENT ON COLUMN public.mml_tasks.request_id IS 'Client-generated idempotency key for task creation requests.';

-- +goose Down
DROP INDEX IF EXISTS public.uq_mml_tasks_creator_request_id;

ALTER TABLE public.mml_tasks
    DROP COLUMN IF EXISTS request_id;

DROP INDEX IF EXISTS public.uq_mml_scripts_creator_name_ci;

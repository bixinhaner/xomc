-- +goose Up
-- #49: Existing databases already recorded migration 000001, so changing the
-- consolidated baseline alone does not update their source_type constraint.
ALTER TABLE public.pm_kpi_export_tasks
    DROP CONSTRAINT IF EXISTS pm_kpi_export_tasks_source_type_chk;

ALTER TABLE public.pm_kpi_export_tasks
    ADD CONSTRAINT pm_kpi_export_tasks_source_type_chk
    CHECK (source_type = ANY (ARRAY['dashboard'::text, 'kpi_query'::text, 'adhoc'::text]));

-- +goose Down
-- Non-destructive by design: kpi_query rows may already exist after Up, so
-- restoring the old constraint would make rollback fail or require data loss.
SELECT 1;

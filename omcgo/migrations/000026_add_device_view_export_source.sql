-- +goose Up
ALTER TABLE public.pm_kpi_export_tasks
    DROP CONSTRAINT IF EXISTS pm_kpi_export_tasks_source_type_chk,
    ADD CONSTRAINT pm_kpi_export_tasks_source_type_chk
        CHECK (source_type = ANY (ARRAY['dashboard'::text, 'device_view'::text, 'kpi_query'::text, 'adhoc'::text]));

-- +goose Down
ALTER TABLE public.pm_kpi_export_tasks
    DROP CONSTRAINT IF EXISTS pm_kpi_export_tasks_source_type_chk,
    ADD CONSTRAINT pm_kpi_export_tasks_source_type_chk
        CHECK (source_type = ANY (ARRAY['dashboard'::text, 'kpi_query'::text, 'adhoc'::text]));

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

CREATE INDEX IF NOT EXISTS idx_pm_aggregation_results_dashboard_network
    ON public.pm_aggregation_results (
        task_id,
        granularity,
        technology,
        metric_path,
        window_start DESC,
        created_at DESC
    )
    WHERE dimension = 'network';

-- +goose Down
DROP INDEX IF EXISTS public.idx_pm_aggregation_results_dashboard_network;

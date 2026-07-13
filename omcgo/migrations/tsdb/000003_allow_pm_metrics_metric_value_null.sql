-- 000003_allow_pm_metrics_metric_value_null.sql
-- #899: 15min raw PM missing supported-but-unreported metrics are stored as NULL.

-- +goose Up
ALTER TABLE public.pm_metrics
    ALTER COLUMN metric_value DROP NOT NULL;

-- +goose Down
ALTER TABLE public.pm_metrics
    ALTER COLUMN metric_value SET NOT NULL;

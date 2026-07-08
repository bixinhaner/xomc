-- 000005_allow_pm_adhoc_metric_value_null.sql
-- #902: adhoc aggregation results preserve supported-but-missing metrics as NULL.

-- +goose Up
ALTER TABLE public.pm_adhoc_aggregation_results
    ALTER COLUMN metric_value DROP NOT NULL;

-- +goose Down
ALTER TABLE public.pm_adhoc_aggregation_results
    ALTER COLUMN metric_value SET NOT NULL;

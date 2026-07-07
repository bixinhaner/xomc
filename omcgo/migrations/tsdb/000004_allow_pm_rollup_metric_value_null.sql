-- 000004_allow_pm_rollup_metric_value_null.sql
-- #901: hourly/daily/weekly/monthly PM rollups preserve missing metric semantics as NULL.

-- +goose Up
ALTER TABLE public.pm_metrics_hourly
    ALTER COLUMN metric_value DROP NOT NULL;
ALTER TABLE public.pm_metrics_daily
    ALTER COLUMN metric_value DROP NOT NULL;
ALTER TABLE public.pm_metrics_weekly
    ALTER COLUMN metric_value DROP NOT NULL;
ALTER TABLE public.pm_metrics_monthly
    ALTER COLUMN metric_value DROP NOT NULL;

ALTER TABLE public.pm_group_metrics_hourly
    ALTER COLUMN metric_value DROP NOT NULL;
ALTER TABLE public.pm_group_metrics_daily
    ALTER COLUMN metric_value DROP NOT NULL;
ALTER TABLE public.pm_group_metrics_weekly
    ALTER COLUMN metric_value DROP NOT NULL;
ALTER TABLE public.pm_group_metrics_monthly
    ALTER COLUMN metric_value DROP NOT NULL;

-- +goose Down
ALTER TABLE public.pm_group_metrics_monthly
    ALTER COLUMN metric_value SET NOT NULL;
ALTER TABLE public.pm_group_metrics_weekly
    ALTER COLUMN metric_value SET NOT NULL;
ALTER TABLE public.pm_group_metrics_daily
    ALTER COLUMN metric_value SET NOT NULL;
ALTER TABLE public.pm_group_metrics_hourly
    ALTER COLUMN metric_value SET NOT NULL;

ALTER TABLE public.pm_metrics_monthly
    ALTER COLUMN metric_value SET NOT NULL;
ALTER TABLE public.pm_metrics_weekly
    ALTER COLUMN metric_value SET NOT NULL;
ALTER TABLE public.pm_metrics_daily
    ALTER COLUMN metric_value SET NOT NULL;
ALTER TABLE public.pm_metrics_hourly
    ALTER COLUMN metric_value SET NOT NULL;

-- +goose Up

ALTER TABLE public.pm_tasks
    ADD COLUMN IF NOT EXISTS planned_end_at timestamp with time zone;

COMMENT ON COLUMN public.pm_tasks.planned_end_at IS
    'PM adhoc 自建 continuous 任务计划结束时间；NULL 表示老任务或内置任务不设置计划结束。';

ALTER TABLE public.pm_aggregation_tasks
    ADD COLUMN IF NOT EXISTS planned_end_at timestamp with time zone;

COMMENT ON COLUMN public.pm_aggregation_tasks.planned_end_at IS
    '流式聚合任务计划结束时间；NULL 表示不设置计划结束。';

-- +goose Down

ALTER TABLE public.pm_aggregation_tasks
    DROP COLUMN IF EXISTS planned_end_at;

ALTER TABLE public.pm_tasks
    DROP COLUMN IF EXISTS planned_end_at;

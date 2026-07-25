-- +goose Up

-- PM 聚合正式切换：旧任务、运行记录和完成水位不迁移。
DROP TRIGGER IF EXISTS trg_ops_pause_pm_natural_aggregation ON public.async_jobs;
DROP TRIGGER IF EXISTS trg_ops_pause_pm_adhoc_aggregation ON public.pm_tasks;
DROP FUNCTION IF EXISTS public.ops_pause_pm_natural_aggregation();
DROP FUNCTION IF EXISTS public.ops_pause_pm_adhoc_aggregation();

DELETE FROM public.async_jobs
 WHERE job_type IN (
    'pm_aggregate_hourly', 'pm_aggregate_daily', 'pm_aggregate_weekly', 'pm_aggregate_monthly',
    'pm_aggregate_group_hourly', 'pm_aggregate_group_daily',
    'pm_aggregate_group_weekly', 'pm_aggregate_group_monthly'
 );
DELETE FROM public.async_jobs_cron_state
 WHERE job_type IN (
    'pm_aggregate_hourly', 'pm_aggregate_daily', 'pm_aggregate_weekly', 'pm_aggregate_monthly',
    'pm_aggregate_group_hourly', 'pm_aggregate_group_daily',
    'pm_aggregate_group_weekly', 'pm_aggregate_group_monthly'
 );

TRUNCATE TABLE public.pm_adhoc_task_runs;
TRUNCATE TABLE public.pm_tasks;
DROP TABLE IF EXISTS public.pm_completion_watermarks;

CREATE TABLE public.pm_aggregation_tasks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(200) NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    visibility varchar(16) NOT NULL DEFAULT 'private',
    creator varchar(100) NOT NULL,
    current_version_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT chk_pm_aggregation_tasks_visibility
        CHECK (visibility IN ('private', 'public'))
);

CREATE TABLE public.pm_aggregation_task_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id uuid NOT NULL REFERENCES public.pm_aggregation_tasks(id) ON DELETE CASCADE,
    version_no integer NOT NULL,
    enabled boolean NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    technology varchar(16),
    dimension varchar(32) NOT NULL,
    granularities text[] NOT NULL,
    object_ldns text[] NOT NULL DEFAULT '{}',
    created_by varchar(100) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_pm_aggregation_task_versions UNIQUE (task_id, version_no),
    CONSTRAINT chk_pm_aggregation_task_versions_time
        CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT chk_pm_aggregation_task_versions_technology
        CHECK (technology IS NULL OR technology IN ('lte', 'nr', 'gsm')),
    CONSTRAINT chk_pm_aggregation_task_versions_dimension
        CHECK (dimension IN ('device', 'aggregate_group', 'device_group', 'product', 'band', 'network')),
    CONSTRAINT chk_pm_aggregation_task_versions_granularities
        CHECK (granularities <@ ARRAY['hourly', 'daily', 'weekly', 'monthly']::text[])
);

ALTER TABLE public.pm_aggregation_tasks
    ADD CONSTRAINT fk_pm_aggregation_tasks_current_version
    FOREIGN KEY (current_version_id)
    REFERENCES public.pm_aggregation_task_versions(id)
    ON DELETE SET NULL
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE public.pm_aggregation_version_metrics (
    task_version_id uuid NOT NULL
        REFERENCES public.pm_aggregation_task_versions(id) ON DELETE CASCADE,
    metric_id text NOT NULL,
    metric_path text NOT NULL,
    metric_type varchar(16) NOT NULL,
    aggregation_op varchar(8) NOT NULL,
    PRIMARY KEY (task_version_id, metric_id),
    CONSTRAINT chk_pm_aggregation_version_metrics_type
        CHECK (metric_type IN ('counter', 'kpi')),
    CONSTRAINT chk_pm_aggregation_version_metrics_op
        CHECK (aggregation_op IN ('sum', 'avg', 'min', 'max'))
);

CREATE TABLE public.pm_aggregation_version_members (
    task_version_id uuid NOT NULL
        REFERENCES public.pm_aggregation_task_versions(id) ON DELETE CASCADE,
    device_id uuid NOT NULL,
    device_sn varchar(128) NOT NULL,
    dimension_key text NOT NULL,
    dimension_name text NOT NULL DEFAULT '',
    object_ldn text NOT NULL DEFAULT '',
    PRIMARY KEY (task_version_id, device_id, dimension_key, object_ldn)
);

CREATE INDEX idx_pm_aggregation_tasks_active
    ON public.pm_aggregation_tasks (enabled, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_pm_aggregation_tasks_creator
    ON public.pm_aggregation_tasks (creator, visibility, updated_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_pm_aggregation_versions_effective
    ON public.pm_aggregation_task_versions (effective_from, effective_to);
CREATE INDEX idx_pm_aggregation_version_members_device
    ON public.pm_aggregation_version_members (device_id, task_version_id);
CREATE INDEX idx_pm_aggregation_version_metrics_path
    ON public.pm_aggregation_version_metrics (metric_path, task_version_id);

CREATE TRIGGER trigger_pm_aggregation_tasks_updated_at
    BEFORE UPDATE ON public.pm_aggregation_tasks
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- +goose Down

DROP TRIGGER IF EXISTS trigger_pm_aggregation_tasks_updated_at
    ON public.pm_aggregation_tasks;
DROP TABLE IF EXISTS public.pm_aggregation_version_members;
DROP TABLE IF EXISTS public.pm_aggregation_version_metrics;
ALTER TABLE IF EXISTS public.pm_aggregation_tasks
    DROP CONSTRAINT IF EXISTS fk_pm_aggregation_tasks_current_version;
DROP TABLE IF EXISTS public.pm_aggregation_task_versions;
DROP TABLE IF EXISTS public.pm_aggregation_tasks;

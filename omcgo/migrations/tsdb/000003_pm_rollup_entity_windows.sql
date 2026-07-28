-- +goose Up
-- 事件驱动逐级汇聚不兼容旧共享窗口：旧活动状态、上卷消息和结果不迁移。
TRUNCATE TABLE
    public.pm_aggregation_rollup_outbox,
    public.pm_aggregation_counter_rollups,
    public.pm_aggregation_windows,
    public.pm_aggregation_results;

ALTER TABLE public.pm_aggregation_counter_rollups
    ADD COLUMN IF NOT EXISTS entity_key text;
ALTER TABLE public.pm_aggregation_counter_rollups
    ALTER COLUMN entity_key SET NOT NULL;

DROP INDEX IF EXISTS public.idx_pm_aggregation_counter_rollups_recovery;
CREATE INDEX idx_pm_aggregation_counter_rollups_recovery
    ON public.pm_aggregation_counter_rollups (
        task_version_id, entity_key, granularity, window_start, chunk_index
    );

ALTER TABLE public.pm_aggregation_windows
    ADD COLUMN IF NOT EXISTS entity_key text;
ALTER TABLE public.pm_aggregation_windows
    ALTER COLUMN entity_key SET NOT NULL;
ALTER TABLE public.pm_aggregation_windows
    DROP CONSTRAINT IF EXISTS pm_aggregation_windows_pkey;
ALTER TABLE public.pm_aggregation_windows
    ADD CONSTRAINT pm_aggregation_windows_pkey
    PRIMARY KEY (task_version_id, entity_key, granularity, window_start);

-- +goose Down
TRUNCATE TABLE
    public.pm_aggregation_rollup_outbox,
    public.pm_aggregation_counter_rollups,
    public.pm_aggregation_windows,
    public.pm_aggregation_results;

ALTER TABLE public.pm_aggregation_windows
    DROP CONSTRAINT IF EXISTS pm_aggregation_windows_pkey;
ALTER TABLE public.pm_aggregation_windows
    ADD CONSTRAINT pm_aggregation_windows_pkey
    PRIMARY KEY (task_version_id, granularity, window_start);
ALTER TABLE public.pm_aggregation_windows
    DROP COLUMN IF EXISTS entity_key;

DROP INDEX IF EXISTS public.idx_pm_aggregation_counter_rollups_recovery;
CREATE INDEX idx_pm_aggregation_counter_rollups_recovery
    ON public.pm_aggregation_counter_rollups (
        task_version_id, granularity, window_start, chunk_index
    );
ALTER TABLE public.pm_aggregation_counter_rollups
    DROP COLUMN IF EXISTS entity_key;

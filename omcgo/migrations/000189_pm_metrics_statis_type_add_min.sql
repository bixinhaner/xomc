-- T-0164 G6 收尾 BUG-D: PM metrics statis_type CHECK 约束扩 'min'
--
-- 现象：真机 BLQ 21:00 PM 文件 BatchInsert 整批失败：
--   ERROR: new row for relation "_hyper_9_9_chunk" violates check constraint
--   "pm_metrics_statis_type_check" (SQLSTATE 23514)
--
-- 根因：perf_indicators_enb.statis_type 元数据中存在 'min' 值（1 条 counter），
-- 但 G3 设计 + 000188 CHECK 只允许 ('sum','avg','max','pct')。BUG-A 修复让
-- collector 把 indicator 的 statis_type 透传到 pm_metrics 后，'min' 撞约束。
--
-- 修复：把 'min' 加入 9 张 metrics 表 + 1 张 adhoc results 表的 CHECK 约束。
-- 配套 aggregator.go / device_group.go SQL 加 `WHEN 'min' THEN MIN(m.metric_value)`
-- 路由 + WHERE 子句加 'min'。
--
-- 同步修了 000188 源文件（让重新部署时直接对），本 migration 修补已部署 DB。

-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'pm_metrics',
        'pm_metrics_hourly',
        'pm_metrics_daily',
        'pm_metrics_weekly',
        'pm_metrics_monthly',
        'pm_group_metrics_hourly',
        'pm_group_metrics_daily',
        'pm_group_metrics_weekly',
        'pm_group_metrics_monthly',
        'pm_adhoc_aggregation_results'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I',
                       t, t || '_statis_type_check');
        EXECUTE format($f$ALTER TABLE %I ADD CONSTRAINT %I
                       CHECK (statis_type IS NULL OR statis_type IN
                              ('sum','avg','max','min','pct'))$f$,
                       t, t || '_statis_type_check');
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'pm_metrics',
        'pm_metrics_hourly',
        'pm_metrics_daily',
        'pm_metrics_weekly',
        'pm_metrics_monthly',
        'pm_group_metrics_hourly',
        'pm_group_metrics_daily',
        'pm_group_metrics_weekly',
        'pm_group_metrics_monthly',
        'pm_adhoc_aggregation_results'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I',
                       t, t || '_statis_type_check');
        EXECUTE format($f$ALTER TABLE %I ADD CONSTRAINT %I
                       CHECK (statis_type IS NULL OR statis_type IN
                              ('sum','avg','max','pct'))$f$,
                       t, t || '_statis_type_check');
    END LOOP;
END $$;
-- +goose StatementEnd

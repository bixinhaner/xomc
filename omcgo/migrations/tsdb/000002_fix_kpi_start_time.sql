-- +goose Up
-- =====================================================================================
-- 历史回填：修正 KPI 行 start_time == end_time（#199 / #208 打点起止相同）
--
-- 背景：metrics.MetricFromKPIValue 历史上把 KPI 行的 start_time / end_time / time 都写成
-- 同一个窗口止点（v.Time），致 pm_metrics.start_time == end_time，前端悬浮框「开始 / 结束」
-- 显示同一时刻。前向逻辑已在 copy_ingest.go 修为 start_time = end_time - 粒度窗口；本迁移把
-- 存量已落库的 KPI 行一并纠正，让历史曲线悬浮也立即正确（否则测试翻历史数据仍见旧症状）。
--
-- 范围与口径：
--   - 仅命中 metric_type = 'kpi' 且 start_time = end_time 的「坏行」——counter 行起止本就正确，
--     不触碰；rollup 表里由 recomputeKPIs 从 counter 继承 start/end 的 KPI 行起止本就正确
--     （start ≠ end），同样被 start_time = end_time 条件自然排除。
--   - 窗口长度按各表自身 granularity 列推导（15min/hourly/daily/weekly/monthly），不硬编码 15min，
--     保证 rollup 表若存在坏 KPI 行也按正确粒度回填。
--   - 幂等：回填后这些行 start_time < end_time，再次执行 WHERE start_time = end_time 命中零行。
--   - start_time 不在任一表的唯一索引/主键键集内（键尾用 end_time，不含 start_time），改它不会
--     引发自然键碰撞。
--
-- granularity → interval 映射用 CASE，未知粒度兜底 15min（与入库层默认窗口一致）。
-- =====================================================================================

-- +goose StatementBegin
DO $$
DECLARE
    tbl text;
    tables text[] := ARRAY[
        'pm_metrics',
        'pm_metrics_hourly',
        'pm_metrics_daily',
        'pm_metrics_weekly',
        'pm_metrics_monthly'
    ];
BEGIN
    FOREACH tbl IN ARRAY tables LOOP
        EXECUTE format(
            'UPDATE public.%I
                SET start_time = end_time - (CASE granularity
                    WHEN ''15min''   THEN INTERVAL ''15 minutes''
                    WHEN ''hourly''  THEN INTERVAL ''1 hour''
                    WHEN ''daily''   THEN INTERVAL ''1 day''
                    WHEN ''weekly''  THEN INTERVAL ''7 days''
                    WHEN ''monthly'' THEN INTERVAL ''1 month''
                    ELSE INTERVAL ''15 minutes''
                END)
              WHERE metric_type = ''kpi''
                AND start_time = end_time',
            tbl
        );
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- 不可逆：回填前的 start_time（== end_time）是错误数据，无单独保留，无法精确还原。
-- 回退本迁移仅删 goose 版本记录，不改回数据（改回反而恢复 bug）。
SELECT 1;

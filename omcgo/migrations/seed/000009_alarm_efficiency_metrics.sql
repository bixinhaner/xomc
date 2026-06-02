-- +goose Up
-- 创建告警效率指标物化视图
-- 用于统计告警处理的效率指标：MTTA（平均确认时间）、MTTR（平均解决时间）、确认率、清除率
--
-- 数据范围说明：
-- - 只统计近30天内已清除的告警（WHERE cleared_at > NOW() - INTERVAL '30 days'）
-- - 不包含未清除的活动告警，因为无法计算 MTTR（解决时间）
-- - 不包含30天前的告警，确保指标反映当前的处理效率
CREATE MATERIALIZED VIEW IF NOT EXISTS alarm_efficiency_metrics AS
SELECT
    severity,
    -- 已确认告警数量
    COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) as acknowledged_count,
    -- 已清除告警数量
    COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) as cleared_count,
    -- 总告警数量
    COUNT(*) as total_count,
    -- 平均确认时间（分钟）
    COALESCE(AVG(EXTRACT(EPOCH FROM (acknowledged_at - raised_at)) / 60)
        FILTER (WHERE acknowledged_at IS NOT NULL), 0) as avg_acknowledge_minutes,
    -- 平均解决时间（分钟）
    COALESCE(AVG(EXTRACT(EPOCH FROM (cleared_at - raised_at)) / 60)
        FILTER (WHERE cleared_at IS NOT NULL), 0) as avg_resolve_minutes,
    -- 确认率（百分比）
    ROUND(100.0 * COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) /
          NULLIF(COUNT(*), 0), 2) as acknowledge_rate,
    -- 清除率（百分比）
    ROUND(100.0 * COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) /
          NULLIF(COUNT(*), 0), 2) as clear_rate
FROM alarms_history
WHERE cleared_at > NOW() - INTERVAL '30 days'
GROUP BY severity;

-- 创建索引以优化查询性能
CREATE INDEX IF NOT EXISTS idx_alarms_history_severity_raised_at
    ON alarms_history(severity, raised_at DESC);

CREATE INDEX IF NOT EXISTS idx_alarms_history_acknowledged_at
    ON alarms_history(acknowledged_at) WHERE acknowledged_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_alarms_history_cleared_at
    ON alarms_history(cleared_at) WHERE cleared_at IS NOT NULL;

-- 创建物化视图刷新函数
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_alarm_efficiency_metrics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY alarm_efficiency_metrics;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- 添加注释
COMMENT ON MATERIALIZED VIEW alarm_efficiency_metrics IS
    '告警处理效率指标物化视图，包含MTTA、MTTR、确认率、清除率等指标';

COMMENT ON FUNCTION refresh_alarm_efficiency_metrics() IS
    '刷新告警效率指标物化视图';

-- +goose Down
-- 删除函数
DROP FUNCTION IF EXISTS refresh_alarm_efficiency_metrics();

-- 删除物化视图
DROP MATERIALIZED VIEW IF EXISTS alarm_efficiency_metrics;

-- 删除索引
DROP INDEX IF EXISTS idx_alarms_history_severity_raised_at;
DROP INDEX IF EXISTS idx_alarms_history_acknowledged_at;
DROP INDEX IF EXISTS idx_alarms_history_cleared_at;

-- +goose Up
-- 修正 MTTR（平均解决时间）负值问题（issue #221）
--
-- 根因：物化视图 alarm_efficiency_metrics 的 avg_resolve_minutes 只过滤了
-- cleared_at IS NOT NULL，未排除"清除时刻早于产生时刻"（cleared_at < raised_at）
-- 的时间倒挂脏数据。倒挂记录算出负的解决时长，被计入平均后把 MTTR 拉成负值。
--
-- 修法：在 MTTR 的 AVG FILTER 里追加 AND cleared_at >= raised_at，排除倒挂记录。
--
-- 物化视图不支持 CREATE OR REPLACE，必须 DROP + CREATE 重定义；原地改
-- seed/000009 对已存在的库不重新生效，故新增本前向迁移。
-- 放在 seed 序列（与原始 000009 同序列），保证视图已存在后再重定义。

-- 先删除依赖该视图的刷新函数（函数体引用视图名）
DROP FUNCTION IF EXISTS refresh_alarm_efficiency_metrics();

DROP MATERIALIZED VIEW IF EXISTS alarm_efficiency_metrics;

CREATE MATERIALIZED VIEW alarm_efficiency_metrics AS
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
    -- 平均解决时间（分钟）；排除时间倒挂脏数据（cleared_at < raised_at）避免负值
    COALESCE(AVG(EXTRACT(EPOCH FROM (cleared_at - raised_at)) / 60)
        FILTER (WHERE cleared_at IS NOT NULL AND cleared_at >= raised_at), 0) as avg_resolve_minutes,
    -- 确认率（百分比）
    ROUND(100.0 * COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) /
          NULLIF(COUNT(*), 0), 2) as acknowledge_rate,
    -- 清除率（百分比）
    ROUND(100.0 * COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) /
          NULLIF(COUNT(*), 0), 2) as clear_rate
FROM alarms_history
WHERE cleared_at > NOW() - INTERVAL '30 days'
GROUP BY severity;

-- 重建刷新函数（DROP 视图前已删函数，需重新创建）
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_alarm_efficiency_metrics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY alarm_efficiency_metrics;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

COMMENT ON MATERIALIZED VIEW alarm_efficiency_metrics IS
    '告警处理效率指标物化视图，包含MTTA、MTTR、确认率、清除率等指标（MTTR 已排除时间倒挂脏数据）';

COMMENT ON FUNCTION refresh_alarm_efficiency_metrics() IS
    '刷新告警效率指标物化视图';

-- CREATE MATERIALIZED VIEW 默认 WITH DATA 已填充；显式刷新一次保证最新
REFRESH MATERIALIZED VIEW alarm_efficiency_metrics;

-- +goose Down
-- 回滚为不带倒挂过滤的旧定义（与 seed/000009 一致）
DROP FUNCTION IF EXISTS refresh_alarm_efficiency_metrics();
DROP MATERIALIZED VIEW IF EXISTS alarm_efficiency_metrics;

CREATE MATERIALIZED VIEW alarm_efficiency_metrics AS
SELECT
    severity,
    COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) as acknowledged_count,
    COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) as cleared_count,
    COUNT(*) as total_count,
    COALESCE(AVG(EXTRACT(EPOCH FROM (acknowledged_at - raised_at)) / 60)
        FILTER (WHERE acknowledged_at IS NOT NULL), 0) as avg_acknowledge_minutes,
    COALESCE(AVG(EXTRACT(EPOCH FROM (cleared_at - raised_at)) / 60)
        FILTER (WHERE cleared_at IS NOT NULL), 0) as avg_resolve_minutes,
    ROUND(100.0 * COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) /
          NULLIF(COUNT(*), 0), 2) as acknowledge_rate,
    ROUND(100.0 * COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) /
          NULLIF(COUNT(*), 0), 2) as clear_rate
FROM alarms_history
WHERE cleared_at > NOW() - INTERVAL '30 days'
GROUP BY severity;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refresh_alarm_efficiency_metrics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY alarm_efficiency_metrics;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

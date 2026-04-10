-- ============================================================================
-- sqlc 查询文件示例: PM 计数器查询
-- ============================================================================
-- 这些是 sqlc 处理的查询语句
--  sqlc:build
-- ============================================================================

-- name: GetCounterStatsByTimeRange :many
-- 查询指定时间范围内的计数器统计信息
-- 使用 TimescaleDB time_bucket 函数
SELECT 
    time_bucket($1::INTERVAL, time) AS bucket,
    device_id,
    counter_name,
    AVG(counter_value) AS avg_value,
    MAX(counter_value) AS max_value,
    MIN(counter_value) AS min_value,
    COUNT(*) AS sample_count
FROM pm_counters
WHERE time >= $2 
  AND time < $3
  AND device_id = $4
GROUP BY bucket, device_id, counter_name
ORDER BY bucket DESC;

-- name: GetLatestCounterValue :one
-- 获取设备的最新计数器值
SELECT 
    time,
    device_id,
    counter_name,
    counter_value,
    metadata
FROM pm_counters
WHERE device_id = $1 
  AND counter_name = $2
ORDER BY time DESC
LIMIT 1;

-- name: BatchInsertCounters :copyfrom
-- 批量插入计数器数据 (使用 COPY FROM 高效插入)
INSERT INTO pm_counters (
    time,
    device_id,
    counter_name,
    counter_value,
    metadata
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: UpdateCounterMetadata :exec
-- 更新计数器元数据 (JSONB 操作)
UPDATE pm_counters
SET metadata = jsonb_set(metadata, $4::TEXT[], $5::JSONB)
WHERE device_id = $1
  AND counter_name = $2
  AND time = $3;

-- name: GetDevicePMStats :one
-- 获取设备的 PM 统计摘要
SELECT 
    COUNT(DISTINCT counter_name) AS total_counters,
    MIN(time) AS first_record,
    MAX(time) AS last_record,
    COUNT(*) AS total_records
FROM pm_counters
WHERE device_id = $1;

-- name: DeleteOldCounters :exec
-- 删除指定时间之前的计数器数据
-- 注意: 这是管理操作,通常在维护窗口执行
DELETE FROM pm_counters
WHERE time < $1;

-- name: GetCounterTrends :many
-- 获取计数器趋势 (连续聚合查询示例)
SELECT 
    time_bucket('1 hour', time) AS hour_bucket,
    counter_name,
    AVG(counter_value) AS hourly_avg,
    LAG(AVG(counter_value)) OVER (
        PARTITION BY counter_name 
        ORDER BY time_bucket('1 hour', time)
    ) AS prev_hour_avg
FROM pm_counters
WHERE device_id = $1
  AND time >= NOW() - INTERVAL '24 hours'
GROUP BY hour_bucket, counter_name
ORDER BY hour_bucket DESC, counter_name;

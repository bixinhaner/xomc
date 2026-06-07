-- +goose Up
-- 一次性清理"假在线"脏数据：is_online=true 但从未上报(last_inform_at IS NULL)
-- 或超阈值未上报的设备，统一翻为离线。
--
-- 背景：历史种子/测试数据把大量设备 is_online 置 true 却无对应 last_inform_at，
-- 而离线对账器（DeviceStatusReconciler.FindStaleDevicesAdaptive）此前只纠正
-- lifecycle_state='commissioned' 且 last_inform_at 非空的设备，导致这些"僵尸在线"
-- 永不离线（现网 4011 假在线，真实仅 2 台）。
--
-- 阈值与对账器一致：max(2×inform_interval, 600s)。幂等：已离线的行不受影响；
-- 此后由对账器（每 5 分钟一轮，已去掉 commissioned 限制）持续维护。
UPDATE devices
SET is_online           = false,
    last_offline_reason = 'heartbeat_timeout',
    updated_at          = NOW()
WHERE is_online = true
  AND (
    last_inform_at IS NULL
    OR last_inform_at < NOW() - (GREATEST(COALESCE(inform_interval, 300) * 2, 600) * INTERVAL '1 second')
  );

-- +goose Down
-- 数据状态纠正，无法（也无意义）恢复历史 is_online 值；保留空 Down 满足 goose 格式。
SELECT 1;

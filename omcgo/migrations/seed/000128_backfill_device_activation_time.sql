-- +goose Up
-- 回填 device_info.first_online_time（激活时间）。
--
-- 背景：op_state（激活状态）口径由"从 devices.status 派生"改为"由
-- device_info.first_online_time 派生"——激活是一次性持久事实，与在线/离线解耦。
-- 改口径后，存量"已激活"设备（status ∈ active/offline/maintenance）若历史上
-- 未写过 first_online_time，会被误判为"未激活"。此处一次性回填，避免回归。
--
-- 取值：优先 last_online_time（设备最近一次离线→在线时刻），无则退回设备
-- created_at（建档时刻）。两者都是对真实激活时刻的合理近似。
UPDATE device_info di
SET first_online_time = COALESCE(di.last_online_time, d.created_at)
FROM devices d
WHERE di.device_id = d.id
  AND di.first_online_time IS NULL
  AND d.status IN ('active', 'offline', 'maintenance');

-- +goose Down
-- 无回滚：first_online_time 是一次性事实，回填值与正常写入值无法区分，
-- 强行清空会误伤正常数据。本迁移不可逆。
